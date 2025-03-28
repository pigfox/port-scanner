package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func init() {
	liveENV := os.Getenv("LIVE_ENV")
	if liveENV != "TRUE" {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}
	brevoUrl := os.Getenv("BREVO_URL")
	brevoAPIKey := os.Getenv("BREVO_APIKEY")
	if brevoUrl == "" || brevoAPIKey == "" {
		log.Fatal("BREVO_URL is not set")
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
}

func recoverPanic() {
	if r := recover(); r != nil {
		fmt.Printf("Recovered from panic: %v\n", r)
	}
}

func update() {
	time.Sleep(12 * time.Hour)
	email.Subject = "Update"
	email.Msg = "Updating..."
	send(email)
}
