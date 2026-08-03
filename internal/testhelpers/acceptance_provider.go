// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

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
		t.Skip("JAMFPLATFORM_BASE_URL must be set for acceptance tests")
	}
	if v := os.Getenv("JAMFPLATFORM_CLIENT_ID"); v == "" {
		t.Skip("JAMFPLATFORM_CLIENT_ID must be set for acceptance tests")
	}
	if v := os.Getenv("JAMFPLATFORM_CLIENT_SECRET"); v == "" {
		t.Skip("JAMFPLATFORM_CLIENT_SECRET must be set for acceptance tests")
	}
	if v := os.Getenv("JAMFPLATFORM_TENANT_ID"); v == "" {
		t.Skip("JAMFPLATFORM_TENANT_ID must be set for acceptance tests")
	}

	if err := os.Setenv("TF_ACC", "1"); err != nil {
		t.Fatalf("setting TF_ACC: %v", err)
	}
}

// AccPreCheckOffline is the precheck for acceptance tests that need no tenant —
// provider-defined functions, which run offline with no API client or provider
// configuration. It sets TF_ACC (so terraform-plugin-testing actually runs the
// test) without gating on the JAMFPLATFORM_* credentials that AccPreCheck
// requires. Use this instead of AccPreCheck in function acceptance tests: it
// keeps them runnable under the raw `go test -tags=acceptance ./...` command
// (see TESTING.md), which does not set TF_ACC itself, so they neither skip
// silently nor demand credentials they don't use.
func AccPreCheckOffline(t *testing.T) {
	t.Helper()
	t.Setenv("TF_ACC", "1")
}
