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
	// Configuration - you can also use environment variables
	clientID := "example-client-id"
	clientSecret := "example-client-secret"
	baseURL := "https://us.apigw.jamf.com"

	// Alternatively, use environment variables
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

	// Example: create a new blueprint (customize as needed)
	// Helper to marshal config to json.RawMessage
	marshalConfig := func(cfg interface{}) json.RawMessage {
		b, err := json.Marshal(cfg)
		if err != nil {
			log.Fatalf("Error marshaling configuration: %v", err)
		}
		return b
	}

	request := &client.BlueprintCreateRequestV1{
		Name:        "Updates and things",
		Description: "This is a description",
		Scope: client.BlueprintCreateScopeV1{
			DeviceGroups: []string{"fce3d9a5-8660-42ff-a95e-625e7b53b48a"},
		},
		Steps: []client.BlueprintStepV1{
			{
				Name: "Declaration group",
				Components: []client.BlueprintComponentV1{
					{
						Identifier: "com.jamf.ddm.sw-updates",
						Configuration: marshalConfig(map[string]interface{}{
							"detailsURL": map[string]interface{}{
								"Included": false,
								"Value":    "",
							},
							"targetLocalDateTime": "2025-08-23T15:45:00",
							"targetOSVersion":     "18.0",
						}),
					},
					{
						Identifier: "com.jamf.ddm-configuration-profile",
						Configuration: marshalConfig(map[string]interface{}{
							"payloadContent": []map[string]interface{}{
								{
									"allowAccountModification":     true,
									"allowActivityContinuation":    true,
									"allowAddingGameCenterFriends": true,
									"payloadIdentifier":            "f54e2c52-0d26-46ea-9d42-7cf40ef015ea",
									"payloadType":                  "com.apple.applicationaccess",
								},
							},
							"payloadDisplayName": "Updates",
						}),
					},
					{
						Identifier: "com.jamf.ddm.disk-management",
						Configuration: marshalConfig(map[string]interface{}{
							"Restrictions": map[string]interface{}{
								"ExternalStorage": map[string]interface{}{
									"Included": true,
									"Value":    "ReadOnly",
								},
								"NetworkStorage": map[string]interface{}{
									"Included": true,
									"Value":    "Disallowed",
								},
							},
							"version": 2,
						}),
					},
				},
			},
		},
	}

	// Print the JSON request before sending
	jsonReq, err := json.MarshalIndent(request, "", "  ")
	if err != nil {
		log.Printf("Error marshaling request to JSON: %v", err)
	} else {
		fmt.Print("\n" + strings.Repeat("-", 50) + "\n")
		fmt.Printf("Blueprint Create Request (JSON):\n")
		fmt.Print(strings.Repeat("-", 50) + "\n")
		fmt.Println(string(jsonReq))
		fmt.Print(strings.Repeat("-", 50) + "\n\n")
	}

	// Initialize the client (baseURL-based)
	apiClient := client.NewClient(baseURL, clientID, clientSecret)

	// Create the blueprint
	blueprint, err := apiClient.CreateBlueprintV1(context.Background(), request)
	if err != nil {
		log.Fatalf("Error creating blueprint: %v", err)
	}

	fmt.Printf("Blueprint created successfully!\n")
	fmt.Printf("ID: %s\n", blueprint.ID)
	fmt.Printf("Href: %s\n", blueprint.Href)

	// Print the full JSON response
	fmt.Print("\n" + strings.Repeat("=", 50) + "\n")
	fmt.Printf("Full JSON Response:\n")
	fmt.Print(strings.Repeat("=", 50) + "\n")

	jsonData, err := json.MarshalIndent(blueprint, "", "  ")
	if err != nil {
		log.Printf("Error marshaling to JSON: %v", err)
	} else {
		fmt.Println(string(jsonData))
	}
}
