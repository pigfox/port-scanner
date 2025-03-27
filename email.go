package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Brevo struct {
	URL    string
	APIKEY string
}

func email(p map[string]string) int {
	method := "POST"
	html := "<html><head></head><body><p>" + p["msg"] + "</p></body></html>"

	payload := map[string]interface{}{
		"sender": map[string]string{
			"name":  p["sender_name"],
			"email": p["sender_email"],
		},
		"to": []map[string]string{
			{
				"email": p["to_email"],
				"name":  p["to_name"],
			},
		},
		"subject":     "" + p["subject"] + "",
		"htmlContent": "" + html + "",
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Println(err)
		return 500
	}
	fmt.Println(string(jsonData))

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
