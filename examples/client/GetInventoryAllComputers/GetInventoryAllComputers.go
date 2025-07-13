// Copyright 2025 Jamf Software LLC.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
)

func main() {
	// Configuration - you can also use environment variables
	clientID := "example-client-id"
	clientSecret := "example-client-secret"
	region := "eu" // us, eu, or apac

	// Alternatively, use environment variables
	if envClientID := os.Getenv("JAMF_CLIENT_ID"); envClientID != "" {
		clientID = envClientID
	}
	if envClientSecret := os.Getenv("JAMF_CLIENT_SECRET"); envClientSecret != "" {
		clientSecret = envClientSecret
	}
	if envRegion := os.Getenv("JAMF_REGION"); envRegion != "" {
		region = envRegion
	}

	if clientID == "" || clientSecret == "" || region == "" {
		log.Fatal("Missing required configuration: JAMF_CLIENT_ID, JAMF_CLIENT_SECRET, JAMF_REGION")
	}

	// Filter from env or default
	filter := ""
	if envFilter := os.Getenv("INVENTORY_FILTER"); envFilter != "" {
		filter = envFilter
	}

	// Initialize the client (region-based)
	apiClient := client.NewInventoryClient(region, clientID, clientSecret)

	// Get all computers (automatic pagination handling)
	// This function automatically fetches all pages and returns all computers as a single slice
	computers, err := apiClient.GetInventoryAllComputers(context.Background(), filter)
	if err != nil {
		log.Fatalf("Error listing computers: %v", err)
	}

	fmt.Printf("Found %d mobile device(s)\n\n", len(computers))
	for _, dev := range computers {
		fmt.Printf("ID: %s\nName: %s\nSerial: %s\nUDID: %s\nUser: %s\nOS: %s\n\n",
			dev.ID,
			dev.General.Name,
			dev.Hardware.SerialNumber,
			dev.General.ManagementId,
			dev.UserAndLocation.Username,
			dev.General.Platform,
		)
	}

	// Print the full JSON response
	fmt.Print("\n" + strings.Repeat("=", 50) + "\n")
	fmt.Printf("Full JSON Response:\n")
	fmt.Print(strings.Repeat("=", 50) + "\n")

	jsonData, err := json.MarshalIndent(computers, "", "  ")
	if err != nil {
		log.Printf("Error marshaling to JSON: %v", err)
	} else {
		fmt.Println(string(jsonData))
	}
}
