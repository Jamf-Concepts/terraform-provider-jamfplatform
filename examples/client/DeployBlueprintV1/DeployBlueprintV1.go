// Copyright 2025 Jamf Software LLC.

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <blueprint-id>\n", os.Args[0])
		os.Exit(1)
	}
	blueprintID := os.Args[1]

	// Configuration - you can also use environment variables
	clientID := "example-client-id"
	clientSecret := "example-client-secret"
	baseURL := "https://us.apigw.jamf.com"

	// Use environment variables if set
	if envClientID := os.Getenv("JAMF_CLIENT_ID"); envClientID != "" {
		clientID = envClientID
	}
	if envClientSecret := os.Getenv("JAMF_CLIENT_SECRET"); envClientSecret != "" {
		clientSecret = envClientSecret
	}
	if envBaseURL := os.Getenv("JAMF_BASE_URL"); envBaseURL != "" {
		baseURL = envBaseURL
	}

	if clientID == "" || clientSecret == "" || baseURL == "" {
		log.Fatal("Missing required configuration: JAMF_CLIENT_ID, JAMF_CLIENT_SECRET, JAMF_BASE_URL")
	}

	// Initialize the client (baseURL-based)
	apiClient := client.NewClient(baseURL, clientID, clientSecret)

	// Deploy blueprint by ID
	fmt.Printf("Deploying blueprint: %s\n", blueprintID)
	if err := apiClient.DeployBlueprintV1(context.Background(), blueprintID); err != nil {
		log.Fatalf("Error deploying blueprint %s: %v", blueprintID, err)
	}
	fmt.Println("Deployment started successfully (202 Accepted)")
}
