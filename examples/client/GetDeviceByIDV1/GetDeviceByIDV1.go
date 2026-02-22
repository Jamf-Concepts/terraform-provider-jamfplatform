// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

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
	clientID := "example-client-id"
	clientSecret := "example-client-secret"
	baseURL := "https://us.apigw.jamf.com"
	deviceID := os.Getenv("JAMF_DEVICE_ID")

	if env := os.Getenv("JAMF_CLIENT_ID"); env != "" {
		clientID = env
	}
	if env := os.Getenv("JAMF_CLIENT_SECRET"); env != "" {
		clientSecret = env
	}
	if env := os.Getenv("JAMF_BASE_URL"); env != "" {
		baseURL = env
	}

	if deviceID == "" {
		log.Fatal("Missing required configuration: JAMF_DEVICE_ID")
	}
	if clientID == "" || clientSecret == "" || baseURL == "" {
		log.Fatal("Missing required configuration: JAMF_CLIENT_ID, JAMF_CLIENT_SECRET, JAMF_BASE_URL")
	}

	apiClient := client.NewClient(baseURL, clientID, clientSecret)

	device, err := apiClient.GetDeviceByIDV1(context.Background(), deviceID)
	if err != nil {
		log.Fatalf("Error getting device %s: %v", deviceID, err)
	}

	fmt.Printf("Device %s (%s)\n", device.ID, device.Name)
	fmt.Printf("Managed: %t\nSupervised: %t\nEnrollment Type: %s\n\n",
		device.Managed, device.Supervised, device.EnrollmentType)

	if device.Hardware != nil {
		fmt.Printf("Hardware: %s %s (%s) Serial %s\n",
			device.Hardware.Make, device.Hardware.Model, device.Hardware.ModelIdentifier, device.Hardware.SerialNumber)
	}
	if device.OperatingSystem != nil {
		fmt.Printf("OS: %s %s (%s)\n",
			device.OperatingSystem.Name, device.OperatingSystem.Version, device.OperatingSystem.Build)
	}
	if device.Security != nil {
		fmt.Printf("Bootstrap Token: %s\n", device.Security.BootstrapTokenEscrowedStatus)
	}

	fmt.Print("\n" + strings.Repeat("=", 50) + "\n")
	fmt.Printf("Full JSON Response:\n")
	fmt.Print(strings.Repeat("=", 50) + "\n")

	b, err := json.MarshalIndent(device, "", "  ")
	if err != nil {
		log.Printf("Error marshaling to JSON: %v", err)
	} else {
		fmt.Println(string(b))
	}
}
