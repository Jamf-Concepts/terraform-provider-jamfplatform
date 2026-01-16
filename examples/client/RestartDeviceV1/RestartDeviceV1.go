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

	commands, err := apiClient.RestartDeviceV1(context.Background(), deviceID)
	if err != nil {
		log.Fatalf("Error restarting device %s: %v", deviceID, err)
	}

	fmt.Printf("Restart requested for device %s\n", deviceID)
	printCommands(commands)
}

func printCommands(commands []client.DeviceCommandResponseV1) {
	if len(commands) == 0 {
		fmt.Println("API returned no command references.")
		return
	}

	fmt.Println("Command references:")
	for i, cmd := range commands {
		fmt.Printf("  %d. Device ID: %s | Command ID: %s\n", i+1, cmd.DeviceID, cmd.CommandID)
	}
}
