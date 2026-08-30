// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package testhelpers

import (
	"context"
	"os"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/aigovernance"
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

// AccPreCheckAIGovernance gates the Jamf AI Governance acceptance tests.
//
// Like the Security Cloud gate, it requires the operator to *declare* that the configured scope
// belongs to a tenant holding AI Governance, and skips otherwise: an environment without the surface
// is a legitimate acceptance environment, not a failure, and the alternative — inferring entitlement
// from an empty policy list — would read a working tenant with no policies as an unentitled one.
//
// Environment scope only, and probed rather than assumed. AI Governance answers a request carrying
// no scope header with REQUEST_CONTEXT_NOT_PROVIDED, and one carrying X-Tenant-Id with
// BAD_PERMISSIONS — while a Jamf Pro request presenting the same header succeeds, so the header
// itself is accepted and the refusal belongs to the namespace. That code is indistinguishable from a
// real privilege gap, so the namespace is presumed unrouted under tenant scope and there is no
// tenant form to declare.
func AccPreCheckAIGovernance(t *testing.T) {
	t.Helper()
	AccPreCheck(t)

	declared := os.Getenv("JAMFPLATFORM_AI_GOVERNANCE_ENVIRONMENT_ID")
	if declared == "" {
		t.Skip("JAMFPLATFORM_AI_GOVERNANCE_ENVIRONMENT_ID must be set to declare that the configured environment holds Jamf AI Governance")
	}
	if configured := os.Getenv("JAMFPLATFORM_ENVIRONMENT_ID"); configured != declared {
		t.Skipf("JAMFPLATFORM_AI_GOVERNANCE_ENVIRONMENT_ID (%s) does not match the configured JAMFPLATFORM_ENVIRONMENT_ID (%s); the provider is not scoped to the declared AI Governance environment", declared, configured)
	}
}

// RequireAIGovernanceTool returns the named tool from Jamf's AI tool catalogue, carrying its current
// settings schema version and every version it still accepts.
//
// The two ways this can come up short are deliberately not the same outcome.
//
// A tool the catalogue does not list SKIPS. The catalogue is Jamf's, not the tenant's — an
// environment only offers the tools Jamf has shipped to it, and there is nothing an acceptance test
// can create to change that, so a tool that is simply absent is a platform capability gap rather
// than a defect.
//
// A catalogue read that FAILS is a defect and fails the test. AccPreCheckAIGovernance has already
// required the operator to *declare* that this environment holds AI Governance, and the whole point
// of that declaration is that entitlement is declared rather than inferred from a read. Skipping on
// a failed read would put the inference straight back: a regression in the namespace path, in the
// scope header, or in the SDK's own catalogue call would take every AI Governance test with it while
// the run still reported green — the exact shape of a broken environment hiding behind a skip.
func RequireAIGovernanceTool(t *testing.T, toolID string) aigovernance.ToolSummary {
	t.Helper()

	response, err := aigovernance.New(NewAcceptanceClient(t)).ListTools(context.Background())
	if err != nil {
		t.Fatalf("Failed to read the AI tool catalogue: %v; this environment was declared to hold Jamf AI Governance, so a failed catalogue read is a defect rather than an environment condition", err)
	}

	for _, tool := range response.Results {
		if tool.ID == toolID {
			return tool
		}
	}

	t.Skipf("Skipping: this environment does not offer the AI tool %s", toolID)
	return aigovernance.ToolSummary{}
}

// AccTenantIDOrSkip returns the tenant ID the provider is configured with, or
// skips when the run is environment-scoped.
//
// For the handful of assertions that need to compare a value the tenant reports
// against the scope the provider was pointed at. Under an environment scope
// there is nothing local to compare with — the provider sends an environment ID
// and the gateway resolves the tenant server-side — so the honest outcome is a
// skip rather than an assertion built on a value the test cannot know.
func AccTenantIDOrSkip(t *testing.T) string {
	t.Helper()

	tenant := os.Getenv("JAMFPLATFORM_TENANT_ID")
	if tenant == "" {
		t.Skip("JAMFPLATFORM_TENANT_ID must be set for this test; an environment-scoped run has no locally known tenant ID to compare against")
	}
	return tenant
}
