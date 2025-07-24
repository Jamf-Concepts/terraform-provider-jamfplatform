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

	// Get computer ID from command line argument or environment variable
	var computerID string
	if len(os.Args) > 1 {
		computerID = os.Args[1]
	} else if envComputerID := os.Getenv("COMPUTER_ID"); envComputerID != "" {
		computerID = envComputerID
	} else {
		log.Fatal("Please provide a computer ID as a command line argument or set COMPUTER_ID environment variable")
	}

	// Initialize the client (region-based)
	apiClient := client.NewClient(region, clientID, clientSecret)

	// Get specific computer by ID
	comp, err := apiClient.GetInventoryComputerByID(context.Background(), computerID)
	if err != nil {
		log.Fatalf("Error getting computer %s: %v", computerID, err)
	}

	fmt.Printf("Computer Details:\n")
	fmt.Printf("ID: %s\n", comp.ID)
	fmt.Printf("Name: %s\n", comp.General.Name)
	fmt.Printf("Serial Number: %s\n", comp.Hardware.SerialNumber)
	fmt.Printf("UDID: %s\n", comp.UDID)
	fmt.Printf("Last IP: %s\n", comp.General.LastIpAddress)
	fmt.Printf("User: %s\n", comp.UserAndLocation.Username)
	fmt.Printf("OS: %s %s\n", comp.OperatingSystem.Name, comp.OperatingSystem.Version)

	// Print the full JSON response
	fmt.Print("\n" + strings.Repeat("=", 50) + "\n")
	fmt.Printf("Full JSON Response:\n")
	fmt.Print(strings.Repeat("=", 50) + "\n")

	jsonData, err := json.MarshalIndent(comp, "", "  ")
	if err != nil {
		log.Printf("Error marshaling to JSON: %v", err)
	} else {
		fmt.Println(string(jsonData))
	}
}
