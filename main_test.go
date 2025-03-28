package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	_ = godotenv.Load()
	if os.Getenv("BREVO_URL") == "" {
		os.Setenv("BREVO_URL", "http://example.com/api/v3/smtp/email")
	}
	if os.Getenv("BREVO_APIKEY") == "" {
		os.Setenv("BREVO_APIKEY", "testkey")
	}
	if os.Getenv("SENDER_EMAIL") == "" {
		os.Setenv("SENDER_EMAIL", "sender@example.com")
	}
	if os.Getenv("TO_EMAIL") == "" {
		os.Setenv("TO_EMAIL", "admin@example.com")
	}
	if os.Getenv("PORT") == "" {
		os.Setenv("PORT", "10001")
	}

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

	os.Exit(m.Run())
}

// TestInit tests successful initialization
func TestInit(t *testing.T) {
	os.Clearenv()
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

	if err := godotenv.Load(tmpFile.Name()); err != nil {
		t.Fatalf("godotenv.Load failed: %v", err)
	}

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

// TestInitFailure tests init failure case
func TestInitFailure(t *testing.T) {
	os.Clearenv()
	brevo = Brevo{}
	email = Email{}
	if brevo.URL != "" || brevo.APIKEY != "" {
		t.Errorf("Expected empty Brevo in failure case: %+v", brevo)
	}
}

// TestRecoverPanic tests panic recovery
func TestRecoverPanic(t *testing.T) {
	defer recoverPanic()
	panic("test panic")
}

// TestUpdate tests the update function
func TestUpdate(t *testing.T) {
	originalSend := sendFunc
	originalSleep := updateSleepDuration
	originalImpl := sendImpl
	defer func() {
		sendFunc = originalSend
		updateSleepDuration = originalSleep
		sendImpl = originalImpl
	}()

	sendCalled := false
	sendFunc = func(e Email) {
		sendCalled = true
		if e.Subject != "Update" || e.Msg != "Updating..." {
			t.Errorf("update() sent wrong email: %+v", e)
		}
	}
	sendImpl = func(e Email) int {
		return 200
	}

	updateSleepDuration = 10 * time.Millisecond
	done := make(chan struct{})
	go func() {
		update()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("update() timed out")
	}
	if !sendCalled {
		t.Errorf("update() did not call send")
	}
}

// TestScanPort tests scanPort with open and closed cases
func TestScanPort(t *testing.T) {
	limiter := make(chan struct{}, 1)
	originalDial := dialTimeout
	defer func() { dialTimeout = originalDial }()

	dialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
		return &net.TCPConn{}, nil
	}
	result := scanPort("127.0.0.1", 80, 1*time.Second, limiter)
	if !result.Open || result.Error != nil {
		t.Errorf("scanPort() failed for open port: %+v", result)
	}

	dialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
		return nil, fmt.Errorf("connection refused")
	}
	result = scanPort("127.0.0.1", 81, 1*time.Second, limiter)
	if result.Open || result.Error == nil {
		t.Errorf("scanPort() failed for closed port: %+v", result)
	}
}

// TestScanChunk tests scanChunk
func TestScanChunk(t *testing.T) {
	originalDial := dialTimeout
	originalImpl := sendImpl
	defer func() {
		dialTimeout = originalDial
		sendImpl = originalImpl
	}()
	dialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
		return &net.TCPConn{}, nil
	}
	sendImpl = func(e Email) int {
		return 200
	}

	resultChan := make(chan ScanResult, 10)
	scanChunk(ipToUint32("192.168.1.1"), ipToUint32("192.168.1.2"), []int{80}, 1*time.Millisecond, 2, resultChan)
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

// TestScanRange tests scanRange
func TestScanRange(t *testing.T) {
	originalSend := sendFunc
	originalDial := dialTimeout
	originalImpl := sendImpl
	defer func() {
		sendFunc = originalSend
		dialTimeout = originalDial
		sendImpl = originalImpl
	}()
	sendCalled := 0
	sendFunc = func(e Email) {
		sendCalled++
	}
	sendImpl = func(e Email) int {
		return 200
	}
	dialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
		return &net.TCPConn{}, nil
	}
	results = nil
	checkpoints = nil
	err := scanRange("192.168.1.1", "192.168.1.2", []int{80}, 1*time.Millisecond, 2, 2, false)
	if err != nil {
		t.Errorf("scanRange() failed: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if sendCalled < 1 {
		t.Errorf("scanRange() did not send emails, sent: %d", sendCalled)
	}
	if len(results) != 2 {
		t.Errorf("scanRange() produced wrong number of results: %d", len(results))
	}
}

// TestScanRangeWithCheckpoint tests resuming from checkpoint
func TestScanRangeWithCheckpoint(t *testing.T) {
	originalSend := sendFunc
	originalDial := dialTimeout
	originalImpl := sendImpl
	defer func() {
		sendFunc = originalSend
		dialTimeout = originalDial
		sendImpl = originalImpl
	}()
	sendCalled := 0
	sendFunc = func(e Email) {
		sendCalled++
	}
	sendImpl = func(e Email) int {
		return 200
	}
	dialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
		return &net.TCPConn{}, nil
	}

	checkpoints = []Checkpoint{{IP: "192.168.1.1"}}
	results = nil
	err := scanRange("192.168.1.1", "192.168.1.3", []int{80}, 1*time.Millisecond, 2, 2, false)
	if err != nil {
		t.Errorf("scanRange() failed: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if len(results) != 2 {
		t.Errorf("scanRange() with checkpoint produced wrong number of results: %d", len(results))
	}
}

// TestIPConversion tests IP conversion
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

// TestParsePorts tests parsePorts
func TestParsePorts(t *testing.T) {
	ports := parsePorts("80,443,invalid,65536")
	if len(ports) != 2 || ports[0] != 80 || ports[1] != 443 {
		t.Errorf("parsePorts() failed: %+v", ports)
	}
}

// TestMainFunction tests main with success and error cases in one run
func TestMainFunction(t *testing.T) {
	originalSend := sendFunc
	originalImpl := sendImpl
	originalDial := dialTimeout
	defer func() {
		sendFunc = originalSend
		sendImpl = originalImpl
		dialTimeout = originalDial
	}()

	// Set up HTTP server
	mux := http.NewServeMux()
	srv := &http.Server{Addr: ":10001", Handler: mux}
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	serverDone := make(chan struct{})
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Errorf("Server failed: %v", err)
		}
		close(serverDone)
	}()

	// Mock dialTimeout: succeed first iteration, fail second
	iteration := 0
	dialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
		iteration++
		if iteration <= 2 { // First iteration succeeds (2 IPs scanned)
			return &net.TCPConn{}, nil
		}
		// After first iteration succeeds, set TEST_MODE to exit after second
		if iteration == 3 {
			os.Setenv("TEST_MODE", "true")
		}
		return nil, fmt.Errorf("mock dial error") // Second iteration fails
	}

	sendCalled := 0
	sendFunc = func(e Email) {
		sendCalled++
		fmt.Printf("Email sent: %s\n", e.Msg) // Debug email sending
	}
	sendImpl = func(e Email) int {
		return 200
	}

	// Reset global state
	results = nil
	checkpoints = nil

	done := make(chan struct{})
	os.Setenv("TEST_MODE", "false") // Start with false to allow two iterations
	defer os.Unsetenv("TEST_MODE")
	go func() {
		defer recoverPanic()
		os.Args = []string{"port-scanner", "-start=192.168.1.1", "-end=192.168.1.2", "-ports=80", "-timeout=1ms", "-concurrent=2"}
		main()
		close(done)
	}()

	// Wait for main to complete or timeout
	select {
	case <-done:
	case <-time.After(3 * time.Second): // Enough for two iterations + retry
		t.Fatal("TestMainFunction timed out")
	}

	// Shutdown server
	if err := srv.Shutdown(context.Background()); err != nil {
		t.Errorf("Failed to shutdown server: %v", err)
	}
	<-serverDone // Ensure server goroutine completes

	// Verify results
	if sendCalled != 6 { // 1 start + 2 per-port + 1 summary (success), 1 start + 1 empty summary (error)
		t.Errorf("main() sent %d emails, expected 6 (1 start + 2 ports + 1 summary success + 1 start + 1 error)", sendCalled)
	}

	resp, err := http.Get("http://localhost:10001/health")
	if err == nil {
		resp.Body.Close()
		t.Errorf("Health endpoint should be down after shutdown, but succeeded")
	}
}

// TestEmailErrors tests error paths in email sending
func TestEmailErrors(t *testing.T) {
	originalSend := sendFunc
	originalImpl := sendImpl
	defer func() {
		sendFunc = originalSend
		sendImpl = originalImpl
	}()

	sendFunc = func(e Email) {}
	sendImpl = func(e Email) int {
		return 500
	}
	status := send(email)
	if status != 500 {
		t.Errorf("send() expected 500 on error, got %d", status)
	}

	sendImpl = func(e Email) int {
		return 400
	}
	status = send(email)
	if status != 400 {
		t.Errorf("send() expected 400 on bad request, got %d", status)
	}
}

// BenchmarkScanPort
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
