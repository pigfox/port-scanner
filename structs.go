package main

import (
	"net"
	"time"
)

var brevo Brevo
var email Email
var results []ScanResult
var checkpoints []Checkpoint

// DialerFunc is a type for the dialer function
type DialerFunc func(network, address string, timeout time.Duration) (net.Conn, error)

// dialTimeout is the global dialer function, defaulting to net.DialTimeout
var dialTimeout DialerFunc = net.DialTimeout

// updateSleepDuration allows overriding the sleep time in tests
var updateSleepDuration = 12 * time.Hour

type ScanResult struct {
	IP    string
	Port  int
	Open  bool
	Error error
}

type Checkpoint struct {
	IP string
}

type Email struct {
	SenderName  string
	SenderEmail string
	ToName      string
	ToEmail     string
	Subject     string
	Msg         string
}

type Brevo struct {
	URL    string
	APIKEY string
}

type EmailPayload struct {
	Sender struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"sender"`
	To []struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"to"`
	Subject     string            `json:"subject"`
	HTMLContent string            `json:"htmlContent"`
	Headers     map[string]string `json:"headers"`
}
