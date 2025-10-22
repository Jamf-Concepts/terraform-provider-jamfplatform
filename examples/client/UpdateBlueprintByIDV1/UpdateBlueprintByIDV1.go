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
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <blueprint-id>\n", os.Args[0])
		os.Exit(1)
	}
	blueprintID := os.Args[1]

	// Configuration - you can also use environment variables
	clientID := "example-client-id"
	clientSecret := "example-client-secret"
	baseURL := "https://us.apigw.jamf.com"

	// Use environment variables if set
	if envClientID := os.Getenv("JAMF_CLIENT_ID"); envClientID != "" {
		clientID = envClientID
	}
	if envClientSecret := os.Getenv("JAMF_CLIENT_SECRET"); envClientSecret != "" {
		clientSecret = envClientSecret
	}
	if envBaseURL := os.Getenv("JAMF_BASE_URL"); envBaseURL != "" {
		baseURL = envBaseURL
	}

	if clientID == "" || clientSecret == "" || baseURL == "" {
		log.Fatal("Missing required configuration: JAMF_CLIENT_ID, JAMF_CLIENT_SECRET, JAMF_BASE_URL")
	}

	// Example: update blueprint (customize as needed)
	// Helper to marshal config to json.RawMessage
	marshalConfig := func(cfg interface{}) json.RawMessage {
		b, err := json.Marshal(cfg)
		if err != nil {
			log.Fatalf("Error marshaling configuration: %v", err)
		}
		return b
	}

	updateRequest := &client.BlueprintUpdateRequestV1{
		Name:        "Updated Blueprint Name",
		Description: "Updated description",
		Scope: client.BlueprintUpdateScopeV1{
			DeviceGroups: []string{"fce3d9a5-8660-42ff-a95e-625e7b53b48a"},
		},
		Steps: []client.BlueprintStepV1{
			{
				Name: "Updated Declaration group",
				Components: []client.BlueprintComponentV1{
					{
						Identifier: "com.jamf.ddm.sw-updates",
						Configuration: marshalConfig(map[string]interface{}{
							"detailsURL": map[string]interface{}{
								"Included": false,
								"Value":    "",
							},
							"targetLocalDateTime": "2025-08-23T16:00:00",
							"targetOSVersion":     "18.1",
						}),
					},
				},
			},
		},
	}

	// Print the JSON request before sending
	jsonReq, err := json.MarshalIndent(updateRequest, "", "  ")
	if err != nil {
		log.Printf("Error marshaling update request to JSON: %v", err)
	} else {
		fmt.Print("\n" + strings.Repeat("-", 50) + "\n")
		fmt.Printf("Blueprint Update Request (JSON):\n")
		fmt.Print(strings.Repeat("-", 50) + "\n")
		fmt.Println(string(jsonReq))
		fmt.Print(strings.Repeat("-", 50) + "\n\n")
	}

	// Initialize the client (baseURL-based)
	apiClient := client.NewClient(baseURL, clientID, clientSecret)

	// Update the blueprint
	err = apiClient.UpdateBlueprintV1(context.Background(), blueprintID, updateRequest)
	if err != nil {
		log.Fatalf("Error updating blueprint: %v", err)
	}

	fmt.Printf("Blueprint updated successfully!\n")
}
