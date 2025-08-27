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
	if envbaseURL := os.Getenv("JAMF_BASE_URL"); envbaseURL != "" {
		baseURL = envbaseURL
	}

	if clientID == "" || clientSecret == "" || baseURL == "" {
		log.Fatal("Missing required configuration: JAMF_CLIENT_ID, JAMF_CLIENT_SECRET, JAMF_BASE_URL")
	}

	// Get blueprint name from command line argument or environment variable
	var blueprintName string
	if len(os.Args) > 1 {
		blueprintName = os.Args[1]
	} else if envBlueprintName := os.Getenv("BLUEPRINT_NAME"); envBlueprintName != "" {
		blueprintName = envBlueprintName
	} else {
		log.Fatal("Please provide a blueprint name as a command line argument or set BLUEPRINT_NAME environment variable")
	}

	// Initialize the Nebula Blueprint client
	apiClient := client.NewClient(baseURL, clientID, clientSecret)

	// Get specific blueprint by name
	blueprint, err := apiClient.GetBlueprintByName(context.Background(), blueprintName)
	if err != nil {
		log.Fatalf("Error getting blueprint by name '%s': %v", blueprintName, err)
	}

	fmt.Printf("Blueprint Details:\n")
	fmt.Printf("ID: %s\n", blueprint.ID)
	fmt.Printf("Name: %s\n", blueprint.Name)
	fmt.Printf("Description: %s\n", blueprint.Description)
	fmt.Printf("Scope: %v\n", blueprint.Scope)
	fmt.Printf("Created: %s\n", blueprint.Created)
	fmt.Printf("Updated: %s\n", blueprint.Updated)
	fmt.Printf("Deployment State: %s\n", blueprint.DeploymentState.State)

	if blueprint.DeploymentState.LastDeployment != nil {
		fmt.Printf("Last Deployment Started: %s\n", blueprint.DeploymentState.LastDeployment.Started)
		fmt.Printf("Last Deployment State: %s\n", blueprint.DeploymentState.LastDeployment.State)
	}

	fmt.Printf("\nSteps (%d):\n", len(blueprint.Steps))
	for i, step := range blueprint.Steps {
		fmt.Printf("  %d. %s\n", i+1, step.Name)
		for j, comp := range step.Components {
			fmt.Printf("      %d. Identifier: %s, Configuration: %s\n", j+1, comp.Identifier, string(comp.Configuration))
		}
	}

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
