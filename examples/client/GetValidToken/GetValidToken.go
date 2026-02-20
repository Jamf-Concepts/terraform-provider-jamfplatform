package main

// Copyright 2025 Jamf Software LLC.

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
)

func main() {
	clientID := os.Getenv("JAMF_CLIENT_ID")
	clientSecret := os.Getenv("JAMF_CLIENT_SECRET")
	baseURL := os.Getenv("JAMF_BASE_URL")

	if clientID == "" || clientSecret == "" || baseURL == "" {
		log.Fatal("Missing required environment variables: JAMF_CLIENT_ID, JAMF_CLIENT_SECRET, JAMF_BASE_URL")
	}

	fmt.Println("Testing OAuth Authentication")
	fmt.Println("============================")
	fmt.Printf("Base URL: %s\n", baseURL)
	fmt.Printf("Client ID: %s\n\n", maskString(clientID))

	apiClient := client.NewClient(baseURL, clientID, clientSecret)

	fmt.Println("Validating credentials...")

	if err := apiClient.ValidateCredentials(context.Background()); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}

	fmt.Println("Authentication successful!")
}

// maskString masks a string for display, showing only first 4 and last 4 characters.
func maskString(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}
