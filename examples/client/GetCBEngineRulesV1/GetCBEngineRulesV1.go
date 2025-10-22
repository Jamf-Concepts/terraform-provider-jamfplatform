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

	// Get baseline ID from command line argument or environment variable
	var baselineID string
	if len(os.Args) > 1 {
		baselineID = os.Args[1]
	} else if envBaselineID := os.Getenv("BASELINE_ID"); envBaselineID != "" {
		baselineID = envBaselineID
	} else {
		log.Fatal("Please provide a baseline ID as a command line argument or set BASELINE_ID environment variable")
	}

	// Initialize the client (baseURL-based)
	apiClient := client.NewClient(baseURL, clientID, clientSecret)

	// Get rules for the given baseline
	rulesResp, err := apiClient.GetCBEngineRulesV1(context.Background(), baselineID)
	if err != nil {
		log.Fatalf("Error getting rules for baseline %s: %v", baselineID, err)
	}

	fmt.Printf("Found %d rule(s) for baseline %s\n\n", len(rulesResp.Rules), baselineID)

	for _, rule := range rulesResp.Rules {
		status := "Disabled"
		if rule.Enabled {
			status = "Enabled"
		}
		odvInfo := ""
		if rule.ODV != nil {
			odvInfo = fmt.Sprintf(" (ODV: %s)", rule.ODV.Value)
		}
		fmt.Printf("Rule: %s\nTitle: %s\nStatus: %s%s\nDescription: %s\n\n",
			rule.ID,
			rule.Title,
			status,
			odvInfo,
			rule.Description,
		)
	}

	// Print the full JSON response
	fmt.Print("\n" + strings.Repeat("=", 50) + "\n")
	fmt.Printf("Full JSON Response:\n")
	fmt.Print(strings.Repeat("=", 50) + "\n")

	jsonData, err := json.MarshalIndent(rulesResp, "", "  ")
	if err != nil {
		log.Printf("Error marshaling to JSON: %v", err)
	} else {
		fmt.Println(string(jsonData))
	}
}
