// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package macos_configuration_profile

import (
	"context"
	"strconv"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/payloadhelpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/plisthelpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

// assignResourceModel populates a ResourceModel from the SDK response.
// Optional sub-blocks are only refreshed when the caller (plan or prior
// state) already manages them — otherwise a server-populated block would
// trip the framework's "produced inconsistent result after apply" check.
func assignResourceModel(ctx context.Context, state *ResourceModel, p *proclassic.OsXConfigurationProfile) diag.Diagnostics {
	var diags diag.Diagnostics
	if p == nil {
		return diags
	}

	if p.ID != nil {
		state.ID = helpers.StringValueFromIntPtr(p.ID)
	} else if p.General != nil && p.General.ID != nil {
		state.ID = helpers.StringValueFromIntPtr(p.General.ID)
	}

	if state.General == nil {
		state.General = &GeneralModel{}
	}
	flattenGeneral(p.General, state.General)

	if state.Scope != nil && p.Scope != nil {
		diags.Append(flattenScope(ctx, p.Scope, state.Scope)...)
	}
	if state.SelfService != nil && p.SelfService != nil {
		flattenSelfService(p.SelfService, state.SelfService)
	}
	return diags
}

func flattenGeneral(g *proclassic.OsXConfigurationProfileGeneral, state *GeneralModel) {
	if g == nil {
		return
	}
	if g.ID != nil {
		state.ID = helpers.StringValueFromIntPtr(g.ID)
	}
	state.Name = helpers.ReconcileOptionalStringPointer(g.Name, state.Name)
	state.Description = helpers.ReconcileOptionalStringPointer(g.Description, state.Description)
	state.UserRemovable = helpers.ReconcileOptionalBoolPointer(g.UserRemovable, state.UserRemovable)
	// RedeployOnUpdate: the Classic API always returns "Newly Assigned" on
	// read regardless of what was sent on Update — the field is effectively
	// write-only on the wire (the API accepts "All" on PUT but never echoes
	// it back on GET). Same wire bug deploymenttheory PR #888 addresses.
	//
	// Adapted for Optional+Computed: write from wire only when state is
	// null/unknown (Create with no user input, or import). Otherwise keep
	// the user-authored value so a subsequent plan after
	// `redeploy_on_update = "All"` does not snap back to "Newly Assigned"
	// on every refresh. The Optional+Computed shape requires a known value
	// after apply, hence the wire fallback when the user omitted the
	// attribute.
	if state.RedeployOnUpdate.IsNull() || state.RedeployOnUpdate.IsUnknown() {
		state.RedeployOnUpdate = helpers.StringPointerValueOrNull(g.RedeployOnUpdate)
	}
	state.UUID = helpers.StringPointerValueOrNull(g.UUID)
	// Self-healing payload assignment. The framework requires
	// plan.Payloads == final state.Payloads for the Required `payloads`
	// attribute, but the server re-serialises the plist and re-assigns
	// top-level UUIDs — so the byte-form changes on every write. Keep
	// the user-authored bytes when the server's canonical form is
	// semantically equivalent; only overwrite with the server form when
	// genuine out-of-band drift is detected.
	if g.Payloads != nil {
		server := string(*g.Payloads)
		keep := false
		if !state.Payloads.IsNull() && !state.Payloads.IsUnknown() {
			if eq, err := payloadhelpers.PayloadsSemanticallyEqual([]byte(state.Payloads.ValueString()), []byte(server)); err == nil && eq {
				keep = true
			}
		}
		if !keep {
			state.Payloads = types.StringValue(string(plisthelpers.CanonicalisePlistXML([]byte(server))))
		}
	}
	state.DistributionMethod = helpers.ReconcileOptionalStringPointer(g.DistributionMethod, state.DistributionMethod)

	// Level wire-read translation. The wire returns System for Computer Level
	// and User for User Level — translate back to the UI-canonical strings.
	if g.Level != nil {
		state.Level = types.StringValue(levelFromWireRead(*g.Level))
	} else if !helpers.IsConfiguredValue(state.Level) {
		state.Level = types.StringNull()
	}

	// Category — emit the assigned ID; "no category" sentinel "-1" stays
	// visible so users see Jamf's default rather than null.
	if g.Category != nil {
		state.CategoryID = helpers.StringValueFromIntPtr(g.Category.ID)
		state.CategoryName = helpers.DerivedRefName(g.Category.ID, g.Category.Name)
	}
	if g.Site != nil {
		state.SiteID = helpers.StringValueFromIntPtr(g.Site.ID)
		state.SiteName = helpers.DerivedRefName(g.Site.ID, g.Site.Name)
	}
}

func flattenScope(ctx context.Context, s *proclassic.OsXConfigurationProfileScope, state *scope.ComputerScopeModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if s == nil {
		return diags
	}

	// Targets are gated on caller management, mirroring the limitations /
	// exclusions sub-blocks below: populating a targets block the user did not
	// declare would violate the framework's "produced inconsistent result after
	// apply" check (plan said null, we would return a populated object).
	if state.Targets != nil {
		state.Targets.AllComputers = helpers.ReconcileOptionalBoolPointer(s.AllComputers, state.Targets.AllComputers)
		state.Targets.AllJssUsers = helpers.ReconcileOptionalBoolPointer(s.AllJssUsers, state.Targets.AllJssUsers)

		if s.Computers != nil {
			v, d := scope.FlattenIDSlice(ctx, s.Computers.Computer, func(c proclassic.OsXConfigurationProfileScopeComputersComputerItem) *int {
				return c.ID
			})
			diags.Append(d...)
			state.Targets.ComputerIDs = v
		} else {
			state.Targets.ComputerIDs = scope.EmptyStringSet()
		}
		if s.ComputerGroups != nil {
			v, d := scope.FlattenIDSlice(ctx, s.ComputerGroups.ComputerGroup, func(c proclassic.IDName) *int { return c.ID })
			diags.Append(d...)
			state.Targets.ComputerGroupIDs = v
		} else {
			state.Targets.ComputerGroupIDs = scope.EmptyStringSet()
		}
		if s.Buildings != nil {
			v, d := scope.FlattenIDSlice(ctx, s.Buildings.Building, func(c proclassic.IDName) *int { return c.ID })
			diags.Append(d...)
			state.Targets.BuildingIDs = v
		} else {
			state.Targets.BuildingIDs = scope.EmptyStringSet()
		}
		if s.Departments != nil {
			v, d := scope.FlattenIDSlice(ctx, s.Departments.Department, func(c proclassic.IDName) *int { return c.ID })
			diags.Append(d...)
			state.Targets.DepartmentIDs = v
		} else {
			state.Targets.DepartmentIDs = scope.EmptyStringSet()
		}
		if s.JssUsers != nil {
			v, d := scope.FlattenIDSlice(ctx, s.JssUsers.User, func(c proclassic.IDName) *int { return c.ID })
			diags.Append(d...)
			state.Targets.UserIDs = v
		} else {
			state.Targets.UserIDs = scope.EmptyStringSet()
		}
		if s.JssUserGroups != nil {
			v, d := scope.FlattenIDSlice(ctx, s.JssUserGroups.JssUserGroup, func(c proclassic.IDName) *int { return c.ID })
			diags.Append(d...)
			state.Targets.UserGroupIDs = v
		} else {
			state.Targets.UserGroupIDs = scope.EmptyStringSet()
		}
	}

	if state.Limitations != nil && s.Limitations != nil {
		diags.Append(flattenScopeLimitations(ctx, s.Limitations, state.Limitations)...)
	}
	if state.Exclusions != nil && s.Exclusions != nil {
		diags.Append(flattenScopeExclusions(ctx, s.Exclusions, state.Exclusions)...)
	}
	return diags
}

func flattenScopeLimitations(ctx context.Context, l *proclassic.OsXConfigurationProfileScopeLimitations, state *scope.ComputerScopeLimitationsModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if l == nil {
		return diags
	}
	if l.NetworkSegments != nil {
		v, d := scope.FlattenIDSlice(ctx, l.NetworkSegments.NetworkSegment, func(c proclassic.OsXConfigurationProfileScopeLimitationsNetworkSegmentsNetworkSegmentItem) *int {
			return c.ID
		})
		diags.Append(d...)
		state.NetworkSegmentIDs = v
	} else {
		state.NetworkSegmentIDs = scope.EmptyStringSet()
	}
	if l.Ibeacons != nil {
		v, d := scope.FlattenIDSlice(ctx, l.Ibeacons.Ibeacon, func(c proclassic.IDName) *int { return c.ID })
		diags.Append(d...)
		state.IbeaconIDs = v
	} else {
		state.IbeaconIDs = scope.EmptyStringSet()
	}
	if l.Users != nil {
		v, d := scope.FlattenNameSlice(ctx, l.Users.User, func(c proclassic.IDName) *string { return c.Name })
		diags.Append(d...)
		state.DirectoryServiceOrLocalUserNames = v
	} else {
		state.DirectoryServiceOrLocalUserNames = scope.EmptyStringSet()
	}
	if l.UserGroups != nil {
		v, d := scope.FlattenNameSlice(ctx, l.UserGroups.UserGroup, func(c proclassic.IDName) *string { return c.Name })
		diags.Append(d...)
		state.DirectoryServiceUserGroupNames = v
	} else {
		state.DirectoryServiceUserGroupNames = scope.EmptyStringSet()
	}
	return diags
}

func flattenScopeExclusions(ctx context.Context, e *proclassic.OsXConfigurationProfileScopeExclusions, state *scope.ComputerScopeExclusionsModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if e == nil {
		return diags
	}
	if e.Computers != nil {
		v, d := scope.FlattenIDSlice(ctx, e.Computers.Computer, func(c proclassic.OsXConfigurationProfileScopeExclusionsComputersComputerItem) *int {
			return c.ID
		})
		diags.Append(d...)
		state.ComputerIDs = v
	} else {
		state.ComputerIDs = scope.EmptyStringSet()
	}
	if e.ComputerGroups != nil {
		v, d := scope.FlattenIDSlice(ctx, e.ComputerGroups.ComputerGroup, func(c proclassic.IDName) *int { return c.ID })
		diags.Append(d...)
		state.ComputerGroupIDs = v
	} else {
		state.ComputerGroupIDs = scope.EmptyStringSet()
	}
	if e.Buildings != nil {
		v, d := scope.FlattenIDSlice(ctx, e.Buildings.Building, func(c proclassic.IDName) *int { return c.ID })
		diags.Append(d...)
		state.BuildingIDs = v
	} else {
		state.BuildingIDs = scope.EmptyStringSet()
	}
	if e.Departments != nil {
		v, d := scope.FlattenIDSlice(ctx, e.Departments.Department, func(c proclassic.IDName) *int { return c.ID })
		diags.Append(d...)
		state.DepartmentIDs = v
	} else {
		state.DepartmentIDs = scope.EmptyStringSet()
	}
	if e.JssUsers != nil {
		v, d := scope.FlattenIDSlice(ctx, e.JssUsers.User, func(c proclassic.IDName) *int { return c.ID })
		diags.Append(d...)
		state.UserIDs = v
	} else {
		state.UserIDs = scope.EmptyStringSet()
	}
	if e.JssUserGroups != nil {
		v, d := scope.FlattenIDSlice(ctx, e.JssUserGroups.UserGroup, func(c proclassic.IDName) *int { return c.ID })
		diags.Append(d...)
		state.UserGroupIDs = v
	} else {
		state.UserGroupIDs = scope.EmptyStringSet()
	}
	if e.NetworkSegments != nil {
		v, d := scope.FlattenIDSlice(ctx, e.NetworkSegments.NetworkSegment, func(c proclassic.OsXConfigurationProfileScopeExclusionsNetworkSegmentsNetworkSegmentItem) *int {
			return c.ID
		})
		diags.Append(d...)
		state.NetworkSegmentIDs = v
	} else {
		state.NetworkSegmentIDs = scope.EmptyStringSet()
	}
	if e.Ibeacons != nil {
		v, d := scope.FlattenIDSlice(ctx, e.Ibeacons.Ibeacon, func(c proclassic.IDName) *int { return c.ID })
		diags.Append(d...)
		state.IbeaconIDs = v
	} else {
		state.IbeaconIDs = scope.EmptyStringSet()
	}
	if e.Users != nil {
		v, d := scope.FlattenNameSlice(ctx, e.Users.User, func(c proclassic.OsXConfigurationProfileScopeExclusionsUsersUserItem) *string {
			return c.Name
		})
		diags.Append(d...)
		state.DirectoryServiceOrLocalUserNames = v
	} else {
		state.DirectoryServiceOrLocalUserNames = scope.EmptyStringSet()
	}
	if e.UserGroups != nil {
		v, d := scope.FlattenNameSlice(ctx, e.UserGroups.UserGroup, func(c proclassic.IDName) *string { return c.Name })
		diags.Append(d...)
		state.DirectoryServiceUserGroupNames = v
	} else {
		state.DirectoryServiceUserGroupNames = scope.EmptyStringSet()
	}
	return diags
}

func flattenSelfService(ss *proclassic.OsXConfigurationProfileSelfService, state *SelfServiceModel) {
	if ss == nil {
		return
	}
	state.SelfServiceDisplayName = helpers.ReconcileOptionalStringPointer(ss.SelfServiceDisplayName, state.SelfServiceDisplayName)
	state.InstallButtonText = helpers.ReconcileOptionalStringPointer(ss.InstallButtonText, state.InstallButtonText)
	state.SelfServiceDescription = helpers.ReconcileOptionalStringPointer(ss.SelfServiceDescription, state.SelfServiceDescription)
	state.EnsureUsersViewDescription = helpers.ReconcileOptionalBoolPointer(ss.ForceUsersToViewDescription, state.EnsureUsersViewDescription)
	state.FeatureOnMainPage = helpers.ReconcileOptionalBoolPointer(ss.FeatureOnMainPage, state.FeatureOnMainPage)
	state.NotificationSubject = helpers.PreserveStringWhenWireEmpty(ss.NotificationSubject, state.NotificationSubject)
	state.NotificationMessage = helpers.PreserveStringWhenWireEmpty(ss.NotificationMessage, state.NotificationMessage)

	if ss.Notification != nil {
		state.DisplayNotifications = helpers.ReconcileOptionalBoolPointer(ss.Notification.Enabled, state.DisplayNotifications)
		state.NotificationLocation = helpers.ReconcileOptionalStringPointer(ss.Notification.Method, state.NotificationLocation)
	}

	if ss.Security != nil {
		state.RemovalDisallowed = helpers.ReconcileOptionalStringPointer(ss.Security.RemovalDisallowed, state.RemovalDisallowed)
	}

	if ss.SelfServiceCategories != nil && ss.SelfServiceCategories.Category != nil && len(*ss.SelfServiceCategories.Category) > 0 {
		cats := *ss.SelfServiceCategories.Category
		items := make([]SelfServiceCategoryItem, 0, len(cats))
		for _, c := range cats {
			it := SelfServiceCategoryItem{
				ID:        idPointerToString(c.ID),
				Name:      helpers.StringPointerValueOrNull(c.Name),
				DisplayIn: helpers.BoolPointerValueOrNull(c.DisplayIn),
				FeatureIn: helpers.BoolPointerValueOrNull(c.FeatureIn),
			}
			items = append(items, it)
		}
		state.Categories = items
	}
}

func idPointerToString(id *int) types.String {
	if id == nil {
		return types.StringNull()
	}
	return types.StringValue(strconv.Itoa(*id))
}
