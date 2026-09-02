// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package providerdata

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// ScopeKind is the request context a Jamf API integration is bound to.
//
// The three kinds are the three choices Jamf offers when an integration is
// created — *Organization management*, *Platform environment*, and *Tenant*. The
// gateway resolves which one applies from a request header (`X-Environment-Id`
// or `X-Tenant-Id`, stamped on every call by the SDK) or, when neither header is
// sent, from the access token alone. It is a property of the configured client,
// not of an individual call: the SDK carries exactly one scope and every request
// inherits it. That is why the check lives on Data and runs in Configure rather
// than per-request.
//
// Scoping used to be a URL path segment (`/api/{namespace}/{version}/tenant/{id}`).
// The Platform API GA moved it to headers, and SDK v0.17.0 followed — its
// `APIPrefix` no longer embeds an ID, and `WithEnvironmentID` / `WithTenantID`
// set a header instead. Nothing in this provider builds URLs, so that transition
// cost no call-site changes; what it added is a second scope to choose between.
//
// The value on a Data is read back off the SDK client via `Client.Scope()`
// (added in v0.18.0) rather than tracked separately, so it always reflects the
// header the client will actually send. See scopeFromClient for why this enum is
// not simply an alias of the SDK's.
type ScopeKind int

const (
	// ScopeOrganization is the absence of an environment or tenant header: the
	// gateway resolves the context from the access token alone. It corresponds
	// to Jamf's *Organization management* integration scope, which covers
	// organization-level resources such as single sign-on and AI Governance.
	//
	// It is deliberately the zero value, because a provider block setting
	// neither `environment_id` nor `tenant_id` is exactly this case.
	//
	// It is the only scope that reaches the jamfplatform_account_* family, and
	// RequireScope rejects it everywhere else. Both directions matter: an
	// organization-scoped integration pointed at a Jamf Pro resource, and an
	// environment-scoped one pointed at a Jamf Account resource, each otherwise
	// produce an opaque gateway error deep in an apply instead of a named
	// diagnostic at Configure.
	ScopeOrganization ScopeKind = iota
	// ScopeEnvironment scopes every request to a platform environment — a group
	// of tenants across product types with interconnected capabilities — sent as
	// `X-Environment-Id`. **This is the preferred scope**, and the one Jamf
	// intends new integrations to be created with. Blueprints and Compliance
	// Benchmarks become exclusive to it at the Platform API GA.
	ScopeEnvironment
	// ScopeTenant scopes every request to a single Jamf Pro, Jamf School, Jamf
	// Protect or Jamf Security Cloud tenant, sent as `X-Tenant-Id`. Jamf
	// describes this as the legacy method for targeting integrations without a
	// platform environment: it remains supported, every published spec still
	// declares this header, and some surfaces are only reachable this way.
	ScopeTenant
)

// String names the scope the way a practitioner would recognise it.
func (k ScopeKind) String() string {
	switch k {
	case ScopeEnvironment:
		return "environment"
	case ScopeTenant:
		return "tenant"
	default:
		return "organization"
	}
}

// Scope reports the integration scope the provider was configured with.
// Nil-receiver-safe so a construct whose Configure never ran reports the zero
// scope rather than panicking.
func (d *Data) Scope() ScopeKind {
	if d == nil {
		return ScopeOrganization
	}
	return d.scope
}

// RequireScope gates a construct on the provider's integration scope, returning
// an error diagnostic when the configured scope is not in allowed.
//
// Scope is enforced per construct rather than once in provider Configure because
// the answer differs per API family and is about to differ more: Jamf Pro is
// reachable under either an environment- or a tenant-scoped integration, while
// Blueprints and Compliance Benchmarks go environment-only at the Platform API
// GA. A single provider-level assertion could not express that, and hard-failing
// Configure on an organization-scoped integration would block the
// organization-level constructs this provider will grow later. Narrowing a
// family at GA is then a one-token edit at its call sites.
//
// Pass allowed in preference order — it is the order the diagnostic lists them
// in, so the scope a user should reach for first comes first.
//
// resourceType is the fully-qualified Terraform type name used in the
// diagnostic (e.g. "jamfplatform_pro_category"). An empty allowed list and a nil
// receiver both pass: the framework calls Configure with nil ProviderData during
// early lifecycle, and callers already guard that before reaching here.
func (d *Data) RequireScope(resourceType string, allowed ...ScopeKind) diag.Diagnostics {
	var diags diag.Diagnostics
	if d == nil || len(allowed) == 0 {
		return diags
	}
	if slices.Contains(allowed, d.scope) {
		return diags
	}
	diags.AddError(
		fmt.Sprintf("Unsupported API Integration Scope for %s", resourceType),
		fmt.Sprintf("%s requires %s, but this provider is configured with %s.\n\n%s",
			resourceType, scopeRequirement(allowed), scopeDescription(d.scope), scopeRemedy(allowed)),
	)
	return diags
}

// scopeRequirement renders allowed as a phrase: "an environment-scoped
// integration", "a tenant-scoped integration", "an environment-scoped or
// tenant-scoped integration".
func scopeRequirement(allowed []ScopeKind) string {
	parts := make([]string, 0, len(allowed))
	for _, k := range allowed {
		parts = append(parts, k.String()+"-scoped")
	}
	joined := joinOr(parts)
	return article(joined) + " " + joined + " integration"
}

// article picks "a" or "an" for a phrase, so the assembled requirement reads as
// English whichever scope happens to come first.
func article(phrase string) string {
	if phrase == "" {
		return "a"
	}
	if strings.ContainsRune("aeiouAEIOU", rune(phrase[0])) {
		return "an"
	}
	return "a"
}

// scopeDescription names the configured scope and both inputs that could have
// selected it, so the diagnostic says what to look at rather than only what is
// wrong.
//
// It names the environment variable alongside the provider-block attribute
// because either source selects a scope and the diagnostic cannot tell which
// one did: the provider resolves the scope from `JAMFPLATFORM_ENVIRONMENT_ID` /
// `JAMFPLATFORM_TENANT_ID` whenever the provider block sets neither attribute,
// and says nothing when it does. Asserting the attribute "is set" would point
// the most likely reader — a CI runner exporting the preferred
// `JAMFPLATFORM_ENVIRONMENT_ID` — at a line that is not in their configuration.
func scopeDescription(k ScopeKind) string {
	switch k {
	case ScopeEnvironment:
		return "an environment-scoped integration (selected by `environment_id` or `JAMFPLATFORM_ENVIRONMENT_ID`)"
	case ScopeTenant:
		return "a tenant-scoped integration (selected by `tenant_id` or `JAMFPLATFORM_TENANT_ID`)"
	default:
		return "an organization-scoped integration (no scope selected by `environment_id`, `tenant_id`, " +
			"or either `JAMFPLATFORM_*` scope variable)"
	}
}

// scopeRemedy says which attribute to set, and warns that the choice is not free
// — the header has to match the scope the API integration was created against.
//
// The organization-only branch names the two environment variables as well as
// the two attributes, for the reason given on scopeDescription: a scope set in
// the environment is invisible in the provider block, so telling the operator to
// remove an attribute they never wrote leaves the diagnostic unactionable.
func scopeRemedy(allowed []ScopeKind) string {
	attrs := make([]string, 0, len(allowed))
	for _, k := range allowed {
		switch k {
		case ScopeEnvironment:
			attrs = append(attrs, "`environment_id`")
		case ScopeTenant:
			attrs = append(attrs, "`tenant_id`")
		}
	}
	if len(attrs) == 0 {
		return "Unset both scope inputs so requests are scoped from the access token alone: remove `environment_id` " +
			"and `tenant_id` from the provider block, and unset `JAMFPLATFORM_ENVIRONMENT_ID` and " +
			"`JAMFPLATFORM_TENANT_ID` in the environment — either source selects a scope."
	}
	return fmt.Sprintf(
		"Set %s in the provider block, or the matching `JAMFPLATFORM_*` environment variable. "+
			"The scope must match the one the API integration was created with: an integration targets a platform environment or a single "+
			"tenant, and crossing over is refused with 403 OWNERSHIP_FORBIDDEN even when both IDs belong to the same customer — so this is a "+
			"choice between two integrations, not two IDs for one.",
		joinOr(attrs),
	)
}

// joinOr renders a list as "a", "a or b", or "a, b or c".
func joinOr(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + " or " + parts[len(parts)-1]
	}
}
