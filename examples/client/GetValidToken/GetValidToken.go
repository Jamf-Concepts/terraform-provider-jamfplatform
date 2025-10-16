package main

// Copyright 2025 Jamf Software LLC.

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

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

	fmt.Println("Testing OAuth Token Retrieval")
	fmt.Println("==============================")
	fmt.Printf("Base URL: %s\n", baseURL)
	fmt.Printf("Client ID: %s\n\n", maskString(clientID))

	// Initialize the client
	apiClient := client.NewClient(baseURL, clientID, clientSecret)

	// Get OAuth client
	oauthClient := apiClient.OAuthClient()

	// Test 1: Get a valid token
	fmt.Println("Test 1: Getting OAuth Token")
	fmt.Println("----------------------------")

	start := time.Now()
	token, err := oauthClient.GetValidToken(context.Background())
	elapsed := time.Since(start)

	if err != nil {
		log.Fatalf("Failed to get token: %v", err)
	}

	fmt.Printf("Token retrieved successfully in %v\n", elapsed)
	fmt.Printf("Token (masked): %s...%s\n", token[:10], token[len(token)-10:])
	fmt.Printf("Token length: %d characters\n\n", len(token))

	// Test 2: Get token info
	fmt.Println("Test 2: Token Information")
	fmt.Println("-------------------------")

	accessToken, expiresAt, isValid := oauthClient.GetTokenInfo()

	fmt.Printf("Access Token (masked): %s...%s\n", accessToken[:10], accessToken[len(accessToken)-10:])
	fmt.Printf("Expires At: %s\n", expiresAt.Format(time.RFC3339))
	fmt.Printf("Time Until Expiry: %v\n", time.Until(expiresAt))
	fmt.Printf("Is Valid: %v\n\n", isValid)

	// Test 3: Check granted scopes
	fmt.Println("Test 3: Granted Scopes")
	fmt.Println("----------------------")

	scopes := oauthClient.GetGrantedScopes()
	if len(scopes) > 0 {
		fmt.Printf("Granted Scopes (%d):\n", len(scopes))
		for i, scope := range scopes {
			fmt.Printf("  %d. %s\n", i+1, scope)
		}
	} else {
		fmt.Println("No specific scopes granted (full access)")
	}
	fmt.Println()

	// Test 4: Token validation check
	fmt.Println("Test 4: Token Validation Check")
	fmt.Println("-------------------------------")

	if oauthClient.IsTokenValid() {
		fmt.Println("Token is currently valid")
	} else {
		fmt.Println("Token is expired or not present")
	}
	fmt.Println()

	// Test 5: Token reuse (should use cached token)
	fmt.Println("Test 5: Token Reuse (Cached)")
	fmt.Println("----------------------------")

	start = time.Now()
	token2, err := oauthClient.GetValidToken(context.Background())
	elapsed = time.Since(start)

	if err != nil {
		log.Fatalf("Failed to get token: %v", err)
	}

	if token == token2 {
		fmt.Printf("Same token returned (cached) in %v\n", elapsed)
		fmt.Println(" (Should be < 1ms if properly cached)")
	} else {
		fmt.Printf("Different token returned in %v\n", elapsed)
	}
	fmt.Println()

	// Print summary as JSON
	fmt.Println("Summary (JSON)")
	fmt.Println("==============")

	summary := map[string]interface{}{
		"base_url":          baseURL,
		"token_retrieved":   true,
		"token_length":      len(token),
		"token_expires_at":  expiresAt.Format(time.RFC3339),
		"token_valid":       isValid,
		"time_until_expiry": time.Until(expiresAt).String(),
		"granted_scopes":    scopes,
		"scope_count":       len(scopes),
		"cached_correctly":  token == token2,
	}

	jsonData, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		log.Printf("Error marshaling to JSON: %v", err)
	} else {
		fmt.Println(string(jsonData))
	}

	fmt.Println("\nAll OAuth tests completed successfully!")
}

// maskString masks a string for display, showing only first 4 and last 4 characters
func maskString(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}
