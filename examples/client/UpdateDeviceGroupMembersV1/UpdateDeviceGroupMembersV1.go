// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

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
	// Replace with a real static group ID
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

	patch := &client.DeviceGroupMemberPatchRepresentationV1{
		Added:   []string{"device-uuid-new-1"},
		Removed: []string{"device-uuid-old-1"},
	}

	if err := apiClient.UpdateDeviceGroupMembersV1(context.Background(), groupID, patch); err != nil {
		log.Fatalf("Error updating device group members: %v", err)
	}

	fmt.Printf("Updated members for group %s\n", groupID)
}
