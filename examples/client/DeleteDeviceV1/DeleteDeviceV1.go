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
	clientID := "example-client-id"
	clientSecret := "example-client-secret"
	baseURL := "https://us.apigw.jamf.com"
	deviceID := os.Getenv("JAMF_DEVICE_ID")

	if env := os.Getenv("JAMF_CLIENT_ID"); env != "" {
		clientID = env
	}
	if env := os.Getenv("JAMF_CLIENT_SECRET"); env != "" {
		clientSecret = env
	}
	if env := os.Getenv("JAMF_BASE_URL"); env != "" {
		baseURL = env
	}

	if clientID == "" || clientSecret == "" || baseURL == "" {
		log.Fatal("Missing required configuration: JAMF_CLIENT_ID, JAMF_CLIENT_SECRET, JAMF_BASE_URL")
	}
	if deviceID == "" {
		log.Fatal("Missing required configuration: JAMF_DEVICE_ID")
	}

	apiClient := client.NewClient(baseURL, clientID, clientSecret)

	if err := apiClient.DeleteDeviceV1(context.Background(), deviceID); err != nil {
		log.Fatalf("Error deleting device %s: %v", deviceID, err)
	}

	fmt.Printf("Device %s deleted.\n", deviceID)
}
