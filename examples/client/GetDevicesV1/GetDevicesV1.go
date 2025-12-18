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

	apiClient := client.NewClient(baseURL, clientID, clientSecret)

	devices, err := apiClient.GetDevicesV1(context.Background(), sortFields, deviceFilter)
	if err != nil {
		log.Fatalf("Error getting devices: %v", err)
	}

	fmt.Printf("Found %d device(s)\n\n", len(devices))
	for _, d := range devices {
		fmt.Printf("ID: %s\nName: %s\nModel: %s (%s)\nSerial: %s\nOS: %s\nLast Inventory: %s\nUserID: %s\n\n",
			d.ID, d.Name, d.Model, d.ModelIdentifier, d.SerialNumber, d.OperatingSystemVersion,
			d.LastInventoryUpdateTime, valueOrPlaceholder(d.UserID, "<unassigned>"))
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

func valueOrPlaceholder(v *string, placeholder string) string {
	if v == nil || *v == "" {
		return placeholder
	}
	return *v
}
