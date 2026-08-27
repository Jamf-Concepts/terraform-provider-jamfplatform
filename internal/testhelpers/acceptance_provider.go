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
	if os.Getenv("JAMFPLATFORM_ENVIRONMENT_ID") == "" && os.Getenv("JAMFPLATFORM_TENANT_ID") == "" {
		t.Skip("one of JAMFPLATFORM_ENVIRONMENT_ID or JAMFPLATFORM_TENANT_ID must be set for acceptance tests")
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

// AccPreCheckSecurityCloud is the precheck for Jamf Security Cloud acceptance
// tests. It runs AccPreCheck, then requires the operator to have *declared* that
// the scope the provider is configured with belongs to a Security Cloud tenant,
// via JAMFPLATFORM_SECURITY_CLOUD_ENVIRONMENT_ID or
// JAMFPLATFORM_SECURITY_CLOUD_TENANT_ID. The declared ID must equal the
// JAMFPLATFORM_* scope actually in use; anything else skips.
//
// Why a declaration rather than a probe: Security Cloud is a separate
// entitlement, so a perfectly valid acceptance tenant can hold Jamf Pro and not
// hold this. Pointing the suite at such a tenant must skip these tests, not fail
// them. Requiring the ID to match the configured scope is what makes the
// declaration load-bearing — a stale value left over from a different tenant
// skips instead of green-lighting a run against the wrong estate.
//
// It deliberately does not introduce a second credential set. One integration
// serves every construct in this provider, and a parallel Security Cloud
// credential path would mean every Security Cloud acceptance config had to emit
// its own provider block, diverging from the rest of the suite for no gain.
func AccPreCheckSecurityCloud(t *testing.T) {
	t.Helper()
	AccPreCheck(t)

	declaredEnvironment := os.Getenv("JAMFPLATFORM_SECURITY_CLOUD_ENVIRONMENT_ID")
	declaredTenant := os.Getenv("JAMFPLATFORM_SECURITY_CLOUD_TENANT_ID")
	if declaredEnvironment == "" && declaredTenant == "" {
		t.Skip("one of JAMFPLATFORM_SECURITY_CLOUD_ENVIRONMENT_ID or JAMFPLATFORM_SECURITY_CLOUD_TENANT_ID must be set to declare that the configured scope is a Jamf Security Cloud one")
	}
	if declaredEnvironment != "" && declaredTenant != "" {
		t.Skip("JAMFPLATFORM_SECURITY_CLOUD_ENVIRONMENT_ID and JAMFPLATFORM_SECURITY_CLOUD_TENANT_ID are both set; unset one so the declared Security Cloud scope is unambiguous")
	}

	if declaredEnvironment != "" {
		if configured := os.Getenv("JAMFPLATFORM_ENVIRONMENT_ID"); configured != declaredEnvironment {
			t.Skipf("JAMFPLATFORM_SECURITY_CLOUD_ENVIRONMENT_ID (%s) does not match the configured JAMFPLATFORM_ENVIRONMENT_ID (%s); the provider is not scoped to the declared Security Cloud environment", declaredEnvironment, configured)
		}
		return
	}
	if configured := os.Getenv("JAMFPLATFORM_TENANT_ID"); configured != declaredTenant {
		t.Skipf("JAMFPLATFORM_SECURITY_CLOUD_TENANT_ID (%s) does not match the configured JAMFPLATFORM_TENANT_ID (%s); the provider is not scoped to the declared Security Cloud tenant", declaredTenant, configured)
	}
}
