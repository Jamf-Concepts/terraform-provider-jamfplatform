// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

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
	newName := os.Getenv("JAMF_DEVICE_NEW_NAME")
	desiredUser := os.Getenv("JAMF_DEVICE_USER_ID")

	if env := os.Getenv("JAMF_CLIENT_ID"); env != "" {
		clientID = env
	}
	if env := os.Getenv("JAMF_CLIENT_SECRET"); env != "" {
		clientSecret = env
	}
	if env := os.Getenv("JAMF_BASE_URL"); env != "" {
		baseURL = env
	}

	if deviceID == "" {
		log.Fatal("Missing required configuration: JAMF_DEVICE_ID")
	}
	if newName == "" {
		newName = "Renamed Device Example"
	}
	if clientID == "" || clientSecret == "" || baseURL == "" {
		log.Fatal("Missing required configuration: JAMF_CLIENT_ID, JAMF_CLIENT_SECRET, JAMF_BASE_URL")
	}

	apiClient := client.NewClient(baseURL, clientID, clientSecret)

	payload := &client.DeviceUpdateRepresentationV1{
		Name: &newName,
	}

	if desiredUser != "" {
		if strings.EqualFold(desiredUser, "null") {
			payload.UserID = client.NewNullableStringNull()
		} else {
			payload.UserID = client.NewNullableString(desiredUser)
		}
	}

	if err := apiClient.UpdateDeviceV1(context.Background(), deviceID, payload); err != nil {
		log.Fatalf("Error updating device %s: %v", deviceID, err)
	}

	fmt.Printf("Updated device %s\n\n", deviceID)

	updatedDevice, err := apiClient.GetDeviceByIDV1(context.Background(), deviceID)
	if err != nil {
		log.Fatalf("Error retrieving updated device %s: %v", deviceID, err)
	}

	fmt.Printf("New Name: %s\nAssigned User ID: %s\n", updatedDevice.Name, valueOrPlaceholder(updatedDevice.UserID, "<unassigned>"))

	fmt.Print("\n" + strings.Repeat("=", 50) + "\n")
	fmt.Printf("Full JSON Response:\n")
	fmt.Print(strings.Repeat("=", 50) + "\n")

	b, err := json.MarshalIndent(updatedDevice, "", "  ")
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
