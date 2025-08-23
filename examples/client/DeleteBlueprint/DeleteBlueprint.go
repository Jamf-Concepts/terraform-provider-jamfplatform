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
	// Configuration - you can also use environment variables
	clientID := "example-client-id"
	clientSecret := "example-client-secret"
	baseURL := "https://us.apigw.jamf.com"

	// Alternatively, use environment variables
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

	// Get Blueprint ID from command line argument or environment variable
	var blueprintID string
	if len(os.Args) > 1 {
		blueprintID = os.Args[1]
	} else if envBlueprintID := os.Getenv("BLUEPRINT_ID"); envBlueprintID != "" {
		blueprintID = envBlueprintID
	} else {
		log.Fatal("Please provide a Blueprint ID as a command line argument or set BLUEPRINT_ID environment variable")
	}

	// Initialize the client (baseURL-based)
	apiClient := client.NewClient(baseURL, clientID, clientSecret)

	// Delete the Blueprint
	err := apiClient.DeleteBlueprint(context.Background(), blueprintID)
	if err != nil {
		log.Fatalf("Error deleting Blueprint %s: %v", blueprintID, err)
	}

	fmt.Printf("Blueprint %s deleted successfully.\n", blueprintID)
}
