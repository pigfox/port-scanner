package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
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
	time.Sleep(updateSleepDuration)
	email.Subject = "Update"
	email.Msg = "Updating..."
	send(email)
}
