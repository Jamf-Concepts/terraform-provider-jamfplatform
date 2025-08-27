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

	// Get benchmark title from command line argument or environment variable
	var benchmarkTitle string
	if len(os.Args) > 1 {
		benchmarkTitle = os.Args[1]
	} else if envBenchmarkTitle := os.Getenv("BENCHMARK_TITLE"); envBenchmarkTitle != "" {
		benchmarkTitle = envBenchmarkTitle
	} else {
		log.Fatal("Please provide a benchmark title as a command line argument or set BENCHMARK_TITLE environment variable")
	}

	// Initialize the client (baseURL-based)
	apiClient := client.NewClient(baseURL, clientID, clientSecret)

	// Get specific benchmark by title
	benchmark, err := apiClient.GetCBEngineBenchmarkByTitle(context.Background(), benchmarkTitle)
	if err != nil {
		log.Fatalf("Error getting benchmark '%s': %v", benchmarkTitle, err)
	}

	fmt.Printf("Benchmark Details:\n")
	fmt.Printf("ID: %s\n", benchmark.BenchmarkID)
	fmt.Printf("Tenant ID: %s\n", benchmark.TenantID)
	fmt.Printf("Title: %s\n", benchmark.Title)
	fmt.Printf("Description: %s\n", benchmark.Description)
	fmt.Printf("Enforcement Mode: %s\n", benchmark.EnforcementMode)
	fmt.Printf("Deleted: %t\n", benchmark.Deleted)
	fmt.Printf("Update Available: %t\n", benchmark.UpdateAvailable)
	fmt.Printf("Last Updated: %s\n", benchmark.LastUpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Target Device Groups: %v\n", benchmark.Target.DeviceGroups)

	fmt.Printf("\nSources (%d):\n", len(benchmark.Sources))
	for i, source := range benchmark.Sources {
		fmt.Printf("  %d. Branch: %s, Revision: %s\n", i+1, source.Branch, source.Revision)
	}

	fmt.Printf("\nRules (%d):\n", len(benchmark.Rules))
	for i, rule := range benchmark.Rules {
		status := "Disabled"
		if rule.Enabled {
			status = "Enabled"
		}

		odvInfo := ""
		if rule.ODV != nil {
			odvInfo = fmt.Sprintf(" (ODV: %s)", rule.ODV.Value)
		}

		fmt.Printf("  %d. %s [%s] - %s%s\n",
			i+1,
			rule.ID,
			status,
			rule.Title,
			odvInfo,
		)

		if len(rule.SupportedOS) > 0 {
			fmt.Printf("      Supported OS: ")
			for j, os := range rule.SupportedOS {
				if j > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("%s v%d (%s)", os.OSType, os.OSVersion, os.ManagementType)
			}
			fmt.Printf("\n")
		}
	}

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
