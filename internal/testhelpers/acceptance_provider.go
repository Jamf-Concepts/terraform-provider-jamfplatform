// Copyright 2026 Jamf Software LLC.

//go:build acceptance

package testhelpers

import (
	"os"
	"testing"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// AccTestProtoV6ProviderFactories returns the provider factories for Terraform acceptance tests.
var AccTestProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"jamfplatform": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// AccPreCheck validates that the required environment variables are set before running
// acceptance tests and ensures TF_ACC is set so terraform-plugin-testing runs the tests.
func AccPreCheck(t *testing.T) {
	t.Helper()

	if v := os.Getenv("JAMFPLATFORM_BASE_URL"); v == "" {
		t.Fatal("JAMFPLATFORM_BASE_URL must be set for acceptance tests")
	}
	if v := os.Getenv("JAMFPLATFORM_CLIENT_ID"); v == "" {
		t.Fatal("JAMFPLATFORM_CLIENT_ID must be set for acceptance tests")
	}
	if v := os.Getenv("JAMFPLATFORM_CLIENT_SECRET"); v == "" {
		t.Fatal("JAMFPLATFORM_CLIENT_SECRET must be set for acceptance tests")
	}

	os.Setenv("TF_ACC", "1")
}
