package main

import (
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

func TestIpToUint32(t *testing.T) {
	tests := []struct {
		ip   string
		want uint32
	}{
		{"192.168.1.1", 3232235777},
		{"10.0.0.1", 167772161},
		{"255.255.255.255", 4294967295},
	}

	for _, tt := range tests {
		got := ipToUint32(tt.ip)
		if got != tt.want {
			t.Errorf("ipToUint32(%q) = %v, want %v", tt.ip, got, tt.want)
		}
	}
}

func TestUint32ToIP(t *testing.T) {
	tests := []struct {
		num  uint32
		want string
	}{
		{3232235777, "192.168.1.1"},
		{167772161, "10.0.0.1"},
		{4294967295, "255.255.255.255"},
	}

	for _, tt := range tests {
		got := uint32ToIP(tt.num)
		if got != tt.want {
			t.Errorf("uint32ToIP(%v) = %q, want %q", tt.num, got, tt.want)
		}
	}
}

func TestParsePorts(t *testing.T) {
	tests := []struct {
		input string
		want  []int
	}{
		{"80,443,8080", []int{80, 443, 8080}},
		{"22", []int{22}},
		{"80,invalid,443", []int{80, 443}},
		{"", []int{}},
		{"65536,0,-1", []int{}},
	}

	for _, tt := range tests {
		got := parsePorts(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("parsePorts(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("parsePorts(%q) = %v, want %v", tt.input, got, tt.want)
				break
			}
		}
	}
}

func TestScanPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock server: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	limiter := make(chan struct{}, 1)

	go func() {
		conn, _ := listener.Accept()
		if conn != nil {
			conn.Close()
		}
	}()
	result := scanPort("127.0.0.1", port, 100*time.Millisecond, limiter)
	if !result.Open || result.IP != "127.0.0.1" || result.Port != port {
		t.Errorf("scanPort open case: got %+v, want Open=true, IP=127.0.0.1, Port=%d", result, port)
	}

	result = scanPort("127.0.0.1", 54321, 100*time.Millisecond, limiter)
	if result.Open || result.Error == nil {
		t.Errorf("scanPort closed case: got %+v, want Open=false with error", result)
	}
}

func TestScanChunk(t *testing.T) {
	ports := []int{54321}
	resultChan := make(chan ScanResult, 10)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanChunk(ipToUint32("192.168.1.1"), ipToUint32("192.168.1.2"), ports, 100*time.Millisecond, 2, resultChan)
	}()

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	count := 0
	for result := range resultChan {
		count++
		if result.Open {
			t.Errorf("scanChunk: unexpected open port %d on %s", result.Port, result.IP)
		}
	}
	if count != 2 {
		t.Errorf("scanChunk: expected 2 results, got %d", count)
	}
}

func TestSaveCheckpoint(t *testing.T) {
	tempFile := "test_checkpoint.txt"
	defer os.Remove(tempFile)

	err := saveCheckpoint(tempFile, "192.168.1.1")
	if err != nil {
		t.Errorf("saveCheckpoint: %v", err)
	}

	data, err := os.ReadFile(tempFile)
	if err != nil {
		t.Errorf("reading checkpoint: %v", err)
	}
	if string(data) != "192.168.1.1\n" {
		t.Errorf("saveCheckpoint: got %q, want %q", string(data), "192.168.1.1\n")
	}

	err = saveCheckpoint("/invalid/path", "1.1.1.1")
	if err == nil {
		t.Errorf("saveCheckpoint: expected error for invalid path")
	}
}

func TestLoadCheckpoint(t *testing.T) {
	tempFile := "test_checkpoint.txt"
	defer os.Remove(tempFile)

	ip, err := loadCheckpoint(tempFile)
	if err != nil || ip != "" {
		t.Errorf("loadCheckpoint no file: got %q, %v; want %q, nil", ip, err, "")
	}

	os.WriteFile(tempFile, []byte("192.168.1.1\n"), 0644)
	ip, err = loadCheckpoint(tempFile)
	if err != nil || ip != "192.168.1.1" {
		t.Errorf("loadCheckpoint: got %q, %v; want %q, nil", ip, err, "192.168.1.1")
	}

	os.Chmod(tempFile, 0000)
	_, err = loadCheckpoint(tempFile)
	if err == nil {
		t.Errorf("loadCheckpoint: expected error for unreadable file")
	}
	os.Chmod(tempFile, 0644)
}

func TestScanRange(t *testing.T) {
	outputFile := "test_output.txt"
	checkpointFile := "test_checkpoint.txt"
	defer os.Remove(outputFile)
	defer os.Remove(checkpointFile)

	// Basic test
	err := scanRange("192.168.1.1", "192.168.1.2", []int{54321}, 100*time.Millisecond, 2, 1, false, outputFile, checkpointFile, false)
	if err != nil {
		t.Errorf("scanRange: %v", err)
	}

	data, err := os.ReadFile(checkpointFile)
	if err != nil || string(data) != "192.168.1.2\n" {
		t.Errorf("scanRange checkpoint: got %q, %v; want %q, nil", string(data), err, "192.168.1.2\n")
	}

	// Test resume
	os.WriteFile(checkpointFile, []byte("192.168.1.1\n"), 0644)
	err = scanRange("192.168.1.1", "192.168.1.2", []int{54321}, 100*time.Millisecond, 2, 1, false, outputFile, checkpointFile, false)
	if err != nil {
		t.Errorf("scanRange resume: %v", err)
	}

	// Test parallel chunks
	err = scanRange("192.168.1.1", "192.168.1.2", []int{54321}, 100*time.Millisecond, 2, 1, true, outputFile, checkpointFile, false)
	if err != nil {
		t.Errorf("scanRange parallel: %v", err)
	}

	// Test compression
	err = scanRange("192.168.1.1", "192.168.1.2", []int{54321}, 100*time.Millisecond, 2, 1, false, outputFile, checkpointFile, true)
	if err != nil {
		t.Errorf("scanRange compress: %v", err)
	}

	// Test error cases
	err = scanRange("192.168.1.1", "192.168.1.2", []int{54321}, 100*time.Millisecond, 2, 1, false, "/invalid/path/output", checkpointFile, false)
	if err == nil {
		t.Errorf("scanRange: expected error for invalid output file")
	}

	// Test invalid checkpoint load
	os.WriteFile(checkpointFile, []byte("192.168.1.1\n"), 0644)
	os.Chmod(checkpointFile, 0000)
	err = scanRange("192.168.1.1", "192.168.1.2", []int{54321}, 100*time.Millisecond, 2, 1, false, outputFile, checkpointFile, false)
	if err == nil {
		t.Errorf("scanRange: expected error for invalid checkpoint file load")
	}
	os.Chmod(checkpointFile, 0644)
}

func TestMain(t *testing.T) {
	// Single run with custom args to cover all flags and timer
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{
		"portscanner",
		"-start=192.168.1.1",
		"-end=192.168.1.2",
		"-ports=54321",
		"-timeout=1ms",
		"-concurrent=2",
		"-chunk=1",
		"-parallel=true",
		"-output=test_main.txt",
		"-checkpoint=test_main_checkpoint.txt",
		"-compress=true",
	}

	done := make(chan struct{})
	go func() {
		main()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second): // Allow timer to tick
	}

	// Cleanup
	os.Remove("test_main.txt")
	os.Remove("test_main_checkpoint.txt")
}
