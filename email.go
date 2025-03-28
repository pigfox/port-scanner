package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// sendFunc for side-effect testing
var sendFunc func(Email) = func(e Email) {}

// sendImpl is the actual implementation, overridable for testing
var sendImpl func(Email) int = func(p Email) int {
	jsonData, err := createPayload(p)
	if err != nil {
		fmt.Println("Error creating JSON payload:", err)
		return 500
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", brevo.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return 500
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("api-key", brevo.APIKEY)
	req.Header.Add("content-type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return 500
	}
	defer res.Body.Close()

	return res.StatusCode
}

// send function uses the mockable implementation
func send(p Email) int {
	sendFunc(p)        // For test side-effects
	return sendImpl(p) // Mockable core logic
}

// createPayload remains unchanged
func createPayload(p Email) ([]byte, error) {
	formattedMsg := strings.ReplaceAll(p.Msg, "\n", "<br>")
	formattedMsg += "<br>on " + time.Now().Format("2006-01-02 15:04:05") + " in timezone " + time.Now().Location().String()
	html := "<html><head></head><body><p>" + formattedMsg + "</p></body></html>"

	payload := EmailPayload{
		Subject:     p.Subject,
		HTMLContent: html,
		Headers:     map[string]string{"Reply-To": p.SenderEmail},
	}
	payload.Sender.Email = p.SenderEmail
	payload.Sender.Name = p.SenderName
	payload.To = []struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}{{Email: p.ToEmail, Name: p.ToName}}

	return json.MarshalIndent(payload, "", "  ")
}
