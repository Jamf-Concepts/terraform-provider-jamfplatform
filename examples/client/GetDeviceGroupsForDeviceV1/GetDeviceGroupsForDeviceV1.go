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
	clientID := "example-client-id"
	clientSecret := "example-client-secret"
	baseURL := "https://us.apigw.jamf.com"
	// Replace with a real device ID for a real run
	deviceID := "example-device-id"

	if env := os.Getenv("JAMF_CLIENT_ID"); env != "" {
		clientID = env
	}
	if env := os.Getenv("JAMF_CLIENT_SECRET"); env != "" {
		clientSecret = env
	}
	if env := os.Getenv("JAMF_BASE_URL"); env != "" {
		baseURL = env
	}
	if env := os.Getenv("JAMF_EXAMPLE_DEVICE_ID"); env != "" {
		deviceID = env
	}

	if clientID == "" || clientSecret == "" || baseURL == "" {
		log.Fatal("Missing required configuration: JAMF_CLIENT_ID, JAMF_CLIENT_SECRET, JAMF_BASE_URL")
	}

	apiClient := client.NewClient(baseURL, clientID, clientSecret)

	resp, err := apiClient.GetDeviceGroupsForDeviceV1(context.Background(), deviceID, 0, 50)
	if err != nil {
		log.Fatalf("Error getting device groups for device: %v", err)
	}

	fmt.Printf("Found %d group(s) on page %d\n\n", len(resp.Results), resp.Page)

	for _, g := range resp.Results {
		fmt.Printf("GroupID: %s\nGroupName: %s\n", g.GroupID, g.GroupName)
	}

	fmt.Print("\n" + strings.Repeat("=", 50) + "\n")
	fmt.Printf("Full JSON Response:\n")
	fmt.Print(strings.Repeat("=", 50) + "\n")

	b, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		log.Printf("Error marshaling to JSON: %v", err)
	} else {
		fmt.Println(string(b))
	}
}
