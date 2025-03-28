package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var senderName = "Port Scanner Bot"
var senderEmail = ""
var toName = "Admin"

// createPayload builds a valid JSON payload for Brevo API
func createPayload(p Email) ([]byte, error) {
	// Replace newlines with <br> for proper HTML formatting
	formattedMsg := strings.ReplaceAll(p.Msg, "\n", "<br>")

	// Append timestamp on a new line
	formattedMsg += "<br>on " + time.Now().Format("2006-01-02 15:04:05") + " in timezone " + time.Now().Location().String()

	// Full HTML body
	html := "<html><head></head><body><p>" + formattedMsg + "</p></body></html>"

	payload := EmailPayload{
		Subject:     p.Subject,
		HTMLContent: html,
		Headers: map[string]string{
			"Reply-To": p.SenderEmail,
		},
	}

	// Fill sender info
	payload.Sender.Email = p.SenderEmail
	payload.Sender.Name = p.SenderName

	// Fill recipient info
	payload.To = []struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}{
		{Email: p.ToEmail, Name: p.ToName},
	}

	// Convert to JSON
	return json.MarshalIndent(payload, "", "  ")
}

// send function sends an email via Brevo API
func send(p Email) int {
	jsonData, err := createPayload(p)
	if err != nil {
		fmt.Println("Error creating JSON payload:", err)
		return 500
	}

	//fmt.Println("Generated JSON Payload:\n", string(jsonData))

	// Create HTTP request
	client := &http.Client{}
	req, err := http.NewRequest("POST", brevo.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return 500
	}

	// Add request headers
	req.Header.Add("accept", "application/json")
	req.Header.Add("api-key", brevo.APIKEY)
	req.Header.Add("content-type", "application/json")

	// Send request
	res, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return 500
	}
	defer res.Body.Close()

	//fmt.Println("Response Status:", res.StatusCode)
	return res.StatusCode
}
