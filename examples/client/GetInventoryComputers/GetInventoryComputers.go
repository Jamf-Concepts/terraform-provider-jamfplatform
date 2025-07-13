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

	// Pagination and filter from env or defaults
	page := 0
	pageSize := 100
	filter := ""
	if envPage := os.Getenv("INVENTORY_PAGE"); envPage != "" {
		fmt.Sscanf(envPage, "%d", &page)
	}
	if envPageSize := os.Getenv("INVENTORY_PAGE_SIZE"); envPageSize != "" {
		fmt.Sscanf(envPageSize, "%d", &pageSize)
	}
	if envFilter := os.Getenv("INVENTORY_FILTER"); envFilter != "" {
		filter = envFilter
	}

	// Initialize the client (region-based)
	apiClient := client.NewInventoryClient(region, clientID, clientSecret)

	// Get computers (paginated, filtered)
	results, err := apiClient.GetInventoryComputers(context.Background(), page, pageSize, filter)
	if err != nil {
		log.Fatalf("Error listing computers: %v", err)
	}

	fmt.Printf("Found %d computer(s) (Total: %d)\n\n", len(results.Results), results.TotalCount)
	for _, comp := range results.Results {
		fmt.Printf("ID: %s\nName: %s\nSerial: %s\nUDID: %s\nUser: %s\nOS: %s %s\n\n",
			comp.ID,
			comp.General.Name,
			comp.Hardware.SerialNumber,
			comp.UDID,
			comp.UserAndLocation.Username,
			comp.OperatingSystem.Name,
			comp.OperatingSystem.Version,
		)
	}

	// Print the full JSON response
	fmt.Print("\n" + strings.Repeat("=", 50) + "\n")
	fmt.Printf("Full JSON Response:\n")
	fmt.Print(strings.Repeat("=", 50) + "\n")

	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		log.Printf("Error marshaling to JSON: %v", err)
	} else {
		fmt.Println(string(jsonData))
	}
}
