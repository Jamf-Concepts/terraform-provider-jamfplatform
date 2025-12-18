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
	deviceID := os.Getenv("JAMF_DEVICE_ID")
	filter := os.Getenv("JAMF_DEVICE_APP_FILTER")
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
	if env := os.Getenv("JAMF_DEVICE_APP_SORT"); env != "" {
		sortFields = strings.Split(env, ",")
	}

	if clientID == "" || clientSecret == "" || baseURL == "" {
		log.Fatal("Missing required configuration: JAMF_CLIENT_ID, JAMF_CLIENT_SECRET, JAMF_BASE_URL")
	}
	if deviceID == "" {
		log.Fatal("Missing required configuration: JAMF_DEVICE_ID")
	}

	apiClient := client.NewClient(baseURL, clientID, clientSecret)

	apps, err := apiClient.GetDeviceInstalledApplicationsV1(context.Background(), deviceID, sortFields, filter)
	if err != nil {
		log.Fatalf("Error getting installed applications for device %s: %v", deviceID, err)
	}

	fmt.Printf("Found %d application(s) on device %s\n\n", len(apps), deviceID)
	for _, app := range apps {
		fmt.Printf("%s - %s\n", app.Name, app.Version)
	}

	fmt.Print("\n" + strings.Repeat("=", 50) + "\n")
	fmt.Printf("Full JSON Response:\n")
	fmt.Print(strings.Repeat("=", 50) + "\n")

	b, err := json.MarshalIndent(apps, "", "  ")
	if err != nil {
		log.Printf("Error marshaling to JSON: %v", err)
	} else {
		fmt.Println(string(b))
	}
}
