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
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/payloadhelpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/plisthelpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

// assignResourceModel populates a ResourceModel from the SDK response.
// Optional sub-blocks are only refreshed when the caller (plan or prior
// state) already manages them — otherwise a server-populated block would
// trip the framework's "produced inconsistent result after apply" check.
//
// includeUnmanaged inverts that gate for the list resource's
// config-generation path (terraform query -generate-config-out): there is no
// plan to stay consistent with, so every wire-present optional section is
// allocated and hydrated, yielding a complete exported config rather than a
// general-only one. CRUD callers pass false.
func assignResourceModel(ctx context.Context, state *ResourceModel, p *proclassic.MobileDeviceConfigurationProfile, includeUnmanaged bool) diag.Diagnostics {
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

	if includeUnmanaged && state.Scope == nil && p.Scope != nil {
		state.Scope = &scope.MobileScopeModel{}
	}
	if state.Scope != nil && p.Scope != nil {
		diags.Append(flattenScope(ctx, p.Scope, state.Scope, includeUnmanaged)...)
	}
	if includeUnmanaged && state.SelfService == nil && p.SelfService != nil {
		state.SelfService = &SelfServiceModel{}
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
	// Wire element is <deployment_method>; TF attribute is distribution_method (UI-canonical).
	state.DistributionMethod = helpers.ReconcileOptionalStringPointer(g.DeploymentMethod, state.DistributionMethod)

	if g.Level != nil {
		state.Level = types.StringValue(levelFromWireRead(*g.Level))
	} else if !helpers.IsConfiguredValue(state.Level) {
		state.Level = types.StringNull()
	}

	if g.Category != nil {
		state.CategoryID = helpers.StringValueFromIntPtr(g.Category.ID)
		state.CategoryName = helpers.DerivedRefName(g.Category.ID, g.Category.Name)
	}
	if g.Site != nil {
		state.SiteID = helpers.StringValueFromIntPtr(g.Site.ID)
		state.SiteName = helpers.DerivedRefName(g.Site.ID, g.Site.Name)
	}
}

// flattenScope refreshes the scope sub-blocks the caller already manages.
// When includeUnmanaged is set (config generation) every wire-present
// sub-block is first allocated so the from-scratch read hydrates the full
// scope rather than leaving unmanaged targets/limitations/exclusions null.
func flattenScope(ctx context.Context, s *proclassic.MobileDeviceConfigurationProfileScope, state *scope.MobileScopeModel, includeUnmanaged bool) diag.Diagnostics {
	var diags diag.Diagnostics
	if s == nil {
		return diags
	}

	if includeUnmanaged {
		if state.Targets == nil {
			state.Targets = &scope.MobileScopeTargetsModel{}
		}
		if state.Limitations == nil && s.Limitations != nil {
			state.Limitations = &scope.MobileScopeLimitationsModel{}
		}
		if state.Exclusions == nil && s.Exclusions != nil {
			state.Exclusions = &scope.MobileScopeExclusionsModel{}
		}
	}

	// Targets are gated on caller management, mirroring the limitations /
	// exclusions sub-blocks below: populating a targets block the user did not
	// declare would violate the framework's "produced inconsistent result after
	// apply" check (plan said null, we would return a populated object).
	if state.Targets != nil {
		state.Targets.AllMobileDevices = helpers.ReconcileOrAdoptBoolPointer(s.AllMobileDevices, state.Targets.AllMobileDevices, includeUnmanaged)
		state.Targets.AllJssUsers = helpers.ReconcileOrAdoptBoolPointer(s.AllJssUsers, state.Targets.AllJssUsers, includeUnmanaged)

		if s.MobileDevices != nil {
			v, d := scope.FlattenIDSlice(ctx, s.MobileDevices.MobileDevice, func(c proclassic.MobileDeviceConfigurationProfileScopeMobileDevicesMobileDeviceItem) *int {
				return c.ID
			})
			diags.Append(d...)
			state.Targets.MobileDeviceIDs = v
		} else {
			state.Targets.MobileDeviceIDs = scope.EmptyStringSet()
		}
		if s.MobileDeviceGroups != nil {
			v, d := scope.FlattenIDSlice(ctx, s.MobileDeviceGroups.MobileDeviceGroup, func(c proclassic.IDName) *int { return c.ID })
			diags.Append(d...)
			state.Targets.MobileDeviceGroupIDs = v
		} else {
			state.Targets.MobileDeviceGroupIDs = scope.EmptyStringSet()
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
			v, d := scope.FlattenIDSlice(ctx, s.JssUserGroups.UserGroup, func(c proclassic.IDName) *int { return c.ID })
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

func flattenScopeLimitations(ctx context.Context, l *proclassic.MobileDeviceConfigurationProfileScopeLimitations, state *scope.MobileScopeLimitationsModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if l == nil {
		return diags
	}
	if l.NetworkSegments != nil {
		v, d := scope.FlattenIDSlice(ctx, l.NetworkSegments.NetworkSegment, func(c proclassic.IDName) *int { return c.ID })
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

func flattenScopeExclusions(ctx context.Context, e *proclassic.MobileDeviceConfigurationProfileScopeExclusions, state *scope.MobileScopeExclusionsModel) diag.Diagnostics {
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
		state.MobileDeviceIDs = scope.EmptyStringSet()
	}
	if e.MobileDeviceGroups != nil {
		v, d := scope.FlattenIDSlice(ctx, e.MobileDeviceGroups.MobileDeviceGroup, func(c proclassic.IDName) *int { return c.ID })
		diags.Append(d...)
		state.MobileDeviceGroupIDs = v
	} else {
		state.MobileDeviceGroupIDs = scope.EmptyStringSet()
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
		v, d := scope.FlattenIDSlice(ctx, e.NetworkSegments.NetworkSegment, func(c proclassic.MobileDeviceConfigurationProfileScopeExclusionsNetworkSegmentsNetworkSegmentItem) *int {
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
		v, d := scope.FlattenNameSlice(ctx, e.Users.User, func(c proclassic.MobileDeviceConfigurationProfileScopeExclusionsUsersUserItem) *string {
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
