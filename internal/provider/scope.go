// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// resolveScope decides which request context the SDK client is built with, from
// the provider block and the environment.
//
// `environment_id` and `tenant_id` are mutually exclusive and both optional, and
// mirror the three choices Jamf offers when an API integration is created:
// *Platform environment* (preferred), *Tenant* (legacy), and *Organization
// management* (neither attribute set — no scope header, the gateway resolves the
// context from the access token). An integration targets exactly one, and the
// SDK sends exactly one scope header, so there is no combination to support:
// setting both is a configuration mistake, not a fallback chain. Which scopes
// reach a given construct differs per API family — Jamf Account is reachable
// only organization-scoped, Jamf Pro under either environment or tenant — which
// is why the requirement is enforced per construct (see
// providerdata.RequireScope) instead of by making one of these attributes
// required here.
//
// Precedence, and why:
//
//   - Both set in the provider block → error. Nothing can disambiguate it.
//   - One set in the provider block → that wins, and a set environment variable
//     for the *other* scope raises a warning. Explicit configuration beating the
//     environment is the Terraform convention, but silently ignoring an exported
//     JAMFPLATFORM_TENANT_ID while honouring an HCL `environment_id` is the kind
//     of thing an operator debugs for an hour, so it is said out loud.
//   - Neither set in the provider block → both environment variables are read,
//     and both being set is an error for the same reason as the first case.
//   - Nothing set anywhere → organization scope, no header. Not an error here:
//     the per-construct gate reports it against the resource that actually needs
//     a scope, which is a far more useful place to read it.
func resolveScope(environment, tenant types.String) (providerdata.ScopeKind, string, diag.Diagnostics) {
	var diags diag.Diagnostics

	cfgEnvironment := environment.ValueString()
	cfgTenant := tenant.ValueString()

	if cfgEnvironment != "" && cfgTenant != "" {
		diags.AddError(
			"Conflicting API Integration Scope",
			"Only one of `environment_id` or `tenant_id` may be set. An API integration targets a platform "+
				"environment or a single tenant, and the provider sends one scope accordingly, so the two are "+
				"alternatives rather than a pair — supplying the one the integration was not created for is "+
				"refused with 403 OWNERSHIP_FORBIDDEN even when both IDs belong to the same customer. "+
				"Remove whichever does not match the integration in use; `environment_id` is the scope to "+
				"prefer for new configurations.",
		)
		return providerdata.ScopeOrganization, "", diags
	}

	if cfgEnvironment != "" {
		if getenv(envTenantID) != "" {
			diags.Append(ignoredScopeEnvVarWarning(envTenantID, "environment_id"))
		}
		return providerdata.ScopeEnvironment, cfgEnvironment, diags
	}

	if cfgTenant != "" {
		if getenv(envEnvironmentID) != "" {
			diags.Append(ignoredScopeEnvVarWarning(envEnvironmentID, "tenant_id"))
		}
		return providerdata.ScopeTenant, cfgTenant, diags
	}

	envEnvironment := getenv(envEnvironmentID)
	envTenant := getenv(envTenantID)

	switch {
	case envEnvironment != "" && envTenant != "":
		diags.AddError(
			"Conflicting API Integration Scope",
			fmt.Sprintf("Both %s and %s are set, but a client can carry only one scope. "+
				"Unset whichever does not match the API integration in use, or set `environment_id` or "+
				"`tenant_id` explicitly in the provider block to override the environment.",
				envEnvironmentID, envTenantID),
		)
		return providerdata.ScopeOrganization, "", diags
	case envEnvironment != "":
		return providerdata.ScopeEnvironment, envEnvironment, diags
	case envTenant != "":
		return providerdata.ScopeTenant, envTenant, diags
	}

	return providerdata.ScopeOrganization, "", diags
}

// ignoredScopeEnvVarWarning reports an environment variable that the explicit
// provider-block attribute has shadowed.
func ignoredScopeEnvVarWarning(envVar, attr string) diag.Diagnostic {
	return diag.NewWarningDiagnostic(
		"Ignored Scope Environment Variable",
		fmt.Sprintf("%s is set but was ignored because `%s` is configured in the provider block, "+
			"and the two scopes are mutually exclusive. Unset %s to remove this warning.",
			envVar, attr, envVar),
	)
}
