// Copyright 2025 Jamf Software LLC.

package main

import (
	"context"
	"log"

	jamfplatform "github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	err := providerserver.Serve(context.Background(), jamfplatform.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/Jamf-Concepts/jamfplatform",
	})
	if err != nil {
		log.Fatalf("Error serving provider: %v", err)
	}
}
