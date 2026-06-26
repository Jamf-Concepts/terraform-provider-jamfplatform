// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ebook

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

// assignEbookResourceModel populates a resource model from the SDK Ebook
// response. general is always refreshed (required block). The optional sections
// (scope / self_service) are only refreshed when the caller (plan or current
// state) already manages them: the classic server echoes every section on GET
// with default values, so populating an unmanaged section would violate the
// framework's "produced inconsistent result after apply" check (plan said null,
// we'd return a populated object). See feedback_server_derived_echo_attrs.
//
// includeUnmanaged inverts those section gates for the list resource's
// config-generation path (terraform query -generate-config-out): there is no
// plan to stay consistent with, so every wire-present optional section is
// allocated and hydrated, yielding a complete exported config rather than a
// general-only one. CRUD callers pass false. The ebook flatteners use the
// PreferCurrent* helpers (which adopt the wire value when the current state is
// null), so allocating an empty section is sufficient for it to fully hydrate.
func assignEbookResourceModel(ctx context.Context, state *EbookResourceModel, e *proclassic.Ebook, includeUnmanaged bool) diag.Diagnostics {
	var diags diag.Diagnostics
	if e == nil {
		return diags
	}

	if id := extractEbookID(e); id != "" {
		state.ID = types.StringValue(id)
	}

	if state.General == nil {
		state.General = &EbookGeneralModel{}
	}
	flattenEbookGeneral(e.General, state.General)

	if includeUnmanaged && state.Scope == nil && e.Scope != nil {
		state.Scope = &EbookScopeModel{}
	}
	if state.Scope != nil && e.Scope != nil {
		flattenEbookScope(ctx, e.Scope, state.Scope, includeUnmanaged)
	}
	if includeUnmanaged && state.SelfService == nil && e.SelfService != nil {
		state.SelfService = &EbookSelfServiceModel{}
	}
	if state.SelfService != nil && e.SelfService != nil {
		flattenEbookSelfService(e.SelfService, state.SelfService)
	}

	return diags
}

func flattenEbookGeneral(g *proclassic.EbookGeneral, state *EbookGeneralModel) {
	if g == nil {
		return
	}
	state.ID = helpers.StringValueFromIntPtr(g.ID)
	state.Name = helpers.StringPointerValueOrNull(g.Name)
	state.URL = helpers.StringPointerValueOrNull(g.URL)
	state.Author = helpers.PreferCurrentStringPointer(g.Author, state.Author)
	state.DeploymentType = helpers.PreferCurrentStringPointer(g.DeploymentType, state.DeploymentType)
	state.DeployAsManaged = helpers.PreferCurrentBoolPointer(g.DeployAsManaged, state.DeployAsManaged)
	state.Free = helpers.PreferCurrentBoolPointer(g.Free, state.Free)
	state.FileType = helpers.PreferCurrentStringPointer(g.FileType, state.FileType)
	state.Version = helpers.PreferCurrentStringPointer(g.Version, state.Version)

	if g.Category != nil {
		state.CategoryID = helpers.PreferCurrentStringPointer(helpers.StringFromIntPtr(g.Category.ID), state.CategoryID)
		state.CategoryName = helpers.DerivedRefName(g.Category.ID, g.Category.Name)
	} else {
		state.CategoryID = helpers.PreferCurrentStringPointer(nil, state.CategoryID)
		state.CategoryName = types.StringNull()
	}

	if g.Site != nil {
		state.SiteID = helpers.PreferCurrentStringPointer(helpers.StringFromIntPtr(g.Site.ID), state.SiteID)
		state.SiteName = helpers.DerivedRefName(g.Site.ID, g.Site.Name)
	} else {
		state.SiteID = helpers.PreferCurrentStringPointer(nil, state.SiteID)
		state.SiteName = types.StringNull()
	}
}

// flattenEbookScope refreshes the scope sub-blocks the caller already manages.
// When includeUnmanaged is set (config generation) every wire-present sub-block
// is first allocated so the from-scratch read hydrates the full scope rather
// than leaving unmanaged targets/limitations/exclusions null.
func flattenEbookScope(ctx context.Context, s *proclassic.EbookScope, state *EbookScopeModel, includeUnmanaged bool) {
	if includeUnmanaged {
		if state.Targets == nil {
			state.Targets = &EbookScopeTargetsModel{}
		}
		if state.Limitations == nil && s.Limitations != nil {
			state.Limitations = &EbookScopeLimitationsModel{}
		}
		if state.Exclusions == nil && s.Exclusions != nil {
			state.Exclusions = &EbookScopeExclusionsModel{}
		}
	}

	if state.Targets != nil {
		t := state.Targets
		t.AllComputers = helpers.PreferCurrentBoolPointer(s.AllComputers, t.AllComputers)
		t.AllMobileDevices = helpers.PreferCurrentBoolPointer(s.AllMobileDevices, t.AllMobileDevices)
		t.AllJssUsers = helpers.PreferCurrentBoolPointer(s.AllJssUsers, t.AllJssUsers)

		t.ComputerIDs = flattenComputerItemSet(ctx, s.Computers)
		t.ComputerGroupIDs = scope.FlattenIDNameSet(ctx, computerGroupSlice(s.ComputerGroups))
		t.MobileDeviceIDs = flattenMobileDeviceItemSet(ctx, s.MobileDevices)
		t.MobileDeviceGroupIDs = scope.FlattenIDNameSet(ctx, mobileDeviceGroupSlice(s.MobileDeviceGroups))
		t.BuildingIDs = scope.FlattenIDNameSet(ctx, buildingSlice(s.Buildings))
		t.DepartmentIDs = scope.FlattenIDNameSet(ctx, departmentSlice(s.Departments))
		t.UserIDs = scope.FlattenIDNameSet(ctx, jssUserSlice(s.JssUsers))
		t.UserGroupIDs = scope.FlattenIDNameSet(ctx, jssUserGroupSlice(s.JssUserGroups))
		t.ClassIDs = scope.FlattenIDNameSet(ctx, classSlice(s.Classes))
	}

	if state.Limitations != nil && s.Limitations != nil {
		l := s.Limitations
		state.Limitations.NetworkSegmentIDs = scope.FlattenIDNameSet(ctx, limitationsSegmentSlice(l.NetworkSegments))
		state.Limitations.DirectoryServiceOrLocalUserNames = scope.FlattenNameSet(ctx, limitationsUserSlice(l.Users))
		state.Limitations.DirectoryServiceUserGroupNames = scope.FlattenNameSet(ctx, limitationsUserGroupSlice(l.UserGroups))
	}

	if state.Exclusions != nil && s.Exclusions != nil {
		e := s.Exclusions
		state.Exclusions.ComputerIDs = flattenExclComputerItemSet(ctx, e.Computers)
		state.Exclusions.ComputerGroupIDs = scope.FlattenIDNameSet(ctx, exclComputerGroupSlice(e.ComputerGroups))
		state.Exclusions.MobileDeviceIDs = flattenExclMobileDeviceItemSet(ctx, e.MobileDevices)
		state.Exclusions.MobileDeviceGroupIDs = scope.FlattenIDNameSet(ctx, exclMobileDeviceGroupSlice(e.MobileDeviceGroups))
		state.Exclusions.BuildingIDs = scope.FlattenIDNameSet(ctx, exclBuildingSlice(e.Buildings))
		state.Exclusions.DepartmentIDs = scope.FlattenIDNameSet(ctx, exclDepartmentSlice(e.Departments))
		state.Exclusions.UserIDs = scope.FlattenIDNameSet(ctx, exclJssUserSlice(e.JssUsers))
		state.Exclusions.UserGroupIDs = scope.FlattenIDNameSet(ctx, exclJssUserGroupSlice(e.JssUserGroups))
		state.Exclusions.NetworkSegmentIDs = flattenExclNetworkSegmentSet(ctx, e.NetworkSegments)
		state.Exclusions.DirectoryServiceOrLocalUserNames = flattenExclUsersNameSet(ctx, e.Users)
		state.Exclusions.DirectoryServiceUserGroupNames = scope.FlattenNameSet(ctx, exclUserGroupSlice(e.UserGroups))
	}
}

func flattenEbookSelfService(ss *proclassic.EbookSelfService, state *EbookSelfServiceModel) {
	state.DisplayName = helpers.PreferCurrentStringPointer(ss.SelfServiceDisplayName, state.DisplayName)
	state.InstallButtonText = helpers.PreferCurrentStringPointer(ss.InstallButtonText, state.InstallButtonText)
	state.SelfServiceDescription = helpers.PreferCurrentStringPointer(ss.SelfServiceDescription, state.SelfServiceDescription)
	state.ForceUsersToViewDescription = helpers.PreferCurrentBoolPointer(ss.ForceUsersToViewDescription, state.ForceUsersToViewDescription)
	state.FeatureOnMainPage = helpers.PreferCurrentBoolPointer(ss.FeatureOnMainPage, state.FeatureOnMainPage)

	var apiEnabled *bool
	var apiMethod *string
	if ss.Notification != nil {
		apiEnabled = ss.Notification.Enabled
		apiMethod = ss.Notification.Method
	}
	state.NotificationEnabled = helpers.PreferCurrentBoolPointer(apiEnabled, state.NotificationEnabled)
	state.NotificationMethod = helpers.PreferCurrentStringPointer(apiMethod, state.NotificationMethod)
	state.NotificationSubject = helpers.PreferCurrentStringPointer(ss.NotificationSubject, state.NotificationSubject)
	state.NotificationMessage = helpers.PreferCurrentStringPointer(ss.NotificationMessage, state.NotificationMessage)

	if ss.SelfServiceIcon != nil {
		state.IconID = helpers.PreferCurrentStringPointer(helpers.StringFromIntPtr(ss.SelfServiceIcon.ID), state.IconID)
		state.IconURI = helpers.StringPointerValueOrNull(ss.SelfServiceIcon.URI)
	} else {
		state.IconID = helpers.PreferCurrentStringPointer(nil, state.IconID)
		state.IconURI = types.StringNull()
	}

	if state.Categories != nil && ss.SelfServiceCategories != nil && ss.SelfServiceCategories.Category != nil {
		flattenEbookSelfServiceCategories(*ss.SelfServiceCategories.Category, state)
	}
}

// flattenEbookSelfServiceCategories refreshes the managed category set, matching
// server items to existing state items by ID so caller-authored values
// (display_in / feature_in) stick across refreshes. The set is keyed by category ID.
func flattenEbookSelfServiceCategories(api []proclassic.EbookSelfServiceSelfServiceCategoriesCategoryItem, state *EbookSelfServiceModel) {
	byID := make(map[string]EbookSelfServiceCategoryModel, len(state.Categories))
	for _, c := range state.Categories {
		byID[c.ID.ValueString()] = c
	}

	out := make([]EbookSelfServiceCategoryModel, 0, len(api))
	for _, c := range api {
		idStr := ""
		if s := helpers.StringFromIntPtr(c.ID); s != nil {
			idStr = *s
		}
		current := byID[idStr]
		out = append(out, EbookSelfServiceCategoryModel{
			ID:        types.StringValue(idStr),
			Name:      helpers.PreferCurrentStringPointer(c.Name, current.Name),
			DisplayIn: helpers.PreferCurrentBoolPointer(c.DisplayIn, current.DisplayIn),
			FeatureIn: helpers.PreferCurrentBoolPointer(c.FeatureIn, current.FeatureIn),
		})
	}
	state.Categories = out
}

// ---- scope sub-slice accessors -------------------------------------------------

func computerGroupSlice(g *proclassic.EbookScopeComputerGroups) *[]proclassic.IDName {
	if g == nil {
		return nil
	}
	return g.ComputerGroup
}

func mobileDeviceGroupSlice(g *proclassic.EbookScopeMobileDeviceGroups) *[]proclassic.IDName {
	if g == nil {
		return nil
	}
	return g.MobileDeviceGroup
}

func buildingSlice(b *proclassic.EbookScopeBuildings) *[]proclassic.IDName {
	if b == nil {
		return nil
	}
	return b.Building
}

func departmentSlice(d *proclassic.EbookScopeDepartments) *[]proclassic.IDName {
	if d == nil {
		return nil
	}
	return d.Department
}

func jssUserSlice(u *proclassic.EbookScopeJssUsers) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.User
}

func jssUserGroupSlice(u *proclassic.EbookScopeJssUserGroups) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.UserGroup
}

func classSlice(c *proclassic.EbookScopeClasses) *[]proclassic.IDName {
	if c == nil {
		return nil
	}
	return c.Class
}

func limitationsSegmentSlice(s *proclassic.EbookScopeLimitationsNetworkSegments) *[]proclassic.IDName {
	if s == nil {
		return nil
	}
	return s.NetworkSegment
}

func limitationsUserSlice(u *proclassic.EbookScopeLimitationsUsers) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.User
}

func limitationsUserGroupSlice(u *proclassic.EbookScopeLimitationsUserGroups) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.UserGroup
}

func exclComputerGroupSlice(g *proclassic.EbookScopeExclusionsComputerGroups) *[]proclassic.IDName {
	if g == nil {
		return nil
	}
	return g.ComputerGroup
}

func exclMobileDeviceGroupSlice(g *proclassic.EbookScopeExclusionsMobileDeviceGroups) *[]proclassic.IDName {
	if g == nil {
		return nil
	}
	return g.MobileDeviceGroup
}

func exclBuildingSlice(b *proclassic.EbookScopeExclusionsBuildings) *[]proclassic.IDName {
	if b == nil {
		return nil
	}
	return b.Building
}

func exclDepartmentSlice(d *proclassic.EbookScopeExclusionsDepartments) *[]proclassic.IDName {
	if d == nil {
		return nil
	}
	return d.Department
}

func exclJssUserSlice(u *proclassic.EbookScopeExclusionsJssUsers) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.User
}

func exclJssUserGroupSlice(u *proclassic.EbookScopeExclusionsJssUserGroups) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.UserGroup
}

func exclUserGroupSlice(u *proclassic.EbookScopeExclusionsUserGroups) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.UserGroup
}

// ---- set flatteners ------------------------------------------------------------

func flattenComputerItemSet(ctx context.Context, c *proclassic.EbookScopeComputers) types.Set {
	if c == nil {
		return types.SetNull(types.StringType)
	}
	out, _ := scope.FlattenIDSlice(ctx, c.Computer, func(i proclassic.EbookScopeComputersComputerItem) *int { return i.ID })
	return out
}

func flattenMobileDeviceItemSet(ctx context.Context, m *proclassic.EbookScopeMobileDevices) types.Set {
	if m == nil {
		return types.SetNull(types.StringType)
	}
	out, _ := scope.FlattenIDSlice(ctx, m.MobileDevice, func(i proclassic.EbookScopeMobileDevicesMobileDeviceItem) *int { return i.ID })
	return out
}

func flattenExclComputerItemSet(ctx context.Context, c *proclassic.EbookScopeExclusionsComputers) types.Set {
	if c == nil {
		return types.SetNull(types.StringType)
	}
	out, _ := scope.FlattenIDSlice(ctx, c.Computer, func(i proclassic.EbookScopeExclusionsComputersComputerItem) *int { return i.ID })
	return out
}

func flattenExclMobileDeviceItemSet(ctx context.Context, m *proclassic.EbookScopeExclusionsMobileDevices) types.Set {
	if m == nil {
		return types.SetNull(types.StringType)
	}
	out, _ := scope.FlattenIDSlice(ctx, m.MobileDevice, func(i proclassic.EbookScopeExclusionsMobileDevicesMobileDeviceItem) *int { return i.ID })
	return out
}

func flattenExclNetworkSegmentSet(ctx context.Context, n *proclassic.EbookScopeExclusionsNetworkSegments) types.Set {
	if n == nil {
		return types.SetNull(types.StringType)
	}
	out, _ := scope.FlattenIDSlice(ctx, n.NetworkSegment, func(i proclassic.EbookScopeExclusionsNetworkSegmentsNetworkSegmentItem) *int { return i.ID })
	return out
}

func flattenExclUsersNameSet(ctx context.Context, u *proclassic.EbookScopeExclusionsUsers) types.Set {
	if u == nil {
		return types.SetNull(types.StringType)
	}
	out, _ := scope.FlattenNameSlice(ctx, u.User, func(i proclassic.EbookScopeExclusionsUsersUserItem) *string { return i.Name })
	return out
}
