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

	// Sections from env or default to general only
	sections := []string{client.MobileDeviceSectionGeneral}
	if envSections := os.Getenv("INVENTORY_SECTIONS"); envSections != "" {
		// Parse comma-separated sections from environment variable
		// Example: INVENTORY_SECTIONS="GENERAL,HARDWARE,SECURITY"
		sections = strings.Split(envSections, ",")
		// Trim whitespace from each section
		for i, section := range sections {
			sections[i] = strings.TrimSpace(section)
		}
	}

	// Alternative: Get all available sections
	// sections = client.ValidMobileDeviceSections()

	// Available sections: GENERAL, HARDWARE, USER_AND_LOCATION, PURCHASING,
	// SECURITY, APPLICATIONS, EBOOKS, NETWORK, SERVICE_SUBSCRIPTIONS,
	// CERTIFICATES, PROFILES, USER_PROFILES, PROVISIONING_PROFILES,
	// SHARED_USERS, EXTENSION_ATTRIBUTES

	fmt.Printf("Requesting sections: %s\n", strings.Join(sections, ", "))

	// Initialize the client (baseURL-based)
	apiClient := client.NewClient(baseURL, clientID, clientSecret)

	// Get all mobile devices (automatic pagination handling)
	// This function automatically fetches all pages and returns all devices as a single slice
	// For manual pagination control, use: apiClient.GetInventoryMobileDevices(ctx, page, pageSize, sections)
	// For memory-efficient processing of large datasets, see the GetInventoryMobileDevicesCallback example
	devices, err := apiClient.GetInventoryAllMobileDevices(context.Background(), sections)
	if err != nil {
		log.Fatalf("Error listing mobile devices: %v", err)
	}

	fmt.Printf("Found %d mobile device(s)\n\n", len(devices))
	for _, dev := range devices {
		fmt.Printf("ID: %s\nName: %s\nSerial: %s\nUDID: %s\nUser: %s\nOS: %s\n\n",
			dev.MobileDeviceId,
			dev.General.DisplayName,
			dev.Hardware.SerialNumber,
			dev.General.Udid,
			dev.UserAndLocation.Username,
			dev.General.OsVersion,
		)
	}

	// Print the full JSON response
	fmt.Print("\n" + strings.Repeat("=", 50) + "\n")
	fmt.Printf("Full JSON Response:\n")
	fmt.Print(strings.Repeat("=", 50) + "\n")

	jsonData, err := json.MarshalIndent(devices, "", "  ")
	if err != nil {
		log.Printf("Error marshaling to JSON: %v", err)
	} else {
		fmt.Println(string(jsonData))
	}
}
