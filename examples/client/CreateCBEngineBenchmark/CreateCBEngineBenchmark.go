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
	region := "eu" // us, eu, or apac

	// Alternatively, use environment variables
	if envClientID := os.Getenv("JAMF_CLIENT_ID"); envClientID != "" {
		clientID = envClientID
	}
	if envClientSecret := os.Getenv("JAMF_CLIENT_SECRET"); envClientSecret != "" {
		clientSecret = envClientSecret
	}
	if envRegion := os.Getenv("JAMF_REGION"); envRegion != "" {
		region = envRegion
	}

	if clientID == "" || clientSecret == "" || region == "" {
		log.Fatal("Missing required configuration: JAMF_CLIENT_ID, JAMF_CLIENT_SECRET, JAMF_REGION")
	}

	// Example: create a new benchmark (customize as needed)
	request := &client.CBEngineBenchmarkRequest{
		Title:            "Test2",
		Description:      "Test",
		SourceBaselineID: "cis_lvl1",
		Sources: []client.CBEngineSource{
			{Branch: "sonoma", Revision: "acd8e3069260029475b15a1e964db89e7bfa01a0"},
			{Branch: "sequoia", Revision: "0b4d809ae043ca16e10a595c1f87c4d9fe21eb4a"},
			{Branch: "ventura", Revision: "c204a2b3c8e6fa948dde8883f8fd0df971183223"},
		},
		Rules: []client.CBEngineRuleRequest{
			{ID: "system_settings_software_update_download_enforce", Enabled: true},
		},
		Target: client.CBEngineTarget{
			DeviceGroups: []string{"4a36a1fe-e45a-430d-a966-a4d3ac993577"},
		},
		EnforcementMode: "MONITOR_AND_ENFORCE",
	}

	// Optionally, load request from a JSON file (uncomment to use)
	// file, err := os.Open("create_benchmark_request.json")
	// if err == nil {
	// 	defer file.Close()
	// 	json.NewDecoder(file).Decode(request)
	// }

	// Initialize the client (region-based)
	apiClient := client.NewCBEngineClient(region, clientID, clientSecret)

	// Print the JSON request before sending
	jsonReq, err := json.MarshalIndent(request, "", "  ")
	if err != nil {
		log.Printf("Error marshaling request to JSON: %v", err)
	} else {
		fmt.Print("\n" + strings.Repeat("-", 50) + "\n")
		fmt.Printf("Benchmark Create Request (JSON):\n")
		fmt.Print(strings.Repeat("-", 50) + "\n")
		fmt.Println(string(jsonReq))
		fmt.Print(strings.Repeat("-", 50) + "\n\n")
	}

	// Create the benchmark
	benchmark, err := apiClient.CreateCBEngineBenchmark(context.Background(), request)
	if err != nil {
		log.Fatalf("Error creating benchmark: %v", err)
	}

	fmt.Printf("Benchmark created successfully!\n")
	fmt.Printf("ID: %s\n", benchmark.BenchmarkID)
	fmt.Printf("Title: %s\n", benchmark.Title)
	fmt.Printf("Description: %s\n", benchmark.Description)
	fmt.Printf("Enforcement Mode: %s\n", benchmark.EnforcementMode)
	fmt.Printf("Update Available: %t\n", benchmark.UpdateAvailable)
	fmt.Printf("Last Updated: %s\n", benchmark.LastUpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Target Device Groups: %v\n", benchmark.Target.DeviceGroups)

	// Print the full JSON response
	fmt.Print("\n" + strings.Repeat("=", 50) + "\n")
	fmt.Printf("Full JSON Response:\n")
	fmt.Print(strings.Repeat("=", 50) + "\n")

	jsonData, err := json.MarshalIndent(benchmark, "", "  ")
	if err != nil {
		log.Printf("Error marshaling to JSON: %v", err)
	} else {
		fmt.Println(string(jsonData))
	}
}
