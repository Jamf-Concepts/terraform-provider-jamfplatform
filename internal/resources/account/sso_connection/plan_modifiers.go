// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_connection

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// ModifyPlan replaces the connection whenever any configured value changes,
// because Jamf Account has no working endpoint for changing one.
//
// PUT /sso/v1/connections/{id} answers 500 UPSTREAM_ERROR for every request —
// with the exact body a create accepts, with the stored name, with a fresh name,
// and at an identifier that does not exist. Creating, reading, listing and
// destroying a connection all work, so the fault is specific to applying a
// change rather than to writing at all. Wire-verified on 2026-09-03 that the
// refused write applies nothing: a connection read back after a PUT changing
// three fields was byte-identical.
//
// So an in-place change cannot succeed, and the only honest plan is a replacement.
// Without this, every edit would plan as an update and fail during apply.
//
// **This whole file is temporary.** When Jamf fixes the endpoint, delete it and
// the `var _ resource.ResourceWithModifyPlan` assertion, and the resource updates
// in place again — the Update method is already written and its acceptance test
// already exists, gated behind skipUnlessConnectionUpdatesWork. That test's own
// doc comment lists the two expectations to restore alongside this deletion.
//
// It lives in one place rather than as a RequiresReplace plan modifier on each of
// the fifty-odd configurable attributes, so that reverting it is deleting a file
// rather than auditing a schema.
//
// Two things it deliberately does not do. It does not force replacement on a
// create or a destroy, where there is nothing to replace. And it does not compare
// Computed attributes, which are Unknown in the plan and would make every plan a
// replacement — only values an operator can actually set are considered.
func (r *ConnectionResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var plan, state ConnectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, changed := range changedConfigurablePaths(plan, state) {
		if !plannedAttributeIsFullyKnown(req.Plan.Raw, changed.String()) {
			continue
		}
		resp.RequiresReplace.Append(changed)
	}
}

// plannedAttributeIsFullyKnown reports whether the plan holds a settled value for
// one top-level attribute.
//
// planValueDiffers can only exempt an unknown value it recognises as one, and it
// recognises a value that implements attr.Value. Six of the compared attributes
// do not: the four settings blocks and the group filter are pointers to structs,
// and the two product collections are plain Go slices, so an unresolved reference
// *inside* any of them decodes into a populated model that simply differs from
// what is in state. That is half the settings surface, and reading it as a change
// would destroy and recreate a live connection on every plan holding a pending
// reference — the outcome the exemption exists to prevent.
//
// The raw plan is what settles it, because it carries unknown-ness at every
// depth rather than only where a Go type can express it. Every compared path is a
// single top-level attribute name, so indexing the plan object is enough and no
// path walk is needed. An object that cannot be read as one, or a name the plan
// does not carry, is treated as known: this guard exists to suppress a
// replacement, and suppressing one that should happen would leave an operator
// with a plan that fails during apply instead.
func plannedAttributeIsFullyKnown(raw tftypes.Value, name string) bool {
	if raw.IsNull() || !raw.IsKnown() {
		return false
	}
	var object map[string]tftypes.Value
	if err := raw.As(&object); err != nil {
		return true
	}
	planned, present := object[name]
	if !present {
		return true
	}
	return planned.IsFullyKnown()
}

// changedConfigurablePaths reports which operator-settable attributes differ
// between the planned and recorded values.
//
// Every attribute an operator can set is listed, because a missed one would plan
// as an in-place update and fail during apply — the failure mode this exists to
// prevent. `client_secret` is absent on purpose: it is WriteOnly, so it is never
// in state and a comparison would always find it changed. Rotation is driven by
// `client_secret_wo_version`, which is in state and is listed.
//
// The nested settings blocks are compared whole rather than field by field. A
// change to anything inside one changes the block, which is the granularity a
// replacement needs anyway.
func changedConfigurablePaths(plan, state ConnectionResourceModel) []path.Path {
	var changed []path.Path
	for _, comparison := range connectionComparisons(plan, state) {
		if planValueDiffers(comparison.planned, comparison.current) {
			changed = append(changed, path.Root(comparison.name))
		}
	}
	return changed
}

// attributeComparison pairs an attribute name with the planned and recorded
// values to compare for it.
type attributeComparison struct {
	name    string
	planned any
	current any
}

// connectionComparisons lists every operator-settable attribute and the values to
// compare for it.
//
// It is a function rather than a literal inside changedConfigurablePaths so that
// plan_modifiers_test.go can read the names back and check them against the
// resource schema. An attribute that can be set but is missing here plans as an
// in-place update and fails during apply, silently and only for that one
// attribute — so the guard matters more than the indirection costs.
func connectionComparisons(plan, state ConnectionResourceModel) []attributeComparison {
	return []attributeComparison{
		{"name", plan.Name, state.Name},
		{"auth_method", plan.AuthMethod, state.AuthMethod},
		{"client_id", plan.ClientID, state.ClientID},
		{"client_secret_wo_version", plan.ClientSecretWOVersion, state.ClientSecretWOVersion},
		{"scopes", plan.Scopes, state.Scopes},
		{"pkce", plan.PKCE, state.PKCE},
		{"send_nonce", plan.SendNonce, state.SendNonce},
		{"sync_attributes_at_login", plan.SyncAttributesAtLogin, state.SyncAttributesAtLogin},
		{"omit_login_hint", plan.OmitLoginHint, state.OmitLoginHint},
		{"custom_username_claim_name", plan.CustomUsernameClaimName, state.CustomUsernameClaimName},
		{"username_domain", plan.UsernameDomain, state.UsernameDomain},
		{"attribute_map", jsonComparable(plan.AttributeMap), jsonComparable(state.AttributeMap)},
		{"group_name_filter", plan.GroupNameFilter, state.GroupNameFilter},
		{"session_duration_minutes", plan.SessionDurationMinutes, state.SessionDurationMinutes},
		{"inactivity_timeout_minutes", plan.InactivityTimeoutMinutes, state.InactivityTimeoutMinutes},
		{"domains", plan.Domains, state.Domains},
		{"enabled_products", plan.EnabledProducts, state.EnabledProducts},
		{"enabled_environments", plan.EnabledEnvironments, state.EnabledEnvironments},
		{"generic_oidc", plan.GenericOIDC, state.GenericOIDC},
		{"entra", plan.Entra, state.Entra},
		{"okta", plan.Okta, state.Okta},
		{"google_workspace", plan.GoogleWorkspace, state.GoogleWorkspace},
	}
}

// planValueDiffers reports whether a planned value differs from the recorded one,
// treating an Unknown planned value as unchanged.
//
// Unknown is what the framework puts in the plan for a value it cannot resolve
// yet — most often one derived from another resource that has not been applied.
// Reading that as a change would replace the connection on every plan where any
// dependency is pending, which is both wrong and destructive, so it is the one
// case that has to be excluded rather than compared.
func planValueDiffers(planned, current any) bool {
	if unknowable, ok := planned.(attr.Value); ok && unknowable.IsUnknown() {
		return false
	}
	return !reflect.DeepEqual(planned, current)
}

// jsonComparable normalises a JSON-object string attribute for comparison, so
// that reindenting or reordering keys is not a change.
//
// attribute_map's schema promised exactly that until this landed — "Formatting
// and key order are not significant: the value is compared as JSON, so
// reindenting it produces no change" — but the comparison behind this file is
// reflect.DeepEqual over the raw strings, which made the promise false in the
// one place it matters most. `-generate-config-out` re-emits the value as
// `jsonencode({ ... })`, whose formatting differs from the string Jamf Account
// returned, and because every changed configurable attribute here forces
// replacement, importing a connection and applying the generated configuration
// DESTROYED AND RECREATED a live SSO connection over whitespace.
//
// An unknown or null value, or one that is not valid JSON, is returned
// unchanged: the caller's own unknown-value exemption still applies, and a
// non-JSON value is the validator's business, not this comparison's.
//
// It settles the replacement decision and nothing else, and the in-place
// formatting diff left outside that decision cannot be closed by semantic
// equality while `PUT /sso/v1/connections/{id}` fails. The framework never
// applies semantic equality while planning — it consults it only where a value
// the provider returned is reconciled against the value it was handed — and the
// convergence it does deliver comes from Update passing the *planned* state as
// the prior side of that comparison, so state adopts the author's formatting on
// the first apply. StringSemanticEquals in
// internal/resources/ai_governance/policy/custom_types.go records that
// mechanism. Here no apply of a change can succeed, so a JSON custom type would
// have nothing to converge on. Once Jamf fixes the update endpoint, one on this
// attribute would converge the way settings_json does — and this file, by its
// own header, is meant to be deleted at the same time.
func jsonComparable(value types.String) any {
	if value.IsUnknown() || value.IsNull() {
		return value
	}
	var parsed any
	if err := json.Unmarshal([]byte(value.ValueString()), &parsed); err != nil {
		return value
	}
	canonical, err := json.Marshal(parsed)
	if err != nil {
		return value
	}
	return types.StringValue(string(canonical))
}
