package main

import (
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

	// Load checkpoint
	resumeIP := loadCheckpoint()
	if resumeIP != "" {
		resume := ipToUint32(resumeIP)
		if resume >= start && resume <= end {
			start = resume + 1
			fmt.Printf("Resuming from %s\n", resumeIP)
		}
	}

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

		// Collect results
		for i := 0; i < int(chunkEnd-chunkStart+1)*len(ports); i++ {
			result := <-resultChan
			results = append(results, result) // Store in-memory instead of writing to file
			if result.Open {
				fmt.Printf("Port %d is open on %s\n", result.Port, result.IP)
				email.Subject = "Open port found"
				email.Msg = fmt.Sprintf("Port %d is open on %s", result.Port, result.IP)
				send(email)
			}
		}

		// Save checkpoint
		saveCheckpoint(uint32ToIP(chunkEnd))
	}

	wg.Wait()
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
	defer recoverPanic()
	go update()
	startIP := flag.String("start", "192.168.1.1", "Starting IP address")
	endIP := flag.String("end", "192.168.1.10", "Ending IP address")
	portList := flag.String("ports", "25", "Comma-separated list of ports")
	timeout := flag.Duration("timeout", 2*time.Second, "Connection timeout")
	maxConcurrent := flag.Int("concurrent", 1000, "Maximum concurrent scans per chunk")
	chunkSize := flag.Int("chunk", 1000000, "Number of IPs per chunk (default: 1M)")
	parallelChunks := flag.Bool("parallel", false, "Run chunks in parallel (experimental)")
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
	err := scanRange(*startIP, *endIP, ports, *timeout, *maxConcurrent, *chunkSize, *parallelChunks)
	if err != nil {
		fmt.Printf("Error during scan: %v\n", err)
		os.Exit(1)
	}

	// Stop the timer goroutine
	close(done)
	//iterate over results and send email
	msgBuilder := strings.Builder{}
	for _, result := range results {
		if result.Open {
			//fmt.Printf("Port %d is open on %s\n", result.Port, result.IP)
			msgBuilder.WriteString(fmt.Sprintf("Port %d is open on %s\n\n", result.Port, result.IP))
		}
	}

	email.Subject = "Open port summary"
	email.Msg = msgBuilder.String()
	send(email)

	// Print total elapsed time
	elapsed := time.Since(startTime)
	fmt.Printf("Scan completed in %.2f minutes.", elapsed.Minutes())

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000" // Default port if PORT environment variable is not set
	}

	// Start HTTP server with health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // Send a 200 OK response
	})

	// Start the HTTP server in a separate goroutine
	go func() {
		fmt.Println("Starting HTTP server on :" + port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			fmt.Printf("Error starting server: %v\n", err)
		}
	}()

	// Block forever to keep the server running
	select {}
}
