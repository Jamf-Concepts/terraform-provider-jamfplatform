// Copyright 2025 Jamf Software LLC.

package main

import (
	"context"
	"fmt"
	"log"
	"os"

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

	// Get benchmark ID from command line argument or environment variable
	var benchmarkID string
	if len(os.Args) > 1 {
		benchmarkID = os.Args[1]
	} else if envBenchmarkID := os.Getenv("BENCHMARK_ID"); envBenchmarkID != "" {
		benchmarkID = envBenchmarkID
	} else {
		log.Fatal("Please provide a benchmark ID as a command line argument or set BENCHMARK_ID environment variable")
	}

	// Initialize the client (region-based)
	apiClient := client.NewClient(region, clientID, clientSecret)

	// Delete the benchmark
	err := apiClient.DeleteCBEngineBenchmark(context.Background(), benchmarkID)
	if err != nil {
		log.Fatalf("Error deleting benchmark %s: %v", benchmarkID, err)
	}

	fmt.Printf("Benchmark %s deleted successfully.\n", benchmarkID)
}
