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

	// Initialize the client (baseURL-based)
	apiClient := client.NewClient(baseURL, clientID, clientSecret)

	// Get all benchmarks
	benchmarks, err := apiClient.GetCBEngineBenchmarks(context.Background())
	if err != nil {
		log.Fatalf("Error getting benchmarks: %v", err)
	}

	fmt.Printf("Found %d benchmark(s)\n\n", len(benchmarks.Benchmarks))

	for _, benchmark := range benchmarks.Benchmarks {
		fmt.Printf("Benchmark Details:\n"+
			"ID: %s\n"+
			"Title: %s\n"+
			"Description: %s\n"+
			"Sync State: %s\n"+
			"Update Available: %t\n"+
			"Target Device Groups: %v\n\n",
			benchmark.ID,
			benchmark.Title,
			benchmark.Description,
			benchmark.SyncState,
			benchmark.UpdateAvailable,
			benchmark.Target.DeviceGroups,
		)
	}

	// Print the full JSON response
	fmt.Print("\n" + strings.Repeat("=", 50) + "\n")
	fmt.Printf("Full JSON Response:\n")
	fmt.Print(strings.Repeat("=", 50) + "\n")

	jsonData, err := json.MarshalIndent(benchmarks, "", "  ")
	if err != nil {
		log.Printf("Error marshaling to JSON: %v", err)
	} else {
		fmt.Println(string(jsonData))
	}
}
