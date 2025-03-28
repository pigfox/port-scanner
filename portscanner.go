package main

import (
	"bufio"
	"compress/gzip"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var brevo Brevo
var email Email

type ScanResult struct {
	IP    string
	Port  int
	Open  bool
	Error error
}

func scanPort(ip string, port int, timeout time.Duration, limiter chan struct{}) ScanResult {
	limiter <- struct{}{}
	defer func() { <-limiter }()

	address := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", address, timeout)

	result := ScanResult{
		IP:    ip,
		Port:  port,
		Error: err,
	}

	if err == nil {
		result.Open = true
		conn.Close()
	}

	return result
}

func scanChunk(startIPNum, endIPNum uint32, ports []int, timeout time.Duration, maxConcurrent int, resultChan chan<- ScanResult) {
	var wg sync.WaitGroup
	limiter := make(chan struct{}, maxConcurrent)

	fmt.Printf("Scanning chunk from %s to %s\n", uint32ToIP(startIPNum), uint32ToIP(endIPNum))

	for ipNum := startIPNum; ipNum <= endIPNum; ipNum++ {
		ip := uint32ToIP(ipNum)
		for _, port := range ports {
			wg.Add(1)
			go func(ip string, port int) {
				defer wg.Done()
				resultChan <- scanPort(ip, port, timeout, limiter)
			}(ip, port)
		}
	}

	wg.Wait()
}

func saveCheckpoint(checkpointFile, lastIP string) error {
	file, err := os.Create(checkpointFile)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(lastIP + "\n")
	return err
}

func loadCheckpoint(checkpointFile string) (string, error) {
	file, err := os.Open(checkpointFile)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	return "", nil
}

func scanRange(startIP, endIP string, ports []int, timeout time.Duration, maxConcurrent, chunkSize int, parallelChunks bool, outputFile, checkpointFile string, compress bool) error {
	start := ipToUint32(startIP)
	end := ipToUint32(endIP)
	totalIPs := uint64(end) - uint64(start) + 1 // Use uint64 to avoid overflow
	if totalIPs == 0 {                          // Only 0 if start > end after conversion
		return fmt.Errorf("invalid IP range: start %s is greater than end %s", startIP, endIP)
	}
	fmt.Printf("Total IPs to scan: %d\n", totalIPs)

	// Load checkpoint
	resumeIP, err := loadCheckpoint(checkpointFile)
	if err != nil {
		return fmt.Errorf("failed to load checkpoint: %v", err)
	}
	if resumeIP != "" {
		resume := ipToUint32(resumeIP)
		if resume >= start && resume <= end {
			start = resume + 1
			fmt.Printf("Resuming from %s\n", resumeIP)
		}
	}

	// Recalculate totalIPs after resume
	totalIPs = uint64(end) - uint64(start) + 1
	if totalIPs == 0 {
		fmt.Printf("Scan complete: resumed beyond end IP %s\n", endIP)
		return nil
	}

	// Open output file
	file, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %v", err)
	}
	defer file.Close()

	var writer *bufio.Writer
	if compress {
		gzWriter := gzip.NewWriter(file)
		defer gzWriter.Close()
		writer = bufio.NewWriter(gzWriter)
	} else {
		writer = bufio.NewWriter(file)
	}
	defer writer.Flush()

	// Channel for results
	resultChan := make(chan ScanResult, maxConcurrent)
	var wg sync.WaitGroup

	// Process chunks
	for chunkStart := start; chunkStart <= end; chunkStart += uint32(chunkSize) {
		chunkEnd := chunkStart + uint32(chunkSize) - 1
		if chunkEnd > end {
			chunkEnd = end
		}

		wg.Add(1)
		go func(startNum, endNum uint32) {
			defer wg.Done()
			scanChunk(startNum, endNum, ports, timeout, maxConcurrent, resultChan)
		}(chunkStart, chunkEnd)

		// Collect results for this chunk
		expectedResults := int(chunkEnd-chunkStart+1) * len(ports)
		fmt.Printf("Expecting %d results for chunk %s to %s\n", expectedResults, uint32ToIP(chunkStart), uint32ToIP(chunkEnd))
		for i := 0; i < expectedResults; i++ {
			result := <-resultChan
			if result.Open {
				line := fmt.Sprintf("Port %d is open on %s\n", result.Port, result.IP)
				email.Subject = "Open port found"
				email.Msg = line
				send(email)
				if _, err := writer.WriteString(line); err != nil {
					fmt.Printf("Error writing to file: %v\n", err)
				}
			}
		}

		// Save checkpoint
		if err := saveCheckpoint(checkpointFile, uint32ToIP(chunkEnd)); err != nil {
			fmt.Printf("Warning: failed to save checkpoint: %v\n", err)
		}

		// Progress update
		processed := uint64(chunkEnd) - uint64(start) + 1
		percent := float64(processed) / float64(totalIPs) * 100
		if percent > 100 {
			percent = 100
		}
		fmt.Printf("Progress: %.2f%% complete\n", percent)

		// Flush periodically
		writer.Flush()

		// Wait if not parallel
		if !parallelChunks {
			wg.Wait()
		}
	}

	// Wait for all parallel chunks and close channel
	if parallelChunks {
		wg.Wait()
	}
	close(resultChan)

	return nil
}

func ipToUint32(ipStr string) uint32 {
	ip := net.ParseIP(ipStr).To4()
	return uint32(ip[0])<<24 + uint32(ip[1])<<16 + uint32(ip[2])<<8 + uint32(ip[3])
}

func uint32ToIP(ipNum uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d",
		byte(ipNum>>24), byte(ipNum>>16), byte(ipNum>>8), byte(ipNum))
}

func parsePorts(portStr string) []int {
	var ports []int
	portList := strings.Split(portStr, ",")
	for _, p := range portList {
		if port, err := strconv.Atoi(p); err == nil && port > 0 && port <= 65535 {
			ports = append(ports, port)
		}
	}
	return ports
}

func main() {
	fmt.Println("Starting port scanner")
	//os.Exit(0)
	//defer recoverPanic()
	go update()
	startIP := flag.String("start", "192.168.1.1", "Starting IP address")
	endIP := flag.String("end", "192.168.1.10", "Ending IP address")
	portList := flag.String("ports", "25", "Comma-separated list of ports")
	timeout := flag.Duration("timeout", 2*time.Second, "Connection timeout")
	maxConcurrent := flag.Int("concurrent", 1000, "Maximum concurrent scans per chunk")
	chunkSize := flag.Int("chunk", 1000000, "Number of IPs per chunk (default: 1M)")
	parallelChunks := flag.Bool("parallel", false, "Run chunks in parallel (experimental)")
	outputFile := flag.String("output", "scan_results.txt", "Output file for results")
	checkpointFile := flag.String("checkpoint", "checkpoint.txt", "Checkpoint file for resuming")
	compress := flag.Bool("compress", false, "Compress output file with gzip")
	flag.Parse()

	email.Msg = "Starting scan from " + *startIP + " to " + *endIP + " on ports " + *portList
	send(email)

	// Start the timer
	startTime := time.Now()

	// Launch a goroutine to print elapsed time every minute
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				elapsed := time.Since(startTime)
				fmt.Printf("Elapsed time: %.0f minutes\n", elapsed.Minutes())
			case <-done:
				return
			}
		}
	}()

	ports := parsePorts(*portList)
	err := scanRange(*startIP, *endIP, ports, *timeout, *maxConcurrent, *chunkSize, *parallelChunks, *outputFile, *checkpointFile, *compress)
	if err != nil {
		fmt.Printf("Error during scan: %v\n", err)
		os.Exit(1)
	}

	// Stop the timer goroutine
	close(done)

	// Print total elapsed time
	elapsed := time.Since(startTime)
	fmt.Printf("Scan completed in %.2f minutes. Results saved to %s\n", elapsed.Minutes(), *outputFile)
	//remove checkpoint file
	//os.Remove(*checkpointFile)
	//remove output file
	//os.Remove(*outputFile)
}
