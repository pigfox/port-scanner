package main

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

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

func scanRange(startIP, endIP string, ports []int, timeout time.Duration, maxConcurrent int) []ScanResult {
	var results []ScanResult
	var wg sync.WaitGroup
	resultChan := make(chan ScanResult, 100)
	limiter := make(chan struct{}, maxConcurrent)

	start := ipToUint32(startIP)
	end := ipToUint32(endIP)

	for ipNum := start; ipNum <= end; ipNum++ {
		ip := uint32ToIP(ipNum)
		for _, port := range ports {
			wg.Add(1)
			go func(ip string, port int) {
				defer wg.Done()
				resultChan <- scanPort(ip, port, timeout, limiter)
			}(ip, port)
		}
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		results = append(results, result)
	}

	return results
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
	startIP := flag.String("start", "192.168.1.1", "Starting IP address")
	endIP := flag.String("end", "192.168.1.10", "Ending IP address")
	portList := flag.String("ports", "25", "Comma-separated list of ports") //"25,80,443"
	timeout := flag.Duration("timeout", 2*time.Second, "Connection timeout")
	maxConcurrent := flag.Int("concurrent", 50, "Maximum concurrent scans")
	flag.Parse()

	ports := parsePorts(*portList)
	results := scanRange(*startIP, *endIP, ports, *timeout, *maxConcurrent)

	for _, result := range results {
		if result.Open {
			fmt.Printf("Port %d is open on %s\n", result.Port, result.IP)
		} else if result.Error != nil {
			fmt.Printf("Error scanning %s:%d - %v\n", result.IP, result.Port, result.Error)
		}
	}
}
