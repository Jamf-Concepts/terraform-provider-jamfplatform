// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package providerdata

import (
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
)

// TestNewDerivesScopeFromClient covers the round-trip New relies on: the scope
// option the SDK client was built with must come back out of Client.Scope() as
// the matching ScopeKind, so the gate can never disagree with the header the
// client actually sends.
func TestNewDerivesScopeFromClient(t *testing.T) {
	tests := []struct {
		name string
		opts []jamfplatform.Option
		want ScopeKind
	}{
		{
			name: "no scope option is organization scope",
			want: ScopeOrganization,
		},
		{
			name: "environment option",
			opts: []jamfplatform.Option{jamfplatform.WithEnvironmentID("e-1")},
			want: ScopeEnvironment,
		},
		{
			name: "tenant option",
			opts: []jamfplatform.Option{jamfplatform.WithTenantID("t-1")},
			want: ScopeTenant,
		},
		{
			name: "an empty ID is not a scope",
			opts: []jamfplatform.Option{jamfplatform.WithTenantID("")},
			want: ScopeOrganization,
		},
		{
			name: "both options set follows the SDK, where environment wins",
			opts: []jamfplatform.Option{jamfplatform.WithTenantID("t-1"), jamfplatform.WithEnvironmentID("e-1")},
			want: ScopeEnvironment,
		},
		{
			name: "both options set, reversed order, still environment",
			opts: []jamfplatform.Option{jamfplatform.WithEnvironmentID("e-1"), jamfplatform.WithTenantID("t-1")},
			want: ScopeEnvironment,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := jamfplatform.NewClient("http://127.0.0.1:1", "test-id", "test-secret", tc.opts...)
			if got := New(c).Scope(); got != tc.want {
				t.Errorf("derived scope: got %v, want %v", got, tc.want)
			}
		})
	}
}

// TestNewNilClient guards the nil path: scopeFromClient must not dereference a
// nil client, and the result has to be the scope that gates everything rather
// than one that passes.
func TestNewNilClient(t *testing.T) {
	if got := New(nil).Scope(); got != ScopeOrganization {
		t.Errorf("nil client: got %v, want ScopeOrganization", got)
	}
}

// TestRequireScope covers the gate every construct's Configure runs through:
// the configured scope must appear in the allowed set, and an organization-scoped
// credential is rejected everywhere until the account-level constructs exist.
func TestRequireScope(t *testing.T) {
	tests := []struct {
		name       string
		configured ScopeKind
		allowed    []ScopeKind
		wantError  bool
	}{
		{
			name:       "tenant satisfies tenant-or-environment",
			configured: ScopeTenant,
			allowed:    []ScopeKind{ScopeEnvironment, ScopeTenant},
		},
		{
			name:       "environment satisfies tenant-or-environment",
			configured: ScopeEnvironment,
			allowed:    []ScopeKind{ScopeEnvironment, ScopeTenant},
		},
		{
			name:       "organization satisfies neither",
			configured: ScopeOrganization,
			allowed:    []ScopeKind{ScopeEnvironment, ScopeTenant},
			wantError:  true,
		},
		{
			name:       "tenant does not satisfy environment-only",
			configured: ScopeTenant,
			allowed:    []ScopeKind{ScopeEnvironment},
			wantError:  true,
		},
		{
			name:       "environment satisfies environment-only",
			configured: ScopeEnvironment,
			allowed:    []ScopeKind{ScopeEnvironment},
		},
		{
			name:       "an empty allowed set gates nothing",
			configured: ScopeOrganization,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pd := &Data{scope: tc.configured}
			diags := pd.RequireScope("jamfplatform_pro_test", tc.allowed...)
			if got := diags.HasError(); got != tc.wantError {
				t.Errorf("HasError: got %v, want %v (%v)", got, tc.wantError, diags)
			}
		})
	}
}

// TestRequireScope_NilReceiver verifies the gate is permissive for a Data value
// that was never configured, matching ImpactCache: the framework's early-lifecycle
// nil ProviderData is the caller's guard, not this one's error.
func TestRequireScope_NilReceiver(t *testing.T) {
	var pd *Data
	if diags := pd.RequireScope("jamfplatform_pro_test", ScopeTenant); diags.HasError() {
		t.Errorf("expected no diagnostics from a nil receiver, got %v", diags)
	}
	if got := pd.Scope(); got != ScopeOrganization {
		t.Errorf("Scope on nil receiver: got %v, want ScopeOrganization", got)
	}
}

// TestRequireScope_DiagnosticIsActionable pins the parts of the message a user
// needs: which construct, which scope it wants, which one is configured, and the
// 403 that follows from pointing the right header at the wrong credential.
func TestRequireScope_DiagnosticIsActionable(t *testing.T) {
	pd := &Data{scope: ScopeOrganization}
	diags := pd.RequireScope("jamfplatform_blueprints_blueprint", ScopeEnvironment)
	if !diags.HasError() {
		t.Fatal("expected an error for an organization-scoped credential")
	}
	err := diags.Errors()[0]
	if !strings.Contains(err.Summary(), "jamfplatform_blueprints_blueprint") {
		t.Errorf("summary does not name the construct: %s", err.Summary())
	}
	for _, want := range []string{
		"an environment-scoped integration",
		"an organization-scoped integration",
		"`environment_id`",
		"OWNERSHIP_FORBIDDEN",
	} {
		if !strings.Contains(err.Detail(), want) {
			t.Errorf("detail does not mention %q: %s", want, err.Detail())
		}
	}
	if strings.Contains(err.Detail(), "`tenant_id` in the provider block") {
		t.Errorf("environment-only requirement should not suggest tenant_id: %s", err.Detail())
	}
}

// TestScopeRequirementPhrasing guards the generated phrasing, which is assembled
// rather than written out and so is easy to break into "a tenant-scoped or
// credential".
func TestScopeRequirementPhrasing(t *testing.T) {
	tests := []struct {
		allowed []ScopeKind
		want    string
	}{
		{[]ScopeKind{ScopeEnvironment}, "an environment-scoped integration"},
		{[]ScopeKind{ScopeTenant}, "a tenant-scoped integration"},
		{[]ScopeKind{ScopeEnvironment, ScopeTenant}, "an environment-scoped or tenant-scoped integration"},
		{[]ScopeKind{ScopeTenant, ScopeEnvironment}, "a tenant-scoped or environment-scoped integration"},
	}
	for _, tc := range tests {
		if got := scopeRequirement(tc.allowed); got != tc.want {
			t.Errorf("scopeRequirement(%v): got %q, want %q", tc.allowed, got, tc.want)
		}
	}
}

// TestScopeKindString pins the names that reach the tflog field and the
// diagnostics, including the zero value.
func TestScopeKindString(t *testing.T) {
	tests := map[ScopeKind]string{
		ScopeOrganization: "organization",
		ScopeTenant:       "tenant",
		ScopeEnvironment:  "environment",
	}
	for kind, want := range tests {
		if got := kind.String(); got != want {
			t.Errorf("ScopeKind(%d).String(): got %q, want %q", kind, got, want)
		}
	}
}
