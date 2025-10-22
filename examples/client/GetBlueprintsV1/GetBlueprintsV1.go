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

	// Initialize the Nebula Blueprint client
	apiClient := client.NewClient(baseURL, clientID, clientSecret)

	// Get all blueprints (pagination handled automatically)
	blueprints, err := apiClient.GetBlueprintsV1(context.Background(), nil, "")
	if err != nil {
		log.Fatalf("Error getting blueprints: %v", err)
	}

	fmt.Printf("Found %d blueprint(s)\n\n", len(blueprints))

	for _, blueprint := range blueprints {
		fmt.Printf("Blueprint Details:\n"+
			"ID: %s\n"+
			"Name: %s\n"+
			"Description: %s\n"+
			"Created: %s\n"+
			"Updated: %s\n"+
			"Deployment State: %s\n\n",
			blueprint.ID,
			blueprint.Name,
			blueprint.Description,
			blueprint.Created,
			blueprint.Updated,
			blueprint.DeploymentState.State,
		)
	}

	// Print the full JSON response
	fmt.Print("\n" + strings.Repeat("=", 50) + "\n")
	fmt.Printf("Full JSON Response:\n")
	fmt.Print(strings.Repeat("=", 50) + "\n")

	jsonData, err := json.MarshalIndent(blueprints, "", "  ")
	if err != nil {
		log.Printf("Error marshaling to JSON: %v", err)
	} else {
		fmt.Println(string(jsonData))
	}
}
