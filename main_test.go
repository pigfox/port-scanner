package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	// Set up environment variables for tests
	os.Setenv("BREVO_URL", "http://example.com/api/v3/smtp/email")
	os.Setenv("BREVO_APIKEY", "testkey")
	os.Setenv("SENDER_EMAIL", "sender@example.com")
	os.Setenv("TO_EMAIL", "admin@example.com")
	os.Setenv("PORT", "10001")

	// Initialize globals with test values
	brevo = Brevo{URL: os.Getenv("BREVO_URL"), APIKEY: os.Getenv("BREVO_APIKEY")}
	email = Email{
		SenderName:  "Port Scanner Bot",
		SenderEmail: os.Getenv("SENDER_EMAIL"),
		ToName:      "Admin",
		ToEmail:     os.Getenv("TO_EMAIL"),
		Subject:     "Port Scan Results",
		Msg:         "",
	}
	results = nil
	checkpoints = nil

	// Run tests
	code := m.Run()

	// Exit with the test result code
	os.Exit(code)
}

// TestInit tests the initialization logic
func TestInit(t *testing.T) {
	// Clear environment to ensure isolation
	os.Clearenv()

	// Simulate .env file content
	envContent := "BREVO_URL=http://test.com\nBREVO_APIKEY=testkey\nSENDER_EMAIL=sender@test.com\nTO_EMAIL=admin@test.com"
	tmpFile, err := os.CreateTemp("", "testenv")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(envContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Load from temp file
	if err := godotenv.Load(tmpFile.Name()); err != nil {
		t.Fatalf("godotenv.Load failed: %v", err)
	}

	// Replicate init logic
	brevo = Brevo{}
	email = Email{}
	brevoUrl := os.Getenv("BREVO_URL")
	brevoAPIKey := os.Getenv("BREVO_APIKEY")
	if brevoUrl == "" || brevoAPIKey == "" {
		t.Fatal("BREVO_URL or BREVO_APIKEY not set")
	}
	brevo = Brevo{URL: brevoUrl, APIKEY: brevoAPIKey}
	email = Email{
		SenderName:  "Port Scanner Bot",
		SenderEmail: os.Getenv("SENDER_EMAIL"),
		ToName:      "Admin",
		ToEmail:     os.Getenv("TO_EMAIL"),
		Subject:     "Port Scan Results",
		Msg:         "",
	}

	if brevo.URL == "" || brevo.APIKEY == "" || email.SenderEmail == "" || email.ToEmail == "" {
		t.Errorf("Initialization failed: %+v, %+v", brevo, email)
	}
}

// TestRecoverPanic tests the recoverPanic function
func TestRecoverPanic(t *testing.T) {
	defer recoverPanic()
	panic("test panic")
	// If we reach here, recoverPanic worked
}

// TestUpdate tests the update function
func TestUpdate(t *testing.T) {
	originalSend := sendFunc
	originalSleep := updateSleepDuration
	defer func() {
		sendFunc = originalSend
		updateSleepDuration = originalSleep
	}()
	sendCalled := false
	sendFunc = func(e Email) {
		sendCalled = true
		if e.Subject != "Update" || e.Msg != "Updating..." {
			t.Errorf("update() sent wrong email: %+v", e)
		}
	}
	updateSleepDuration = 10 * time.Millisecond
	done := make(chan struct{})
	go func() {
		update()
		close(done)
	}()
	select {
	case <-done:
		// Update completed
	case <-time.After(100 * time.Millisecond):
		t.Fatal("update() timed out")
	}
	if !sendCalled {
		t.Errorf("update() did not call send")
	}
}

// TestScanPort tests the scanPort function
func TestScanPort(t *testing.T) {
	limiter := make(chan struct{}, 1)

	originalDial := dialTimeout
	defer func() { dialTimeout = originalDial }()

	// Test open port
	dialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
		return &net.TCPConn{}, nil
	}
	result := scanPort("127.0.0.1", 80, 1*time.Second, limiter)
	if !result.Open || result.Error != nil {
		t.Errorf("scanPort() failed for open port: %+v", result)
	}

	// Test closed port
	dialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
		return nil, fmt.Errorf("connection refused")
	}
	result = scanPort("127.0.0.1", 81, 1*time.Second, limiter)
	if result.Open || result.Error == nil {
		t.Errorf("scanPort() failed for closed port: %+v", result)
	}
}

// TestScanChunk tests the scanChunk function
func TestScanChunk(t *testing.T) {
	originalDial := dialTimeout
	dialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
		return &net.TCPConn{}, nil
	}
	defer func() { dialTimeout = originalDial }()
	resultChan := make(chan ScanResult, 10)
	var wg sync.WaitGroup
	wg.Add(1)
	go scanChunk(ipToUint32("192.168.1.1"), ipToUint32("192.168.1.2"), []int{80}, 1*time.Millisecond, 2, resultChan)
	wg.Wait()
	close(resultChan)

	count := 0
	for result := range resultChan {
		count++
		if !result.Open {
			t.Errorf("scanChunk() failed: %+v", result)
		}
	}
	if count != 2 {
		t.Errorf("scanChunk() scanned wrong number of IPs: %d", count)
	}
}

// TestSaveAndLoadCheckpoint tests checkpoint functions
func TestSaveAndLoadCheckpoint(t *testing.T) {
	checkpoints = nil
	saveCheckpoint("192.168.1.1")
	if len(checkpoints) != 1 || checkpoints[0].IP != "192.168.1.1" {
		t.Errorf("saveCheckpoint() failed: %+v", checkpoints)
	}
	if loadCheckpoint() != "192.168.1.1" {
		t.Errorf("loadCheckpoint() failed: %s", loadCheckpoint())
	}
}

// TestScanRange tests the scanRange function
func TestScanRange(t *testing.T) {
	originalSend := sendFunc
	originalDial := dialTimeout
	defer func() {
		sendFunc = originalSend
		dialTimeout = originalDial
	}()
	sendCalled := 0
	sendFunc = func(e Email) {
		sendCalled++
	}
	dialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
		return &net.TCPConn{}, nil
	}
	results = nil // Reset global state
	checkpoints = nil
	err := scanRange("192.168.1.1", "192.168.1.2", []int{80}, 1*time.Millisecond, 2, 2, false)
	if err != nil {
		t.Errorf("scanRange() failed: %v", err)
	}
	time.Sleep(50 * time.Millisecond) // Allow goroutines to finish
	if sendCalled < 1 {
		t.Errorf("scanRange() did not send emails, sent: %d", sendCalled)
	}
	if len(results) != 2 {
		t.Errorf("scanRange() produced wrong number of results: %d", len(results))
	}
}

// TestIPConversion tests IP conversion functions
func TestIPConversion(t *testing.T) {
	ip := "192.168.1.1"
	num := ipToUint32(ip)
	if num != 3232235777 {
		t.Errorf("ipToUint32() failed: %d", num)
	}
	if uint32ToIP(num) != ip {
		t.Errorf("uint32ToIP() failed: %s", uint32ToIP(num))
	}
}

// TestParsePorts tests the parsePorts function
func TestParsePorts(t *testing.T) {
	ports := parsePorts("80,443,invalid,65536")
	if len(ports) != 2 || ports[0] != 80 || ports[1] != 443 {
		t.Errorf("parsePorts() failed: %+v", ports)
	}
}

// TestMainFunction tests the main function
func TestMainFunction(t *testing.T) {
	originalSend := sendFunc
	defer func() { sendFunc = originalSend }()
	sendCalled := 0
	sendFunc = func(e Email) {
		sendCalled++
	}

	// Start server in a controlled way
	srv := &http.Server{Addr: ":10001"}
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Errorf("Server failed: %v", err)
		}
	}()

	// Run main in a goroutine with clean flag state
	done := make(chan struct{})
	go func() {
		defer recoverPanic()
		os.Args = []string{"port-scanner"}
		main()
		close(done)
	}()

	time.Sleep(100 * time.Millisecond) // Allow main to start
	if sendCalled == 0 {
		t.Errorf("main() did not send initial email")
	}

	// Test health endpoint
	resp, err := http.Get("http://localhost:10001/health")
	if err != nil {
		t.Errorf("Health endpoint failed: %v", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Errorf("Health endpoint returned %d, expected 200", resp.StatusCode)
		resp.Body.Close()
	} else {
		resp.Body.Close()
	}

	// Shutdown server
	if err := srv.Shutdown(nil); err != nil {
		t.Errorf("Failed to shutdown server: %v", err)
	}
	<-done // Ensure main finishes
}

// BenchmarkScanPort benchmarks the scanPort function
func BenchmarkScanPort(b *testing.B) {
	limiter := make(chan struct{}, 1)
	originalDial := dialTimeout
	dialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
		return &net.TCPConn{}, nil
	}
	defer func() { dialTimeout = originalDial }()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanPort("127.0.0.1", 80, 1*time.Millisecond, limiter)
	}
}
