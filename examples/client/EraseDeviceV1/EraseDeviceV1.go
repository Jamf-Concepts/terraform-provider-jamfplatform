// Copyright 2025 Jamf Software LLC.

package main

import (
	"context"
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

	eraseRequest, hasOptions := buildEraseRequestFromEnv()
	if !hasOptions {
		eraseRequest = nil
	}

	commands, err := apiClient.EraseDeviceV1(context.Background(), deviceID, eraseRequest)
	if err != nil {
		log.Fatalf("Error erasing device %s: %v", deviceID, err)
	}

	fmt.Printf("Erase requested for device %s\n", deviceID)
	printCommands(commands)
}

func buildEraseRequestFromEnv() (*client.EraseDeviceRequestV1, bool) {
	req := &client.EraseDeviceRequestV1{}
	var changed bool

	if v := os.Getenv("JAMF_ERASE_PRESERVE_DATA_PLAN"); v != "" {
		val := parseBool(v)
		req.PreserveDataPlan = &val
		changed = true
	}
	if v := os.Getenv("JAMF_ERASE_DISALLOW_PROXIMITY_SETUP"); v != "" {
		val := parseBool(v)
		req.DisallowProximitySetup = &val
		changed = true
	}
	if v := os.Getenv("JAMF_ERASE_CLEAR_ACTIVATION_LOCK"); v != "" {
		val := parseBool(v)
		req.ClearActivationLock = &val
		changed = true
	}
	if v := os.Getenv("JAMF_ERASE_RETURN_TO_SERVICE"); v != "" {
		val := parseBool(v)
		req.ReturnToService = &val
		changed = true
	}
	if v := os.Getenv("JAMF_ERASE_PIN"); v != "" {
		pin := v
		req.Pin = &pin
		changed = true
	}

	return req, changed
}

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
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
