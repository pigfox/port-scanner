package main

var brevo Brevo
var email Email
var results []ScanResult
var checkpoints []Checkpoint

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
