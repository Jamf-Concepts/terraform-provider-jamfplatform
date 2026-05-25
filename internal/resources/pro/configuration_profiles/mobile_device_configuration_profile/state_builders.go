// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_configuration_profile

import (
	"context"
	"strconv"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/configuration_profiles/payloadhelpers"
)

func assignResourceModel(ctx context.Context, state *ResourceModel, p *proclassic.MobileDeviceConfigurationProfile) diag.Diagnostics {
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

func flattenGeneral(g *proclassic.MobileDeviceConfigurationProfileGeneral, state *GeneralModel) {
	if g == nil {
		return
	}
	if g.ID != nil {
		state.ID = helpers.StringValueFromIntPtr(g.ID)
	}
	state.Name = helpers.ReconcileOptionalStringPointer(g.Name, state.Name)
	state.Description = helpers.ReconcileOptionalStringPointer(g.Description, state.Description)
	// RedeployOnUpdate: the Classic API always returns "Newly Assigned" on read
	// regardless of what was PUT — write-only on the wire. Preserve the
	// user-authored value so a plan after `redeploy_on_update = "All"` does
	// not snap back to "Newly Assigned" on every refresh.
	if state.RedeployOnUpdate.IsNull() || state.RedeployOnUpdate.IsUnknown() {
		state.RedeployOnUpdate = helpers.StringPointerValueOrNull(g.RedeployOnUpdate)
	}
	if g.RedeployDaysBeforeCertificateExpires != nil {
		state.RedeployDaysBeforeCertificateExpires = types.Int64Value(int64(*g.RedeployDaysBeforeCertificateExpires))
	}
	state.UUID = helpers.StringPointerValueOrNull(g.UUID)
	// Self-healing payload: keep user-authored bytes when the server's
	// canonical form is semantically equivalent; only overwrite when genuine
	// out-of-band drift is detected.
	if g.Payloads != nil {
		server := *g.Payloads
		keep := false
		if !state.Payloads.IsNull() && !state.Payloads.IsUnknown() {
			if eq, err := payloadhelpers.PayloadsSemanticallyEqual([]byte(state.Payloads.ValueString()), []byte(server)); err == nil && eq {
				keep = true
			}
		}
		if !keep {
			state.Payloads = types.StringValue(server)
		}
	}
	// Wire element is <deployment_method>; TF attribute is distribution_method (UI-canonical).
	state.DistributionMethod = helpers.ReconcileOptionalStringPointer(g.DeploymentMethod, state.DistributionMethod)

	if g.Level != nil {
		state.Level = types.StringValue(levelFromWireRead(*g.Level))
	} else if !helpers.IsConfiguredValue(state.Level) {
		state.Level = types.StringNull()
	}

	if g.Category != nil {
		state.CategoryID = helpers.StringValueFromIntPtr(g.Category.ID)
		state.CategoryName = helpers.StringPointerValueOrNull(g.Category.Name)
	}
	if g.Site != nil {
		state.SiteID = helpers.StringValueFromIntPtr(g.Site.ID)
		state.SiteName = helpers.StringPointerValueOrNull(g.Site.Name)
	}
}

func flattenScope(ctx context.Context, s *proclassic.MobileDeviceConfigurationProfileScope, state *ScopeModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if s == nil {
		return diags
	}

	state.AllMobileDevices = helpers.ReconcileOptionalBoolPointer(s.AllMobileDevices, state.AllMobileDevices)
	state.AllJssUsers = helpers.ReconcileOptionalBoolPointer(s.AllJssUsers, state.AllJssUsers)

	if s.MobileDevices != nil {
		v, d := scope.FlattenIDSlice(ctx, s.MobileDevices.MobileDevice, func(c proclassic.MobileDeviceConfigurationProfileScopeMobileDevicesMobileDeviceItem) *int {
			return c.ID
		})
		diags.Append(d...)
		state.MobileDeviceIDs = v
	} else {
		state.MobileDeviceIDs = types.SetNull(types.StringType)
	}
	if s.MobileDeviceGroups != nil {
		v, d := scope.FlattenIDSlice(ctx, s.MobileDeviceGroups.MobileDeviceGroup, func(c proclassic.IDName) *int { return c.ID })
		diags.Append(d...)
		state.MobileDeviceGroupIDs = v
	} else {
		state.MobileDeviceGroupIDs = types.SetNull(types.StringType)
	}
	if s.Buildings != nil {
		v, d := scope.FlattenIDSlice(ctx, s.Buildings.Building, func(c proclassic.IDName) *int { return c.ID })
		diags.Append(d...)
		state.BuildingIDs = v
	} else {
		state.BuildingIDs = types.SetNull(types.StringType)
	}
	if s.Departments != nil {
		v, d := scope.FlattenIDSlice(ctx, s.Departments.Department, func(c proclassic.IDName) *int { return c.ID })
		diags.Append(d...)
		state.DepartmentIDs = v
	} else {
		state.DepartmentIDs = types.SetNull(types.StringType)
	}
	if s.JssUsers != nil {
		v, d := scope.FlattenIDSlice(ctx, s.JssUsers.User, func(c proclassic.IDName) *int { return c.ID })
		diags.Append(d...)
		state.UserIDs = v
	} else {
		state.UserIDs = types.SetNull(types.StringType)
	}
	if s.JssUserGroups != nil {
		v, d := scope.FlattenIDSlice(ctx, s.JssUserGroups.UserGroup, func(c proclassic.IDName) *int { return c.ID })
		diags.Append(d...)
		state.UserGroupIDs = v
	} else {
		state.UserGroupIDs = types.SetNull(types.StringType)
	}

	if state.Limitations != nil && s.Limitations != nil {
		diags.Append(flattenScopeLimitations(ctx, s.Limitations, state.Limitations)...)
	}
	if state.Exclusions != nil && s.Exclusions != nil {
		diags.Append(flattenScopeExclusions(ctx, s.Exclusions, state.Exclusions)...)
	}
	return diags
}

func flattenScopeLimitations(ctx context.Context, l *proclassic.MobileDeviceConfigurationProfileScopeLimitations, state *ScopeLimitationsModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if l == nil {
		return diags
	}
	if l.NetworkSegments != nil {
		v, d := scope.FlattenIDSlice(ctx, l.NetworkSegments.NetworkSegment, func(c proclassic.IDName) *int { return c.ID })
		diags.Append(d...)
		state.NetworkSegmentIDs = v
	} else {
		state.NetworkSegmentIDs = types.SetNull(types.StringType)
	}
	if l.Ibeacons != nil {
		v, d := scope.FlattenIDSlice(ctx, l.Ibeacons.Ibeacon, func(c proclassic.IDName) *int { return c.ID })
		diags.Append(d...)
		state.IbeaconIDs = v
	} else {
		state.IbeaconIDs = types.SetNull(types.StringType)
	}
	if l.Users != nil {
		v, d := scope.FlattenNameSlice(ctx, l.Users.User, func(c proclassic.IDName) *string { return c.Name })
		diags.Append(d...)
		state.DirectoryServiceOrLocalUserNames = v
	} else {
		state.DirectoryServiceOrLocalUserNames = types.SetNull(types.StringType)
	}
	if l.UserGroups != nil {
		v, d := scope.FlattenNameSlice(ctx, l.UserGroups.UserGroup, func(c proclassic.IDName) *string { return c.Name })
		diags.Append(d...)
		state.DirectoryServiceUserGroupNames = v
	} else {
		state.DirectoryServiceUserGroupNames = types.SetNull(types.StringType)
	}
	return diags
}

func flattenScopeExclusions(ctx context.Context, e *proclassic.MobileDeviceConfigurationProfileScopeExclusions, state *ScopeExclusionsModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if e == nil {
		return diags
	}
	if e.MobileDevices != nil {
		v, d := scope.FlattenIDSlice(ctx, e.MobileDevices.MobileDevice, func(c proclassic.MobileDeviceConfigurationProfileScopeExclusionsMobileDevicesMobileDeviceItem) *int {
			return c.ID
		})
		diags.Append(d...)
		state.MobileDeviceIDs = v
	} else {
		state.MobileDeviceIDs = types.SetNull(types.StringType)
	}
	if e.MobileDeviceGroups != nil {
		v, d := scope.FlattenIDSlice(ctx, e.MobileDeviceGroups.MobileDeviceGroup, func(c proclassic.IDName) *int { return c.ID })
		diags.Append(d...)
		state.MobileDeviceGroupIDs = v
	} else {
		state.MobileDeviceGroupIDs = types.SetNull(types.StringType)
	}
	if e.Buildings != nil {
		v, d := scope.FlattenIDSlice(ctx, e.Buildings.Building, func(c proclassic.IDName) *int { return c.ID })
		diags.Append(d...)
		state.BuildingIDs = v
	} else {
		state.BuildingIDs = types.SetNull(types.StringType)
	}
	if e.Departments != nil {
		v, d := scope.FlattenIDSlice(ctx, e.Departments.Department, func(c proclassic.IDName) *int { return c.ID })
		diags.Append(d...)
		state.DepartmentIDs = v
	} else {
		state.DepartmentIDs = types.SetNull(types.StringType)
	}
	if e.JssUsers != nil {
		v, d := scope.FlattenIDSlice(ctx, e.JssUsers.User, func(c proclassic.IDName) *int { return c.ID })
		diags.Append(d...)
		state.UserIDs = v
	} else {
		state.UserIDs = types.SetNull(types.StringType)
	}
	if e.JssUserGroups != nil {
		v, d := scope.FlattenIDSlice(ctx, e.JssUserGroups.UserGroup, func(c proclassic.IDName) *int { return c.ID })
		diags.Append(d...)
		state.UserGroupIDs = v
	} else {
		state.UserGroupIDs = types.SetNull(types.StringType)
	}
	if e.NetworkSegments != nil {
		v, d := scope.FlattenIDSlice(ctx, e.NetworkSegments.NetworkSegment, func(c proclassic.MobileDeviceConfigurationProfileScopeExclusionsNetworkSegmentsNetworkSegmentItem) *int {
			return c.ID
		})
		diags.Append(d...)
		state.NetworkSegmentIDs = v
	} else {
		state.NetworkSegmentIDs = types.SetNull(types.StringType)
	}
	if e.Ibeacons != nil {
		v, d := scope.FlattenIDSlice(ctx, e.Ibeacons.Ibeacon, func(c proclassic.IDName) *int { return c.ID })
		diags.Append(d...)
		state.IbeaconIDs = v
	} else {
		state.IbeaconIDs = types.SetNull(types.StringType)
	}
	if e.Users != nil {
		v, d := scope.FlattenNameSlice(ctx, e.Users.User, func(c proclassic.MobileDeviceConfigurationProfileScopeExclusionsUsersUserItem) *string {
			return c.Name
		})
		diags.Append(d...)
		state.DirectoryServiceOrLocalUserNames = v
	} else {
		state.DirectoryServiceOrLocalUserNames = types.SetNull(types.StringType)
	}
	if e.UserGroups != nil {
		v, d := scope.FlattenNameSlice(ctx, e.UserGroups.UserGroup, func(c proclassic.IDName) *string { return c.Name })
		diags.Append(d...)
		state.DirectoryServiceUserGroupNames = v
	} else {
		state.DirectoryServiceUserGroupNames = types.SetNull(types.StringType)
	}
	return diags
}

func flattenSelfService(ss *proclassic.MobileDeviceConfigurationProfileSelfService, state *SelfServiceModel) {
	if ss == nil {
		return
	}
	state.SelfServiceDescription = helpers.PreserveStringWhenWireEmpty(ss.SelfServiceDescription, state.SelfServiceDescription)
	state.FeatureOnMainPage = helpers.ReconcileOptionalBoolPointer(ss.FeatureOnMainPage, state.FeatureOnMainPage)

	if ss.Security != nil {
		state.RemovalDisallowed = helpers.ReconcileOptionalStringPointer(ss.Security.RemovalDisallowed, state.RemovalDisallowed)
		state.AuthorizationPassword = helpers.PreserveStringWhenWireEmpty(ss.Security.Password, state.AuthorizationPassword)
	}

	if ss.SelfServiceCategories != nil && ss.SelfServiceCategories.Category != nil && len(*ss.SelfServiceCategories.Category) > 0 {
		cats := *ss.SelfServiceCategories.Category
		items := make([]SelfServiceCategoryItem, 0, len(cats))
		for _, c := range cats {
			it := SelfServiceCategoryItem{
				ID:   idPointerToString(c.ID),
				Name: helpers.StringPointerValueOrNull(c.Name),
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
