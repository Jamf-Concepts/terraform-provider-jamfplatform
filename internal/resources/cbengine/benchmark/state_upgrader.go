// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package benchmark

import (
	"context"

	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.ResourceWithUpgradeState = &BenchmarkResource{}

// benchmarkResourceModelV0 is the v0 resource model: the current one plus the
// removed singular target_device_group. State written by v0 carries that
// attribute, so decoding it needs a struct that declares it — the framework
// errors rather than ignoring an attribute the target struct does not define.
type benchmarkResourceModelV0 struct {
	ID                  types.String           `tfsdk:"id"`
	Title               types.String           `tfsdk:"title"`
	Description         types.String           `tfsdk:"description"`
	SourceBaselineID    types.String           `tfsdk:"source_baseline_id"`
	Sources             types.List             `tfsdk:"sources"`
	SelectedOsVersions  types.Set              `tfsdk:"selected_os_versions"`
	AvailableOsVersions types.List             `tfsdk:"available_os_versions"`
	Rules               []RuleModel            `tfsdk:"rules"`
	TargetDeviceGroup   types.String           `tfsdk:"target_device_group"`
	TargetDeviceGroups  types.Set              `tfsdk:"target_device_groups"`
	EnforcementMode     types.String           `tfsdk:"enforcement_mode"`
	TenantID            types.String           `tfsdk:"tenant_id"`
	Deleted             types.Bool             `tfsdk:"deleted"`
	UpdateAvailable     types.Bool             `tfsdk:"update_available"`
	CanSwitchToEnforce  types.Bool             `tfsdk:"can_switch_to_enforce"`
	LastUpdatedAt       types.String           `tfsdk:"last_updated_at"`
	Timeouts            resourceTimeouts.Value `tfsdk:"timeouts"`
}

// benchmarkSchemaV0 returns the v0 schema: the current one with the removed
// singular target_device_group added back, and target_device_groups relaxed to
// Optional as it was then. Deriving it from benchmarkResourceSchema rather than
// transcribing it keeps the prior schema honest as the rest of the resource
// changes — a hand-copied 160-line rules block would drift on the next edit and
// fail to decode real state.
func benchmarkSchemaV0(ctx context.Context) *schema.Schema {
	s := benchmarkResourceSchema(ctx)
	s.Version = 0
	s.Attributes["target_device_group"] = schema.StringAttribute{Optional: true}
	s.Attributes["target_device_groups"] = schema.SetAttribute{Optional: true, ElementType: types.StringType}
	return &s
}

// UpgradeState migrates v0 state, which could target device groups through either
// the singular target_device_group or the plural target_device_groups, onto v1,
// where only the plural survives.
//
// The singular value is folded into the plural set rather than dropped: a user who
// targeted through the deprecated attribute keeps the same device group after the
// upgrade, so the removal costs them a config edit and never a re-scope. The
// upgrader fixes state only — their .tf still has to change, which is what the
// deprecation window was for.
func (r *BenchmarkResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: benchmarkSchemaV0(ctx),
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var old benchmarkResourceModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &old)...)
				if resp.Diagnostics.HasError() {
					return
				}

				upgraded := BenchmarkResourceModel{
					ID:                  old.ID,
					Title:               old.Title,
					Description:         old.Description,
					SourceBaselineID:    old.SourceBaselineID,
					Sources:             old.Sources,
					SelectedOsVersions:  old.SelectedOsVersions,
					AvailableOsVersions: old.AvailableOsVersions,
					Rules:               old.Rules,
					TargetDeviceGroups:  upgradeTargetDeviceGroups(old),
					EnforcementMode:     old.EnforcementMode,
					TenantID:            old.TenantID,
					Deleted:             old.Deleted,
					UpdateAvailable:     old.UpdateAvailable,
					CanSwitchToEnforce:  old.CanSwitchToEnforce,
					LastUpdatedAt:       old.LastUpdatedAt,
					Timeouts:            old.Timeouts,
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
			},
		},
	}
}

// upgradeTargetDeviceGroups resolves v0's two targeting attributes into v1's one.
// The plural wins when it is set, because v0's schema validation made the two
// mutually exclusive; the singular is promoted to a one-element set when it is the
// one that was used.
func upgradeTargetDeviceGroups(old benchmarkResourceModelV0) types.Set {
	if !old.TargetDeviceGroups.IsNull() && !old.TargetDeviceGroups.IsUnknown() {
		return old.TargetDeviceGroups
	}
	if !old.TargetDeviceGroup.IsNull() && !old.TargetDeviceGroup.IsUnknown() && old.TargetDeviceGroup.ValueString() != "" {
		return types.SetValueMust(types.StringType, []attr.Value{types.StringValue(old.TargetDeviceGroup.ValueString())})
	}
	return types.SetNull(types.StringType)
}
