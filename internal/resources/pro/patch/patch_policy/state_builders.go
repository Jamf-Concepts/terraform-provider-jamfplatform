// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_policy

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

// assignPatchPolicyResourceModel populates a resource model from the SDK
// PatchPolicy response. The general fields are always refreshed; the optional
// scope and user_interaction blocks are only refreshed when the caller (plan or
// current state) already manages them. The classic server echoes both <scope>
// and <user_interaction> on GET with default values, so populating an unmanaged
// block would violate the framework's "produced inconsistent result after apply"
// check (plan said null, we'd return a populated object). See
// feedback_server_derived_echo_attrs.
func assignPatchPolicyResourceModel(ctx context.Context, state *PatchPolicyResourceModel, p *proclassic.PatchPolicy) diag.Diagnostics {
	var diags diag.Diagnostics
	if p == nil {
		return diags
	}

	if id := extractPatchPolicyID(p); id != "" {
		state.ID = types.StringValue(id)
	}
	if p.SoftwareTitleConfigurationID != nil {
		state.SoftwareTitleConfigurationID = helpers.PreferCurrentStringPointer(helpers.StringFromIntPtr(p.SoftwareTitleConfigurationID), state.SoftwareTitleConfigurationID)
	}

	diags.Append(flattenGeneral(ctx, p.General, state)...)

	if state.Scope != nil && p.Scope != nil {
		flattenScope(ctx, p.Scope, state.Scope)
	}

	if state.UserInteraction != nil && p.UserInteraction != nil {
		flattenUserInteraction(p.UserInteraction, state.UserInteraction)
	}

	return diags
}

// flattenGeneral maps the wire general block onto the model. Writable fields use
// the preferCurrent helpers (managed-section echo protection); the server-derived
// fields (release_date, incremental_update, reboot, minimum_os, kill_apps) are
// adopted verbatim — they are Computed-only and reflect the patch definition.
func flattenGeneral(ctx context.Context, g *proclassic.PatchPolicyGeneral, state *PatchPolicyResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if g == nil {
		return diags
	}

	state.Name = helpers.PreferCurrentStringPointer(g.Name, state.Name)
	state.TargetVersion = helpers.PreferCurrentStringPointer(g.TargetVersion, state.TargetVersion)
	state.Enabled = helpers.PreferCurrentBoolPointer(g.Enabled, state.Enabled)
	state.DistributionMethod = helpers.PreferCurrentStringPointer(g.DistributionMethod, state.DistributionMethod)
	state.AllowDowngrade = helpers.PreferCurrentBoolPointer(g.AllowDowngrade, state.AllowDowngrade)
	state.PatchUnknown = helpers.PreferCurrentBoolPointer(g.PatchUnknown, state.PatchUnknown)

	// Server-derived (Computed-only): adopt verbatim.
	state.ReleaseDate = int64ValueOrNull(g.ReleaseDate)
	state.IncrementalUpdate = helpers.BoolPointerValueOrNull(g.IncrementalUpdate)
	state.Reboot = helpers.BoolPointerValueOrNull(g.Reboot)
	state.MinimumOS = helpers.StringPointerValueOrNull(g.MinimumOs)

	killApps, d := flattenKillApps(ctx, g.KillApps)
	diags.Append(d...)
	state.KillApps = killApps

	return diags
}

// flattenKillApps maps the server-derived kill_apps list into a Computed
// List<Object>. Returns a null list when the wire block is absent / empty.
func flattenKillApps(ctx context.Context, ka *proclassic.PatchPolicyGeneralKillApps) (types.List, diag.Diagnostics) {
	objType := types.ObjectType{AttrTypes: killAppAttrTypes}
	if ka == nil || ka.KillApp == nil || len(*ka.KillApp) == 0 {
		return types.ListNull(objType), nil
	}

	objects := make([]attr.Value, 0, len(*ka.KillApp))
	for _, item := range *ka.KillApp {
		obj, d := types.ObjectValue(killAppAttrTypes, map[string]attr.Value{
			"kill_app_name":      helpers.StringPointerValueOrNull(item.KillAppName),
			"kill_app_bundle_id": helpers.StringPointerValueOrNull(item.KillAppBundleID),
		})
		if d.HasError() {
			return types.ListNull(objType), d
		}
		objects = append(objects, obj)
	}
	return types.ListValue(objType, objects)
}

func flattenScope(ctx context.Context, s *proclassic.PatchPolicyScope, state *PatchPolicyScopeModel) {
	if state.Targets != nil {
		state.Targets.AllComputers = helpers.PreferCurrentBoolPointer(s.AllComputers, state.Targets.AllComputers)
		state.Targets.ComputerIDs = flattenComputerSet(ctx, computerSlice(s.Computers))
		state.Targets.ComputerGroupIDs = flattenIDNameSet(ctx, computerGroupSlice(s.ComputerGroups))
		state.Targets.BuildingIDs = flattenIDNameSet(ctx, buildingSlice(s.Buildings))
		state.Targets.DepartmentIDs = flattenIDNameSet(ctx, departmentSlice(s.Departments))
	}

	if state.Limitations != nil && s.Limitations != nil {
		l := s.Limitations
		state.Limitations.NetworkSegmentIDs = flattenIDNameSet(ctx, limitationsNetworkSegmentSlice(l.NetworkSegments))
		state.Limitations.IbeaconIDs = flattenIDNameSet(ctx, limitationsIbeaconSlice(l.Ibeacons))
	}

	if state.Exclusions != nil && s.Exclusions != nil {
		e := s.Exclusions
		state.Exclusions.ComputerIDs = flattenExclComputerSet(ctx, exclComputerSlice(e.Computers))
		state.Exclusions.ComputerGroupIDs = flattenIDNameSet(ctx, exclComputerGroupSlice(e.ComputerGroups))
		state.Exclusions.BuildingIDs = flattenIDNameSet(ctx, exclBuildingSlice(e.Buildings))
		state.Exclusions.DepartmentIDs = flattenIDNameSet(ctx, exclDepartmentSlice(e.Departments))
		state.Exclusions.NetworkSegmentIDs = flattenIDNameSet(ctx, exclNetworkSegmentSlice(e.NetworkSegments))
		state.Exclusions.IbeaconIDs = flattenIDNameSet(ctx, exclIbeaconSlice(e.Ibeacons))
	}
}

func flattenUserInteraction(ui *proclassic.PatchPolicyUserInteraction, state *PatchPolicyUserInteractionModel) {
	state.InstallButtonText = helpers.PreferCurrentStringPointer(ui.InstallButtonText, state.InstallButtonText)
	state.SelfServiceDescription = helpers.PreferCurrentStringPointer(ui.SelfServiceDescription, state.SelfServiceDescription)
	if ui.SelfServiceIcon != nil {
		state.SelfServiceIconID = helpers.PreferCurrentStringPointer(helpers.StringFromIntPtr(ui.SelfServiceIcon.ID), state.SelfServiceIconID)
	} else {
		state.SelfServiceIconID = helpers.PreferCurrentStringPointer(nil, state.SelfServiceIconID)
	}

	// A managed sub-block resolves ALL its Computed leaves even when the server
	// omits that sub-block on the post-PUT GET (n/d/g may be nil). preferCurrent
	// treats a nil API value as "no echo" → keeps a configured value, else null.
	// Guarding on `ui.X != nil` (the old shape) skipped the branch wholesale and
	// left an unset Computed leaf stuck Unknown → "invalid result object after
	// apply" (wire-observed: the server can return <user_interaction> without a
	// <notifications> child right after a PUT).
	if state.Notifications != nil {
		n := ui.Notifications
		var (
			nEnabled              *bool
			nSubject, nMsg, nType *string
			nReminders            *proclassic.PatchPolicyUserInteractionNotificationsReminders
		)
		if n != nil {
			nEnabled, nSubject, nMsg, nType, nReminders = n.NotificationEnabled, n.NotificationSubject, n.NotificationMessage, n.NotificationType, n.Reminders
		}
		state.Notifications.Enabled = helpers.PreferCurrentBoolPointer(nEnabled, state.Notifications.Enabled)
		state.Notifications.Subject = helpers.PreferCurrentStringPointer(nSubject, state.Notifications.Subject)
		state.Notifications.Message = helpers.PreferCurrentStringPointer(nMsg, state.Notifications.Message)
		state.Notifications.Type = helpers.PreferCurrentStringPointer(nType, state.Notifications.Type)
		if state.Notifications.Reminders != nil {
			var (
				rEnabled *bool
				rFreq    *int
			)
			if nReminders != nil {
				rEnabled, rFreq = nReminders.NotificationRemindersEnabled, nReminders.NotificationReminderFrequency
			}
			state.Notifications.Reminders.Enabled = helpers.PreferCurrentBoolPointer(rEnabled, state.Notifications.Reminders.Enabled)
			state.Notifications.Reminders.Frequency = preferCurrentInt64Pointer(rFreq, state.Notifications.Reminders.Frequency)
		}
	}

	if state.Deadlines != nil {
		var (
			dEnabled *bool
			dPeriod  *int
		)
		if ui.Deadlines != nil {
			dEnabled, dPeriod = ui.Deadlines.DeadlineEnabled, ui.Deadlines.DeadlinePeriod
		}
		state.Deadlines.Enabled = helpers.PreferCurrentBoolPointer(dEnabled, state.Deadlines.Enabled)
		state.Deadlines.Period = preferCurrentInt64Pointer(dPeriod, state.Deadlines.Period)
	}

	if state.GracePeriod != nil {
		var (
			gDuration      *int
			gSubject, gMsg *string
		)
		if ui.GracePeriod != nil {
			gDuration, gSubject, gMsg = ui.GracePeriod.GracePeriodDuration, ui.GracePeriod.NotificationCenterSubject, ui.GracePeriod.Message
		}
		state.GracePeriod.Duration = preferCurrentInt64Pointer(gDuration, state.GracePeriod.Duration)
		state.GracePeriod.NotificationCenterSubject = helpers.PreferCurrentStringPointer(gSubject, state.GracePeriod.NotificationCenterSubject)
		state.GracePeriod.Message = helpers.PreferCurrentStringPointer(gMsg, state.GracePeriod.Message)
	}
}

// ---- scope sub-slice accessors -------------------------------------------------

func computerSlice(c *proclassic.PatchPolicyScopeComputers) *[]proclassic.PatchPolicyScopeComputersComputerItem {
	if c == nil {
		return nil
	}
	return c.Computer
}

func computerGroupSlice(g *proclassic.PatchPolicyScopeComputerGroups) *[]proclassic.IDName {
	if g == nil {
		return nil
	}
	return g.ComputerGroup
}

func buildingSlice(b *proclassic.PatchPolicyScopeBuildings) *[]proclassic.IDName {
	if b == nil {
		return nil
	}
	return b.Building
}

func departmentSlice(d *proclassic.PatchPolicyScopeDepartments) *[]proclassic.IDName {
	if d == nil {
		return nil
	}
	return d.Department
}

func limitationsNetworkSegmentSlice(n *proclassic.PatchPolicyScopeLimitationsNetworkSegments) *[]proclassic.IDName {
	if n == nil {
		return nil
	}
	return n.NetworkSegment
}

func limitationsIbeaconSlice(i *proclassic.PatchPolicyScopeLimitationsIbeacons) *[]proclassic.IDName {
	if i == nil {
		return nil
	}
	return i.Ibeacon
}

func exclComputerSlice(c *proclassic.PatchPolicyScopeExclusionsComputers) *[]proclassic.PatchPolicyScopeExclusionsComputersComputerItem {
	if c == nil {
		return nil
	}
	return c.Computer
}

func exclComputerGroupSlice(g *proclassic.PatchPolicyScopeExclusionsComputerGroups) *[]proclassic.IDName {
	if g == nil {
		return nil
	}
	return g.ComputerGroup
}

func exclBuildingSlice(b *proclassic.PatchPolicyScopeExclusionsBuildings) *[]proclassic.IDName {
	if b == nil {
		return nil
	}
	return b.Building
}

func exclDepartmentSlice(d *proclassic.PatchPolicyScopeExclusionsDepartments) *[]proclassic.IDName {
	if d == nil {
		return nil
	}
	return d.Department
}

func exclNetworkSegmentSlice(n *proclassic.PatchPolicyScopeExclusionsNetworkSegments) *[]proclassic.IDName {
	if n == nil {
		return nil
	}
	return n.NetworkSegment
}

func exclIbeaconSlice(i *proclassic.PatchPolicyScopeExclusionsIbeacons) *[]proclassic.IDName {
	if i == nil {
		return nil
	}
	return i.Ibeacon
}

// ---- set flatteners ------------------------------------------------------------

func flattenIDNameSet(ctx context.Context, items *[]proclassic.IDName) types.Set {
	out, _ := scope.FlattenIDSlice(ctx, items, func(i proclassic.IDName) *int { return i.ID })
	return out
}

func flattenComputerSet(ctx context.Context, items *[]proclassic.PatchPolicyScopeComputersComputerItem) types.Set {
	out, _ := scope.FlattenIDSlice(ctx, items, func(i proclassic.PatchPolicyScopeComputersComputerItem) *int { return i.ID })
	return out
}

func flattenExclComputerSet(ctx context.Context, items *[]proclassic.PatchPolicyScopeExclusionsComputersComputerItem) types.Set {
	out, _ := scope.FlattenIDSlice(ctx, items, func(i proclassic.PatchPolicyScopeExclusionsComputersComputerItem) *int { return i.ID })
	return out
}
