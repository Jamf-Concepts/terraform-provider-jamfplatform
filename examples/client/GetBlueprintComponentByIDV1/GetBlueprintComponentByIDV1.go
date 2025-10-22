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
		fmt.Fprintf(os.Stderr, "Usage: %s <component-identifier>\n", os.Args[0])
		os.Exit(1)
	}
	identifier := os.Args[1]

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

	// Initialize the client (baseURL-based)
	apiClient := client.NewClient(baseURL, clientID, clientSecret)

	// Get blueprint component by identifier
	component, err := apiClient.GetBlueprintComponentByIDV1(context.Background(), identifier)
	if err != nil {
		log.Fatalf("Error getting blueprint component '%s': %v", identifier, err)
	}

	fmt.Printf("Component Details:\n"+
		"Identifier: %s\n"+
		"Name: %s\n"+
		"Description: %s\n",
		component.Identifier,
		component.Name,
		component.Description,
	)
	fmt.Printf("Supported OS:\n")
	for osFamily, versions := range component.Meta.SupportedOs {
		fmt.Printf("  %s: ", osFamily)
		var verList []string
		for _, v := range versions {
			verList = append(verList, v.Version)
		}
		fmt.Printf("%s\n", strings.Join(verList, ", "))
	}
	fmt.Println()

	// Print the full JSON response
	fmt.Print("\n" + strings.Repeat("=", 50) + "\n")
	fmt.Printf("Full JSON Response:\n")
	fmt.Print(strings.Repeat("=", 50) + "\n")

	jsonData, err := json.MarshalIndent(component, "", "  ")
	if err != nil {
		log.Printf("Error marshaling to JSON: %v", err)
	} else {
		fmt.Println(string(jsonData))
	}
}
