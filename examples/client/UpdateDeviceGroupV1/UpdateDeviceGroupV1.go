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
	// Replace this with the group you want to update
	groupID := "example-group-id"

	if env := os.Getenv("JAMF_CLIENT_ID"); env != "" {
		clientID = env
	}
	if env := os.Getenv("JAMF_CLIENT_SECRET"); env != "" {
		clientSecret = env
	}
	if env := os.Getenv("JAMF_BASE_URL"); env != "" {
		baseURL = env
	}
	if env := os.Getenv("JAMF_EXAMPLE_GROUP_ID"); env != "" {
		groupID = env
	}

	if clientID == "" || clientSecret == "" || baseURL == "" {
		log.Fatal("Missing required configuration: JAMF_CLIENT_ID, JAMF_CLIENT_SECRET, JAMF_BASE_URL")
	}

	apiClient := client.NewClient(baseURL, clientID, clientSecret)

	description := "Updated description"
	req := &client.DeviceGroupUpdateRepresentationV1{
		Name:        "example-updated-name",
		Description: &description,
		// Optionally update criteria or deviceIds depending on group type
	}

	if err := apiClient.UpdateDeviceGroupV1(context.Background(), groupID, req); err != nil {
		log.Fatalf("Error updating device group: %v", err)
	}

	fmt.Println("Device group updated successfully")

	// Retrieve and print the updated group
	grp, err := apiClient.GetDeviceGroupByIDV1(context.Background(), groupID)
	if err != nil {
		log.Fatalf("Error retrieving updated group: %v", err)
	}

	fmt.Print("\n" + strings.Repeat("=", 50) + "\n")
	b, err := json.MarshalIndent(grp, "", "  ")
	if err != nil {
		log.Printf("Error marshaling to JSON: %v", err)
	} else {
		fmt.Println(string(b))
	}
}
