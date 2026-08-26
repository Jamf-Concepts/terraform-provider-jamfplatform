// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// TestResolveScope covers the whole precedence lattice: the provider block beats
// the environment, the two scopes never combine, and setting nothing anywhere is
// organization scope rather than an error.
func TestResolveScope(t *testing.T) {
	tests := []struct {
		name        string
		cfgTenant   string
		cfgEnv      string
		envTenant   string
		envEnv      string
		wantKind    providerdata.ScopeKind
		wantID      string
		wantError   bool
		wantWarning bool
	}{
		{
			name:     "nothing set anywhere is organization scope",
			wantKind: providerdata.ScopeOrganization,
		},
		{
			name:      "tenant from config",
			cfgTenant: "t-cfg",
			wantKind:  providerdata.ScopeTenant,
			wantID:    "t-cfg",
		},
		{
			name:     "environment from config",
			cfgEnv:   "e-cfg",
			wantKind: providerdata.ScopeEnvironment,
			wantID:   "e-cfg",
		},
		{
			name:      "tenant from environment variable",
			envTenant: "t-env",
			wantKind:  providerdata.ScopeTenant,
			wantID:    "t-env",
		},
		{
			name:     "environment from environment variable",
			envEnv:   "e-env",
			wantKind: providerdata.ScopeEnvironment,
			wantID:   "e-env",
		},
		{
			name:      "both in config is an error",
			cfgTenant: "t-cfg",
			cfgEnv:    "e-cfg",
			wantKind:  providerdata.ScopeOrganization,
			wantError: true,
		},
		{
			name:      "both in the environment is an error",
			envTenant: "t-env",
			envEnv:    "e-env",
			wantKind:  providerdata.ScopeOrganization,
			wantError: true,
		},
		{
			name:        "config tenant shadows the environment variable for the other scope",
			cfgTenant:   "t-cfg",
			envEnv:      "e-env",
			wantKind:    providerdata.ScopeTenant,
			wantID:      "t-cfg",
			wantWarning: true,
		},
		{
			name:        "config environment shadows the environment variable for the other scope",
			cfgEnv:      "e-cfg",
			envTenant:   "t-env",
			wantKind:    providerdata.ScopeEnvironment,
			wantID:      "e-cfg",
			wantWarning: true,
		},
		{
			name:      "config tenant with the matching environment variable set warns about nothing",
			cfgTenant: "t-cfg",
			envTenant: "t-env",
			wantKind:  providerdata.ScopeTenant,
			wantID:    "t-cfg",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(envTenantID, tc.envTenant)
			t.Setenv(envEnvironmentID, tc.envEnv)

			kind, id, diags := resolveScope(strOrNull(tc.cfgEnv), strOrNull(tc.cfgTenant))

			if kind != tc.wantKind {
				t.Errorf("scope kind: got %v, want %v", kind, tc.wantKind)
			}
			if id != tc.wantID {
				t.Errorf("scope id: got %q, want %q", id, tc.wantID)
			}
			if got := diags.HasError(); got != tc.wantError {
				t.Errorf("HasError: got %v, want %v (%v)", got, tc.wantError, diags)
			}
			if got := countWarnings(diags) > 0; got != tc.wantWarning {
				t.Errorf("warning present: got %v, want %v (%v)", got, tc.wantWarning, diags)
			}
		})
	}
}

// TestResolveScope_ConflictErrorNamesBothAttributes guards the message a user
// actually reads when they set both: it has to name what to remove, and say why
// the pair is not a fallback chain.
func TestResolveScope_ConflictErrorNamesBothAttributes(t *testing.T) {
	t.Setenv(envTenantID, "")
	t.Setenv(envEnvironmentID, "")

	_, _, diags := resolveScope(types.StringValue("e"), types.StringValue("t"))
	if !diags.HasError() {
		t.Fatal("expected an error when both scopes are configured")
	}
	detail := diags.Errors()[0].Detail()
	for _, want := range []string{"tenant_id", "environment_id", "OWNERSHIP_FORBIDDEN"} {
		if !strings.Contains(detail, want) {
			t.Errorf("conflict detail does not mention %q: %s", want, detail)
		}
	}
}

// strOrNull models an unset provider attribute as null rather than the empty
// string, which is what the framework actually hands Configure.
func strOrNull(v string) types.String {
	if v == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}

// countWarnings returns the number of warning diagnostics in d.
func countWarnings(d diag.Diagnostics) int {
	n := 0
	for _, e := range d {
		if e.Severity() == diag.SeverityWarning {
			n++
		}
	}
	return n
}
