// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package provider_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// scopeProbeConfig is a provider block plus the cheapest data source that takes
// no required arguments, so Terraform has a reason to configure the provider.
// Neither test reaches the read: one fails in provider Configure and the other in
// the data source's Configure.
const scopeProbeConfig = `
provider "jamfplatform" {
%s
}

data "jamfplatform_device_groups" "scope_probe" {}
`

// TestAccProviderScope_ConflictingScopesRejected pins the mutual exclusion of
// environment_id and tenant_id against a real Terraform run.
//
// It needs no environment-scoped integration, so it is the one scope test that
// earns its place in CI before the Platform API GA: the IDs never reach the wire.
// resolveScope runs before ValidateCredentials in provider Configure, so the
// conflict is caught with whatever credentials the run already has, under
// whatever scope those credentials carry.
//
// The regex matches only the diagnostic summary. Terraform hard-wraps detail text
// at roughly 80 columns, which breaks any regex spanning more than a few words;
// the detail's wording is asserted by the unit tests in scope_test.go instead.
func TestAccProviderScope_ConflictingScopesRejected(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testhelpers.AccPreCheck(t) },
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(scopeProbeConfig, `
  environment_id = "11111111-1111-4111-8111-111111111111"
  tenant_id      = "22222222-2222-4222-8222-222222222222"`),
				ExpectError: regexp.MustCompile(`Conflicting API Integration Scope`),
			},
		},
	})
}

// TestAccProviderScope_OrganizationScopeRejectedPerConstruct verifies the
// per-construct gate end-to-end: an integration carrying no scope at all
// configures the provider successfully, then fails on the first construct that
// needs a scope.
//
// This is the only test that exercises the organization-scope rejection path,
// and it will keep being the only one until the organization-level constructs
// exist. It runs in CI today because it needs credentials but deliberately no
// scope — both scope variables are cleared for the duration, which is also why
// it cannot use testhelpers.AccPreCheck (that helper skips when neither is set).
func TestAccProviderScope_OrganizationScopeRejectedPerConstruct(t *testing.T) {
	accPreCheckCredentialsOnly(t)

	// Cleared before the provider is configured, so resolveScope sees neither
	// variable and the client is built with no scope header at all.
	t.Setenv("JAMFPLATFORM_ENVIRONMENT_ID", "")
	t.Setenv("JAMFPLATFORM_TENANT_ID", "")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.AccTestProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(scopeProbeConfig, ""),
				ExpectError: regexp.MustCompile(`Unsupported API Integration Scope`),
			},
		},
	})
}

// accPreCheckCredentialsOnly gates on the credentials alone, without the scope
// requirement testhelpers.AccPreCheck adds. Only the organization-scope test
// wants this: every other acceptance test needs a scope to reach anything.
func accPreCheckCredentialsOnly(t *testing.T) {
	t.Helper()

	for _, key := range []string{
		"JAMFPLATFORM_BASE_URL",
		"JAMFPLATFORM_CLIENT_ID",
		"JAMFPLATFORM_CLIENT_SECRET",
	} {
		if os.Getenv(key) == "" {
			t.Skipf("%s must be set for acceptance tests", key)
		}
	}

	t.Setenv("TF_ACC", "1")
}
