package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

func scanPort(ip string, port int, timeout time.Duration, limiter chan struct{}) ScanResult {
	limiter <- struct{}{}
	defer func() { <-limiter }()

	address := fmt.Sprintf("%s:%d", ip, port)
	conn, err := dialTimeout("tcp", address, timeout)

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
	limiter := make(chan struct{}, maxConcurrent)
	var innerWg sync.WaitGroup

	fmt.Printf("Scanning chunk from %s to %s\n", uint32ToIP(startIPNum), uint32ToIP(endIPNum))

	for ipNum := startIPNum; ipNum <= endIPNum; ipNum++ {
		ip := uint32ToIP(ipNum)
		for _, port := range ports {
			innerWg.Add(1)
			go func(ip string, port int) {
				defer innerWg.Done()
				result := scanPort(ip, port, timeout, limiter)
				resultChan <- result
			}(ip, port)
		}
	}
	innerWg.Wait()
}

func saveCheckpoint(lastIP string) {
	checkpoints = append(checkpoints, Checkpoint{IP: lastIP})
}

func loadCheckpoint() string {
	if len(checkpoints) > 0 {
		return checkpoints[len(checkpoints)-1].IP
	}
	return ""
}

func scanRange(startIP, endIP string, ports []int, timeout time.Duration, maxConcurrent, chunkSize int, parallelChunks bool) error {
	start := ipToUint32(startIP)
	end := ipToUint32(endIP)

	resumeIP := loadCheckpoint()
	if resumeIP != "" {
		resume := ipToUint32(resumeIP)
		if resume >= start && resume <= end {
			start = resume + 1
			fmt.Printf("Resuming from %s\n", resumeIP)
		}
	}

	resultChan := make(chan ScanResult, maxConcurrent)
	var wg sync.WaitGroup
	done := make(chan struct{})

	// Collect results in a separate goroutine
	go func() {
		for result := range resultChan {
			results = append(results, result)
			if result.Open {
				fmt.Printf("Port %d is open on %s\n", result.Port, result.IP)
				email.Subject = "Open port found"
				email.Msg = fmt.Sprintf("Port %d is open on %s", result.Port, result.IP)
				send(email)
			}
		}
		close(done)
	}()

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

		saveCheckpoint(uint32ToIP(chunkEnd))
	}

	wg.Wait()
	close(resultChan) // Safe to close after all chunks are done
	<-done            // Wait for result collection to finish

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
	defer recoverPanic()
	go update()

	// Flags
	startIP := flag.String("start", "192.168.1.1", "Starting IP address")
	endIP := flag.String("end", "192.168.1.10", "Ending IP address")
	portList := flag.String("ports", "25", "Comma-separated list of ports")
	timeout := flag.Duration("timeout", 2*time.Second, "Connection timeout")
	maxConcurrent := flag.Int("concurrent", 1000, "Maximum concurrent scans per chunk")
	chunkSize := flag.Int("chunk", 1000000, "Number of IPs per chunk (default: 1M)")
	parallelChunks := flag.Bool("parallel", false, "Run chunks in parallel (experimental)")
	flag.Parse()

	ports := parsePorts(*portList)

	// HTTP server setup
	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	server := &http.Server{Addr: ":" + port, Handler: mux}

	go func() {
		fmt.Println("Starting HTTP server on :" + port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Error starting server: %v\n", err)
		}
	}()

	// Infinite scan loop
	for {
		// Reset global state for each run
		results = nil
		checkpoints = nil

		email.Msg = "Starting scan from " + *startIP + " to " + *endIP + " on ports " + *portList
		send(email)

		startTime := time.Now()
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

		err := scanRange(*startIP, *endIP, ports, *timeout, *maxConcurrent, *chunkSize, *parallelChunks)
		if err != nil {
			fmt.Printf("Error during scan: %v\n", err)
			time.Sleep(5 * time.Second) // Brief delay before retrying on error
			continue
		}

		close(done)

		msgBuilder := strings.Builder{}
		for _, result := range results {
			if result.Open {
				msgBuilder.WriteString(fmt.Sprintf("Port %d is open on %s\n\n", result.Port, result.IP))
			}
		}

		email.Subject = "Open port summary"
		email.Msg = msgBuilder.String()
		send(email)

		elapsed := time.Since(startTime)
		fmt.Printf("Scan completed in %.2f minutes. Restarting...\n", elapsed.Minutes())

		// Optional delay between scans (e.g., to avoid overwhelming the network)
		time.Sleep(1 * time.Second)

		// For testing, allow exit
		if os.Getenv("TEST_MODE") == "true" {
			break // Exit the loop in test mode
		}
	}

	// Shutdown server (only reached in TEST_MODE)
	if err := server.Shutdown(context.Background()); err != nil {
		fmt.Printf("Error shutting down server: %v\n", err)
	}
}
