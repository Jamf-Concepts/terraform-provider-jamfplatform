// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_app

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/securitycloud"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// wildcardAll is the host name that matches every destination — the full-tunnel
// case. A host name pattern rather than an API enum value, which is why it is a
// literal here.
const wildcardAll = "*"

// normalisedHostnameValidator refuses a host name Jamf Security Cloud would store
// in a different form from the one written.
type normalisedHostnameValidator struct{}

// normalisedHostname returns a validator.String refusing host names the server
// would rewrite.
//
// The endpoint accepts `Foo.Example.COM.` and stores `foo.example.com`: upper case
// is folded and a trailing root dot is dropped (wire-verified 2026-08-30). Because
// `hostnames` is Optional rather than Optional+Computed — removing it has to clear
// the list, not leave the server's value in place — Terraform will not let the
// provider rewrite the planned value to match, so silently accepting either form
// would leave the configuration disagreeing with its own refresh on every plan,
// forever. Refusing them instead costs one clear error at plan time and says which
// form to write.
//
// This is deliberately not part of commonvalidators.DNSHostnameOrWildcard, which
// describes what the endpoint accepts. The extra restriction here is the provider's,
// not Jamf's.
func normalisedHostname() validator.String {
	return normalisedHostnameValidator{}
}

// Description returns a plain-text description of the validator.
func (normalisedHostnameValidator) Description(_ context.Context) string {
	return "must be lower-case and must not end in a dot"
}

// MarkdownDescription returns the markdown description of the validator.
func (v normalisedHostnameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString implements validator.String.
func (v normalisedHostnameValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if strings.HasSuffix(value, ".") {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Host name ends in a dot",
			"Jamf Security Cloud drops a trailing dot, so it would store this host name in a different form "+
				"from the one written here and every subsequent plan would show a change. Remove the trailing "+
				"dot. Got: "+value,
		)
		return
	}
	if lower := strings.ToLower(value); lower != value {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Host name is not lower-case",
			"Jamf Security Cloud folds host names to lower case, so it would store this one in a different form "+
				"from the one written here and every subsequent plan would show a change. Write it as \""+
				lower+"\". Got: "+value,
		)
	}
}

// hostnameOverlapValidator enforces that no host name on an application is covered
// by another entry's wildcard.
type hostnameOverlapValidator struct{}

var _ resource.ConfigValidator = hostnameOverlapValidator{}

// Description returns a plain-text description of the validator.
func (hostnameOverlapValidator) Description(_ context.Context) string {
	return "hostnames must be mutually exclusive: no entry may be covered by another entry's wildcard"
}

// MarkdownDescription returns the markdown description of the validator.
func (v hostnameOverlapValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateResource implements resource.ConfigValidator.
//
// The server refuses an overlapping set with `400 [INVALID_FIELD] hostnames:
// Hostnames have to be mutually exclusive.` — which names the attribute but not
// which pair collided, and only after the write has been attempted.
func (v hostnameOverlapValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config ZtnaAppResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateHostnameOverlap(ctx, config, &resp.Diagnostics)
}

// validateHostnameOverlap applies the mutual-exclusivity rule. Split from
// ValidateResource so it can be unit-tested against a model rather than a
// framework config, which is the only way the pairing logic is reachable.
func validateHostnameOverlap(_ context.Context, config ZtnaAppResourceModel, diags *diag.Diagnostics) {
	hostnames, _ := knownStrings(config.Hostnames)
	sort.Strings(hostnames)

	for _, candidate := range hostnames {
		for _, other := range hostnames {
			if other == candidate || !coversHostname(other, candidate) {
				continue
			}
			diags.AddAttributeError(
				path.Root("hostnames"),
				"Host names overlap",
				fmt.Sprintf(
					"Jamf Security Cloud requires an application's host names to be mutually exclusive, and "+
						"%q is already covered by %q. Remove the more specific entry, or narrow the wildcard.",
					candidate, other,
				),
			)
			return
		}
	}
}

// coversHostname reports whether pattern matches candidate by wildcard. `*` covers
// everything; `*.example.com` covers any name ending in `.example.com` but not
// `example.com` itself — the parent has to be listed separately, the same rule the
// custom DNS zone domains follow.
func coversHostname(pattern, candidate string) bool {
	if pattern == wildcardAll {
		return true
	}
	suffix, ok := strings.CutPrefix(pattern, wildcardAll)
	if !ok {
		return false
	}
	return strings.HasSuffix(candidate, suffix)
}

// appFormValidator enforces that an application is either predefined or custom, and
// that each form carries only the fields belonging to it.
type appFormValidator struct{}

var _ resource.ConfigValidator = appFormValidator{}

// Description returns a plain-text description of the validator.
func (appFormValidator) Description(_ context.Context) string {
	return "exactly one of name or predefined_app_id must be set"
}

// MarkdownDescription returns the markdown description of the validator.
func (v appFormValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateResource implements resource.ConfigValidator.
//
// The absent-name case the server does catch, with `400 [INVALID_FIELD] name: App
// name is required for Enterprise applications.` The combination case it does not:
// a name sent alongside a predefined app ID is accepted with `201` and then
// silently discarded, so the created application reads back with a null name
// (wire-verified 2026-08-30). Left to the server, the name an operator wrote would
// vanish without a word and the next plan would try to set it again.
func (v appFormValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config ZtnaAppResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateAppForm(config, &resp.Diagnostics)
}

// validateAppForm applies the form rules. Split from ValidateResource so it can be
// unit-tested against a model rather than a framework config.
func validateAppForm(config ZtnaAppResourceModel, diags *diag.Diagnostics) {
	hasName := !config.Name.IsNull()
	hasPredefined := !config.PredefinedAppID.IsNull()

	switch {
	case hasName && hasPredefined:
		diags.AddAttributeError(
			path.Root("name"),
			"A predefined application cannot be renamed",
			"`predefined_app_id` is set, which makes this a predefined application whose name belongs to the "+
				"Jamf-maintained definition. Jamf Security Cloud accepts a `name` here and then discards it, so "+
				"remove `name` — or remove `predefined_app_id` to define a custom application instead.",
		)
	case !hasName && !hasPredefined:
		diags.AddAttributeError(
			path.Root("name"),
			"An application needs a name",
			"A custom application must have a `name`. Set one, or set `predefined_app_id` to base this "+
				"application on one of the definitions the "+
				"`jamfplatform_security_cloud_ztna_predefined_apps` data source lists.",
		)
	}
}

// routingCombinationValidator enforces the cross-field rules on every routing block
// — the application's own and each per-group override's.
type routingCombinationValidator struct{}

var _ resource.ConfigValidator = routingCombinationValidator{}

// Description returns a plain-text description of the validator.
func (routingCombinationValidator) Description(_ context.Context) string {
	return "a routing block must set gateway_id and routing_mode when routing via ZTNA, and neither when routing directly"
}

// MarkdownDescription returns the markdown description of the validator.
func (v routingCombinationValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateResource implements resource.ConfigValidator.
//
// All four mistakes — a missing gateway, a missing routing mode, a gateway supplied
// for direct routing and a routing mode supplied for direct routing — draw the same
// `400 [INVALID_FIELD] routing: Routing definition is not valid.`, which says
// nothing about which one it was. So does an unrecognised gateway ID, which is the
// one case this cannot pre-empt.
func (v routingCombinationValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config ZtnaAppResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateAllRouting(ctx, config, &resp.Diagnostics)
}

// validateAllRouting applies the routing rules to the application's own routing and
// to every override's. Split from ValidateResource so it can be unit-tested against
// a model rather than a framework config.
func validateAllRouting(ctx context.Context, config ZtnaAppResourceModel, diags *diag.Diagnostics) {
	validateRouting(config.Routing, path.Root("routing"), diags)

	overrides, overrideDiags := routingOverrideModels(ctx, config)
	if overrideDiags.HasError() {
		return
	}
	for i, override := range overrides {
		validateRouting(override.Routing, path.Root("routing_overrides").AtListIndex(i).AtName("routing"), diags)
	}
}

// validateRouting applies the routing cross-field rules at one path.
func validateRouting(routing *RoutingModel, at path.Path, diags *diag.Diagnostics) {
	if routing == nil || routing.Mode.IsNull() || routing.Mode.IsUnknown() {
		return
	}

	hasGateway := !routing.GatewayID.IsNull()
	hasRoutingMode := !routing.RoutingMode.IsNull()
	viaZTNA := routing.Mode.ValueString() == routingModeLabels[securitycloud.RoutingTypeCustom]

	if viaZTNA {
		if !hasGateway {
			diags.AddAttributeError(
				at.AtName("gateway_id"),
				"Routing via ZTNA needs an access gateway",
				"`mode` is \""+routingModeLabels[securitycloud.RoutingTypeCustom]+"\", which sends this application's traffic through "+
					"a gateway, so `gateway_id` has to name one. Use the "+
					"`jamfplatform_security_cloud_ztna_shared_gateways` data source for a Jamf-managed gateway, "+
					"or reference one of your own.",
			)
		}
		if !hasRoutingMode {
			diags.AddAttributeError(
				at.AtName("routing_mode"),
				"Routing via ZTNA needs a routing mode",
				"`mode` is \""+routingModeLabels[securitycloud.RoutingTypeCustom]+"\", and Jamf Security Cloud requires a "+
					"`routing_mode` alongside it. \"Standard\" is the recommended setting; choose \"Legacy\" "+
					"only for devices or applications known to be incompatible with IPv6.",
			)
		}
		return
	}

	if hasGateway {
		diags.AddAttributeError(
			at.AtName("gateway_id"),
			"Direct routing does not use an access gateway",
			"`mode` is \""+routingModeLabels[securitycloud.RoutingTypeDirect]+"\", which leaves traffic to the device's own routing, "+
				"so there is no gateway to send it through. Remove `gateway_id`, or set `mode` to \""+
				routingModeLabels[securitycloud.RoutingTypeCustom]+"\".",
		)
	}
	if hasRoutingMode {
		diags.AddAttributeError(
			at.AtName("routing_mode"),
			"Direct routing has no routing mode",
			"`routing_mode` selects how ZTNA resolves addresses, so it only applies when `mode` is \""+
				routingModeLabels[securitycloud.RoutingTypeCustom]+"\". Remove `routing_mode`, or change `mode`.",
		)
	}
}

// deviceGroupAssignmentValidator enforces the rules tying the device group
// assignment to the per-group routing overrides.
type deviceGroupAssignmentValidator struct{}

var _ resource.ConfigValidator = deviceGroupAssignmentValidator{}

// Description returns a plain-text description of the validator.
func (deviceGroupAssignmentValidator) Description(_ context.Context) string {
	return "device_group_ids applies only when all_device_groups is false, and each override may only name assigned groups, once"
}

// MarkdownDescription returns the markdown description of the validator.
func (v deviceGroupAssignmentValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateResource implements resource.ConfigValidator.
//
// Three rules, all wire-verified 2026-08-30:
//
//   - `device_group_ids` alongside `all_device_groups = true` is accepted and
//     ignored — the server stores an empty group list — so it has to be refused
//     here or it would vanish from state on the first refresh.
//   - An override naming a group the application is not assigned to is refused with
//     a `400` carrying an empty `field`, so nothing points at the override.
//   - A group named by two overrides is refused with a `400` blaming
//     `groupOverrides` as a whole.
//
// The subset rule applies only when `all_device_groups` is false: with it true the
// server accepts an override on any group, because every group is assigned.
func (v deviceGroupAssignmentValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config ZtnaAppResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateDeviceGroupAssignment(ctx, config, &resp.Diagnostics)
}

// validateDeviceGroupAssignment applies the assignment and override rules. Split
// from ValidateResource so it can be unit-tested against a model rather than a
// framework config.
func validateDeviceGroupAssignment(ctx context.Context, config ZtnaAppResourceModel, diags *diag.Diagnostics) {
	allGroups := config.AllDeviceGroups.ValueBool()
	if allGroups && !config.DeviceGroupIDs.IsNull() && !config.DeviceGroupIDs.IsUnknown() {
		diags.AddAttributeError(
			path.Root("device_group_ids"),
			"Selected device groups conflict with all device groups",
			"`all_device_groups` is true, so every device in the fleet can reach this application and Jamf "+
				"Security Cloud ignores any group list sent with it. Set `all_device_groups` to false to "+
				"restrict access to `device_group_ids`, or remove `device_group_ids`.",
		)
		return
	}

	assigned, assignedComplete := knownStrings(config.DeviceGroupIDs)
	assignedSet := make(map[string]struct{}, len(assigned))
	for _, id := range assigned {
		assignedSet[id] = struct{}{}
	}

	overrides, overrideDiags := routingOverrideModels(ctx, config)
	if overrideDiags.HasError() {
		return
	}

	seen := map[string]int{}
	for i, override := range overrides {
		groups, _ := knownStrings(override.DeviceGroupIDs)
		at := path.Root("routing_overrides").AtListIndex(i).AtName("device_group_ids")
		for _, id := range groups {
			if first, ok := seen[id]; ok {
				diags.AddAttributeError(
					at,
					"Device group has more than one routing override",
					fmt.Sprintf(
						"Device group %q already has a routing override at index %d, and Jamf Security Cloud "+
							"allows only one per group. Merge the two overrides, or drop one.",
						id, first,
					),
				)
				return
			}
			seen[id] = i

			if allGroups || !assignedComplete {
				continue
			}
			if _, ok := assignedSet[id]; !ok {
				diags.AddAttributeError(
					at,
					"Routing override names an unassigned device group",
					fmt.Sprintf(
						"Device group %q has a routing override but is not in `device_group_ids`, so it cannot "+
							"reach this application at all. Add it to `device_group_ids`, set "+
							"`all_device_groups` to true, or drop the override.",
						id,
					),
				)
				return
			}
		}
	}
}

// knownStrings returns the values of a config set that Terraform has already
// resolved, and whether the whole collection was resolved.
//
// A config-time validator cannot use ElementsAs here: it refuses a collection holding
// an unknown element, and an unknown element is the commonest case there is — a group
// ID referencing a device group the same plan is about to create. Reading the
// elements directly lets a check run on what is visible and defer on what is not, per
// STYLE_GUIDE §Config-time validators.
//
// The completeness flag matters for one rule only. Two known values that collide are
// a real error whatever else is unknown, so the overlap and duplicate checks need no
// gate. Proving a group is *absent* from a collection needs the whole collection, so
// the subset rule skips unless it has it — otherwise a group whose assignment is
// still unknown would be reported as unassigned.
func knownStrings(set types.Set) ([]string, bool) {
	if set.IsNull() {
		return nil, true
	}
	if set.IsUnknown() {
		return nil, false
	}
	elements := set.Elements()
	out := make([]string, 0, len(elements))
	complete := true
	for _, element := range elements {
		str, ok := element.(types.String)
		if !ok || str.IsUnknown() || str.IsNull() {
			complete = false
			continue
		}
		out = append(out, str.ValueString())
	}
	return out, complete
}

// routingOverrideModels decodes the routing_overrides list from a config model,
// returning an empty slice when it is absent or not yet resolved.
func routingOverrideModels(ctx context.Context, config ZtnaAppResourceModel) ([]RoutingOverrideModel, diag.Diagnostics) {
	if config.RoutingOverrides.IsNull() || config.RoutingOverrides.IsUnknown() {
		return nil, nil
	}
	var models []RoutingOverrideModel
	diags := config.RoutingOverrides.ElementsAs(ctx, &models, false)
	return models, diags
}
