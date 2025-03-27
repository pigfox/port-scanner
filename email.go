package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var senderName = "Port Scanner Bot"
var senderEmail = ""
var toName = "Admin"

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

func send(p Email) int {
	p.Msg += " on " + time.Now().Format("2006-01-02 15:04:05") + " in timezone " + time.Now().Location().String()
	method := "POST"
	html := "<html><head></head><body><p>" + p.Msg + "</p></body></html>"

	payload := map[string]interface{}{
		"sender": map[string]string{
			"name":  p.SenderName,
			"email": p.SenderEmail,
		},
		"to": []map[string]string{
			{
				"email": p.ToEmail,
				"name":  p.ToName,
			},
		},
		"subject":     "" + p.Subject + "",
		"htmlContent": "" + html + "",
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Println(err)
		return 500
	}

	// Create a new request using http
	client := &http.Client{}
	req, err := http.NewRequest(method, brevo.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println(err)
		return 500
	}

	// Add request headers
	req.Header.Add("accept", "application/json")
	req.Header.Add("api-key", brevo.APIKEY) // Replace YOUR_API_KEY with your actual API key
	req.Header.Add("content-type", "application/json")

	// Send the request
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return 500
	}
	defer res.Body.Close()
	return res.StatusCode
}
