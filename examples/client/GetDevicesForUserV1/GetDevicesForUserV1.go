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
	userID := os.Getenv("JAMF_USER_ID")
	deviceFilter := os.Getenv("JAMF_DEVICE_FILTER")
	var sortFields []string

	if env := os.Getenv("JAMF_CLIENT_ID"); env != "" {
		clientID = env
	}
	if env := os.Getenv("JAMF_CLIENT_SECRET"); env != "" {
		clientSecret = env
	}
	if env := os.Getenv("JAMF_BASE_URL"); env != "" {
		baseURL = env
	}
	if env := os.Getenv("JAMF_DEVICE_SORT"); env != "" {
		sortFields = strings.Split(env, ",")
	}

	if clientID == "" || clientSecret == "" || baseURL == "" {
		log.Fatal("Missing required configuration: JAMF_CLIENT_ID, JAMF_CLIENT_SECRET, JAMF_BASE_URL")
	}
	if userID == "" {
		log.Fatal("Missing required configuration: JAMF_USER_ID")
	}

	apiClient := client.NewClient(baseURL, clientID, clientSecret)

	devices, err := apiClient.GetDevicesForUserV1(context.Background(), userID, sortFields, deviceFilter)
	if err != nil {
		log.Fatalf("Error getting devices for user %s: %v", userID, err)
	}

	fmt.Printf("User %s has %d device(s)\n\n", userID, len(devices))
	for _, d := range devices {
		fmt.Printf("Device: %s (%s) - %s\n", d.ID, d.Name, d.Model)
	}

	fmt.Print("\n" + strings.Repeat("=", 50) + "\n")
	fmt.Printf("Full JSON Response:\n")
	fmt.Print(strings.Repeat("=", 50) + "\n")

	b, err := json.MarshalIndent(devices, "", "  ")
	if err != nil {
		log.Printf("Error marshaling to JSON: %v", err)
	} else {
		fmt.Println(string(b))
	}
}
