// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package providerdata

import (
	"fmt"
	"slices"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/account"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/aigovernance"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/blueprints"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/compliancebenchmarks"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/deviceactions"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/devicegroups"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/devices"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/securitycloud"
)

// The scope sets below are what RequireScope call sites pass, and they are
// derived from the SDK privilege registry rather than written out per construct.
//
// SDK v0.22.0 began emitting MethodPrivileges.Scopes — the scope kinds an
// endpoint is published at — so the answer to "which integration scope reaches
// this API family" is now data the SDK carries instead of a literal repeated at
// every Configure. That matters because the answer moves: the Platform API GA
// withdrew X-Tenant-Id from six Platform specs, and a set written out by hand at
// 25 call sites is a set that goes stale silently at 25 call sites. Deriving it
// means a spec ingest that narrows or widens a family arrives as one changed
// value, and scopes_test.go fails on the entries whose justification the change
// retires.
//
// Scopes is an ALTERNATIVES set per method — the endpoint is published at each
// kind listed and the caller picks one, because a client carries exactly one
// scope. A construct calls several methods through one client, so the scope its
// credential must carry is the INTERSECTION across those methods, not the union.
// Scope is declared per spec rather than per operation, so in practice every
// method in a package agrees and the intersection is that package's set;
// TestFamilyScopesAreUniformPerRegistry pins the assumption rather than trusting
// it, since the SDK keeps the field per-method precisely because two specs in
// one package have disagreed before.

// scopeOrder is the order a resolved scope set is reported in, and therefore the
// order RequireScope's diagnostic offers the acceptable scopes: environment
// first, because it is the scope Jamf intends new integrations to be created
// with, then tenant, then organization.
var scopeOrder = []ScopeKind{ScopeEnvironment, ScopeTenant, ScopeOrganization}

// Scope sets per API family, in the preference order RequireScope documents.
//
// AccountScopes resolves to organization alone from the SDK's own
// config.scopeTypes rather than from a published extension: no account spec
// declares x-scope-types, and the routes resolve the organization from the
// access token. It is the one family whose set is a wire-evidenced assertion on
// the SDK's side too — read MethodPrivileges.ScopesSource to tell that apart
// from a transcription.
var (
	AccountScopes              = declaredScopes("Jamf Account", account.Privileges)
	AIGovernanceScopes         = declaredScopes("Jamf AI Governance", aigovernance.Privileges)
	BlueprintsScopes           = declaredScopes("Blueprints", blueprints.Privileges)
	ComplianceBenchmarksScopes = declaredScopes("Compliance Benchmarks", compliancebenchmarks.Privileges)
	DeviceActionsScopes        = declaredScopes("Platform device actions", deviceactions.Privileges)
	DeviceGroupsScopes         = declaredScopes("Platform device groups", devicegroups.Privileges)
	DevicesScopes              = declaredScopes("Platform devices", devices.Privileges)
	ProScopes                  = declaredScopes("Jamf Pro", pro.Privileges, proclassic.Privileges)
	SecurityCloudScopes        = declaredScopes("Jamf Security Cloud", securitycloud.Privileges)
)

// widening records a scope the gateway still serves after the published spec
// withdrew it, so a construct keeps accepting a credential that demonstrably
// works. It is the provider-side counterpart of the SDK's own config.scopeTypes
// override, and carries the same obligation: an entry is an assertion about the
// gateway, evidenced on the wire on a stated date, not a preference.
//
// An entry is deleted, not edited, when either half of its justification goes:
// the spec catching up makes it redundant, and the gateway following the
// withdrawal makes it wrong. TestWideningEntriesAreStillWidenings fails on the
// first case, because a redundant entry is indistinguishable from a live one at
// a call site and would go on claiming evidence for a scope the spec now grants
// outright. The second case cannot be caught from here — it needs a probe — so
// each entry names its evidence and its date in full, and dropping the scope is
// the deliberate act the blueprints and compliance-benchmarks families already
// went through.
type widening struct {
	family string
	scope  ScopeKind
	// why states the wire evidence: what was probed, when, and what answered.
	why string
}

// gatewayWidenings are the families whose spec-declared scope set understates
// what the gateway serves.
//
// All three are the Platform API GA's tenant withdrawal (public-apis-oas#436,
// #437, #439), which stripped X-Tenant-Id from six Platform specs while the
// gateway went on serving it. Two independent probes, both 2026-09-04, each with
// GET /pro/v1/jamf-pro-version at 200 as the served control in the same
// invocation: the SDK's, recorded in jamfplatform-go-sdk v0.22.0 alongside an
// unrouted control, and this repo's own through the SDK against eu with the
// pro-tenant credential — devices and device-groups both 200 under X-Tenant-Id.
// The acceptance suite then ran green under tenant scope for jamfplatform_device_group
// (9 tests) and jamfplatform_devices, which is the part a status code cannot
// show: these constructs work end to end on a credential the spec says should
// not reach them, so narrowing them would break working configurations.
// jamfplatform-go-sdk's TestAcceptance_TenantScopePlatformSpecsStillServed fails
// the day the withdrawal lands.
//
// Blueprints and Compliance Benchmarks are deliberately ABSENT, and the reasoning
// is worth reading before adding them back. Their specs withdrew tenant on the
// same build, and two DIFFERENT tenant-scoped credentials are refused 403
// BAD_PERMISSIONS on both — the SDK's on 2026-09-04 and this repo's pro-tenant
// acceptance credential, first on 2026-09-03 in .github/acceptance-lanes.json's
// evidence field and re-probed 2026-09-04. That still does not establish the
// route is unrouted: this provider's own law is that a Platform 403
// BAD_PERMISSIONS is indistinguishable from a privilege gap, and two credentials
// that both lack the capability tell them apart no better than one does.
//
// The narrowing does not rest on that question, which is why it stands anyway.
// The operative fact is that these capabilities cannot be GRANTED to a
// tenant-scoped integration in Jamf Account at all, so no tenant credential can
// ever reach them whether or not the gateway routes the path — recorded in the
// lane table for the same two families. The spec's withdrawal agrees, and the
// environment-only outcome for exactly these two was this repo's recorded GA
// decision before either. A tenant-scoped operator therefore gets a named
// diagnostic at Configure — verified live, in 0.25s, where the same run
// previously spent 134 seconds applying before a 403 — instead of an opaque
// failure mid-apply.
//
// AI Governance needs no entry either and never had one: it is environment-only
// in the spec, refused 403 under tenant scope on the same probe, and was already
// gated ScopeEnvironment by hand before this file existed.
var gatewayWidenings = []widening{
	{
		family: "Platform devices",
		scope:  ScopeTenant,
		why:    "GET /devices/v1/devices answered 200 with 13 rows under X-Tenant-Id",
	},
	{
		family: "Platform device groups",
		scope:  ScopeTenant,
		why:    "GET /device-groups/v1/groups answered 200 with 53 rows under X-Tenant-Id",
	},
	{
		family: "Platform device actions",
		scope:  ScopeTenant,
		why: "POST /device-management-action/v1 answered 404 NOT_FOUND under X-Tenant-Id — a " +
			"routed handler rejecting the request, not the gateway refusing the header",
	},
}

// resolvedFamilies records every family name declaredScopes has been called
// with. gatewayWidenings is keyed by that name, and a key that matches nothing
// widens nothing — silently, since a widening that does not apply looks exactly
// like a family the gateway never diverged on. TestWideningFamiliesAreResolved
// reads this map so a mistyped or renamed family fails the build instead.
var resolvedFamilies = map[string]bool{}

// declaredScopes resolves the scope kinds a credential may carry to reach every
// method in the given SDK privilege registries, widened by any evidenced
// gatewayWidenings entry for the family.
//
// It panics rather than returning an empty set, because RequireScope treats an
// empty allowed list as "no scope requirement" and would let an unreachable
// construct plan and then fail at the gateway. Both panics are deterministic and
// fire at package initialisation, so `go test ./...` reaches them before any
// build ships: an empty registry means the SDK renamed or dropped the package,
// and an empty intersection means two specs in one package disagree so
// completely that no single credential reaches the family — which is a defect in
// the split, not a scope to enforce.
func declaredScopes(family string, regs ...map[string]jamfplatform.MethodPrivileges) []ScopeKind {
	resolvedFamilies[family] = true
	var accepted []ScopeKind
	seen := false
	for _, reg := range regs {
		for _, mp := range reg {
			kinds := methodScopes(family, mp)
			if !seen {
				accepted, seen = kinds, true
				continue
			}
			accepted = slices.DeleteFunc(accepted, func(k ScopeKind) bool {
				return !slices.Contains(kinds, k)
			})
		}
	}
	if !seen {
		panic(fmt.Sprintf("providerdata: the SDK privilege registry for %s is empty — "+
			"the package was renamed or withdrawn, so its scope cannot be resolved", family))
	}
	for _, w := range gatewayWidenings {
		if w.family == family && !slices.Contains(accepted, w.scope) {
			accepted = append(accepted, w.scope)
		}
	}
	if len(accepted) == 0 {
		panic(fmt.Sprintf("providerdata: no scope reaches every %s method — the SDK registry's "+
			"specs disagree, so no single credential can serve the family", family))
	}
	return slices.DeleteFunc(slices.Clone(scopeOrder), func(k ScopeKind) bool {
		return !slices.Contains(accepted, k)
	})
}

// methodScopes maps one registry entry's SDK scope kinds onto the provider's,
// deduplicated.
func methodScopes(family string, mp jamfplatform.MethodPrivileges) []ScopeKind {
	out := make([]ScopeKind, 0, len(mp.Scopes))
	for _, s := range mp.Scopes {
		k := scopeKindFromSDK(family, mp.Method, s)
		if !slices.Contains(out, k) {
			out = append(out, k)
		}
	}
	return out
}

// scopeKindFromSDK maps an SDK scope kind onto the provider's.
//
// An unrecognised kind panics instead of falling through to a scope name. The
// SDK models organization as its zero value, so a silent default would map a
// fourth kind the SDK grew onto organization — widening every construct that
// reads it to a credential the endpoint does not accept, which is the failure
// this whole file exists to make impossible.
func scopeKindFromSDK(family, method string, s jamfplatform.ScopeKind) ScopeKind {
	switch s {
	case jamfplatform.ScopeOrganization:
		return ScopeOrganization
	case jamfplatform.ScopeEnvironment:
		return ScopeEnvironment
	case jamfplatform.ScopeTenant:
		return ScopeTenant
	}
	panic(fmt.Sprintf("providerdata: %s.%s declares SDK scope kind %d, which this provider has "+
		"no ScopeKind for — add one rather than letting it default", family, method, s))
}
