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

	// Get mobile device ID from command line argument or environment variable
	var deviceID string
	if len(os.Args) > 1 {
		deviceID = os.Args[1]
	} else if envDeviceID := os.Getenv("MOBILE_DEVICE_ID"); envDeviceID != "" {
		deviceID = envDeviceID
	} else {
		log.Fatal("Please provide a mobile device ID as a command line argument or set MOBILE_DEVICE_ID environment variable")
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

	// Get specific mobile device by ID
	dev, err := apiClient.GetInventoryMobileDeviceByID(context.Background(), deviceID, sections)
	if err != nil {
		log.Fatalf("Error getting mobile device %s: %v", deviceID, err)
	}

	fmt.Printf("Mobile Device Details:\n")
	fmt.Printf("ID: %s\n", dev.MobileDeviceId)
	fmt.Printf("Name: %s\n", dev.General.DisplayName)
	fmt.Printf("Serial Number: %s\n", dev.Hardware.SerialNumber)
	fmt.Printf("UDID: %s\n", dev.General.Udid)
	fmt.Printf("Last IP: %s\n", dev.General.IpAddress)
	fmt.Printf("User: %s\n", dev.UserAndLocation.Username)
	fmt.Printf("OS: %s\n", dev.General.OsVersion)

	// Print the full JSON response
	fmt.Print("\n" + strings.Repeat("=", 50) + "\n")
	fmt.Printf("Full JSON Response:\n")
	fmt.Print(strings.Repeat("=", 50) + "\n")

	jsonData, err := json.MarshalIndent(dev, "", "  ")
	if err != nil {
		log.Printf("Error marshaling to JSON: %v", err)
	} else {
		fmt.Println(string(jsonData))
	}
}
