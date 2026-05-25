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

func buildInput(ctx context.Context, plan ResourceModel, existingUUID string) (*proclassic.MobileDeviceConfigurationProfile, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := &proclassic.MobileDeviceConfigurationProfile{}

	if plan.General != nil {
		general, _, d := buildGeneral(plan.General, existingUUID)
		diags.Append(d...)
		out.General = general
	}
	if plan.Scope != nil {
		s, d := buildScope(ctx, plan.Scope)
		diags.Append(d...)
		out.Scope = s
	}
	if plan.SelfService != nil {
		ss, d := buildSelfService(plan.SelfService)
		diags.Append(d...)
		out.SelfService = ss
	}
	return out, diags
}

func buildGeneral(m *GeneralModel, existingUUID string) (*proclassic.MobileDeviceConfigurationProfileGeneral, []byte, diag.Diagnostics) {
	var diags diag.Diagnostics
	g := &proclassic.MobileDeviceConfigurationProfileGeneral{
		Name:             helpers.OptionalStringPointer(m.Name),
		Description:      helpers.OptionalStringPointer(m.Description),
		RedeployOnUpdate: helpers.OptionalStringPointer(m.RedeployOnUpdate),
	}

	if !m.RedeployDaysBeforeCertificateExpires.IsNull() && !m.RedeployDaysBeforeCertificateExpires.IsUnknown() {
		g.RedeployDaysBeforeCertificateExpires = helpers.OptionalInt64Pointer(m.RedeployDaysBeforeCertificateExpires)
	}

	if v := m.Level.ValueString(); !m.Level.IsNull() && !m.Level.IsUnknown() {
		wire := levelToWireWrite(v)
		g.Level = &wire
	}
	if v := m.DistributionMethod.ValueString(); !m.DistributionMethod.IsNull() && !m.DistributionMethod.IsUnknown() {
		dm := v
		g.DeploymentMethod = &dm
	}

	if id := optionalIntFromStringID(m.CategoryID); id != nil {
		g.Category = &proclassic.CategoryObject{ID: id}
	}
	if id := optionalIntFromStringID(m.SiteID); id != nil {
		g.Site = &proclassic.SiteObject{ID: id}
	}

	if v := m.Payloads.ValueString(); !m.Payloads.IsNull() && !m.Payloads.IsUnknown() && v != "" {
		raw := []byte(v)
		prepared, err := payloadhelpers.InjectTopLevelIdentifierValues(raw, existingUUID, existingUUID)
		if err != nil {
			diags.AddError("Failed to inject server-canonical PayloadUUID/PayloadIdentifier into update payload", err.Error())
			return nil, nil, diags
		}
		s := string(prepared)
		g.Payloads = &s
		return g, prepared, diags
	}
	return g, nil, diags
}

func buildScope(ctx context.Context, m *ScopeModel) (*proclassic.MobileDeviceConfigurationProfileScope, diag.Diagnostics) {
	var diags diag.Diagnostics
	s := &proclassic.MobileDeviceConfigurationProfileScope{
		AllMobileDevices: optionalBoolPointer(m.AllMobileDevices),
		AllJssUsers:      optionalBoolPointer(m.AllJssUsers),
	}

	mds, d := scope.BuildIDSlice(ctx, m.MobileDeviceIDs, func(id int) proclassic.MobileDeviceConfigurationProfileScopeMobileDevicesMobileDeviceItem {
		return proclassic.MobileDeviceConfigurationProfileScopeMobileDevicesMobileDeviceItem{ID: &id}
	})
	diags.Append(d...)
	if mds != nil {
		s.MobileDevices = &proclassic.MobileDeviceConfigurationProfileScopeMobileDevices{MobileDevice: mds}
	}

	mdgs, d := scope.BuildIDSlice(ctx, m.MobileDeviceGroupIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if mdgs != nil {
		s.MobileDeviceGroups = &proclassic.MobileDeviceConfigurationProfileScopeMobileDeviceGroups{MobileDeviceGroup: mdgs}
	}

	buildings, d := scope.BuildIDSlice(ctx, m.BuildingIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if buildings != nil {
		s.Buildings = &proclassic.MobileDeviceConfigurationProfileScopeBuildings{Building: buildings}
	}

	departments, d := scope.BuildIDSlice(ctx, m.DepartmentIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if departments != nil {
		s.Departments = &proclassic.MobileDeviceConfigurationProfileScopeDepartments{Department: departments}
	}

	jssUsers, d := scope.BuildIDSlice(ctx, m.UserIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if jssUsers != nil {
		s.JssUsers = &proclassic.MobileDeviceConfigurationProfileScopeJssUsers{User: jssUsers}
	}

	jssUserGroups, d := scope.BuildIDSlice(ctx, m.UserGroupIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if jssUserGroups != nil {
		s.JssUserGroups = &proclassic.MobileDeviceConfigurationProfileScopeJssUserGroups{UserGroup: jssUserGroups}
	}

	if m.Limitations != nil {
		l, ld := buildScopeLimitations(ctx, m.Limitations)
		diags.Append(ld...)
		s.Limitations = l
	}
	if m.Exclusions != nil {
		e, ed := buildScopeExclusions(ctx, m.Exclusions)
		diags.Append(ed...)
		s.Exclusions = e
	}

	if s.AllMobileDevices == nil && s.AllJssUsers == nil && s.MobileDevices == nil &&
		s.MobileDeviceGroups == nil && s.Buildings == nil && s.Departments == nil &&
		s.JssUsers == nil && s.JssUserGroups == nil &&
		s.Limitations == nil && s.Exclusions == nil {
		return nil, diags
	}
	return s, diags
}

func buildScopeLimitations(ctx context.Context, m *ScopeLimitationsModel) (*proclassic.MobileDeviceConfigurationProfileScopeLimitations, diag.Diagnostics) {
	var diags diag.Diagnostics
	l := &proclassic.MobileDeviceConfigurationProfileScopeLimitations{}

	segs, d := scope.BuildIDSlice(ctx, m.NetworkSegmentIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if segs != nil {
		l.NetworkSegments = &proclassic.MobileDeviceConfigurationProfileScopeLimitationsNetworkSegments{NetworkSegment: segs}
	}

	ibeacons, d := scope.BuildIDSlice(ctx, m.IbeaconIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if ibeacons != nil {
		l.Ibeacons = &proclassic.MobileDeviceConfigurationProfileScopeLimitationsIbeacons{Ibeacon: ibeacons}
	}

	users, d := scope.BuildNameSlice(ctx, m.DirectoryServiceOrLocalUserNames, func(name string) proclassic.IDName {
		n := name
		return proclassic.IDName{Name: &n}
	})
	diags.Append(d...)
	if users != nil {
		l.Users = &proclassic.MobileDeviceConfigurationProfileScopeLimitationsUsers{User: users}
	}

	userGroups, d := scope.BuildNameSlice(ctx, m.DirectoryServiceUserGroupNames, func(name string) proclassic.IDName {
		n := name
		return proclassic.IDName{Name: &n}
	})
	diags.Append(d...)
	if userGroups != nil {
		l.UserGroups = &proclassic.MobileDeviceConfigurationProfileScopeLimitationsUserGroups{UserGroup: userGroups}
	}

	if l.NetworkSegments == nil && l.Ibeacons == nil && l.Users == nil && l.UserGroups == nil {
		return nil, diags
	}
	return l, diags
}

func buildScopeExclusions(ctx context.Context, m *ScopeExclusionsModel) (*proclassic.MobileDeviceConfigurationProfileScopeExclusions, diag.Diagnostics) {
	var diags diag.Diagnostics
	e := &proclassic.MobileDeviceConfigurationProfileScopeExclusions{}

	mds, d := scope.BuildIDSlice(ctx, m.MobileDeviceIDs, func(id int) proclassic.MobileDeviceConfigurationProfileScopeExclusionsMobileDevicesMobileDeviceItem {
		return proclassic.MobileDeviceConfigurationProfileScopeExclusionsMobileDevicesMobileDeviceItem{ID: &id}
	})
	diags.Append(d...)
	if mds != nil {
		e.MobileDevices = &proclassic.MobileDeviceConfigurationProfileScopeExclusionsMobileDevices{MobileDevice: mds}
	}

	mdgs, d := scope.BuildIDSlice(ctx, m.MobileDeviceGroupIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if mdgs != nil {
		e.MobileDeviceGroups = &proclassic.MobileDeviceConfigurationProfileScopeExclusionsMobileDeviceGroups{MobileDeviceGroup: mdgs}
	}

	buildings, d := scope.BuildIDSlice(ctx, m.BuildingIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if buildings != nil {
		e.Buildings = &proclassic.MobileDeviceConfigurationProfileScopeExclusionsBuildings{Building: buildings}
	}

	departments, d := scope.BuildIDSlice(ctx, m.DepartmentIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if departments != nil {
		e.Departments = &proclassic.MobileDeviceConfigurationProfileScopeExclusionsDepartments{Department: departments}
	}

	jssUsers, d := scope.BuildIDSlice(ctx, m.UserIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if jssUsers != nil {
		e.JssUsers = &proclassic.MobileDeviceConfigurationProfileScopeExclusionsJssUsers{User: jssUsers}
	}

	jssUserGroups, d := scope.BuildIDSlice(ctx, m.UserGroupIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if jssUserGroups != nil {
		e.JssUserGroups = &proclassic.MobileDeviceConfigurationProfileScopeExclusionsJssUserGroups{UserGroup: jssUserGroups}
	}

	segs, d := scope.BuildIDSlice(ctx, m.NetworkSegmentIDs, func(id int) proclassic.MobileDeviceConfigurationProfileScopeExclusionsNetworkSegmentsNetworkSegmentItem {
		return proclassic.MobileDeviceConfigurationProfileScopeExclusionsNetworkSegmentsNetworkSegmentItem{ID: &id}
	})
	diags.Append(d...)
	if segs != nil {
		e.NetworkSegments = &proclassic.MobileDeviceConfigurationProfileScopeExclusionsNetworkSegments{NetworkSegment: segs}
	}

	ibeacons, d := scope.BuildIDSlice(ctx, m.IbeaconIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if ibeacons != nil {
		e.Ibeacons = &proclassic.MobileDeviceConfigurationProfileScopeExclusionsIbeacons{Ibeacon: ibeacons}
	}

	users, d := scope.BuildNameSlice(ctx, m.DirectoryServiceOrLocalUserNames, func(name string) proclassic.MobileDeviceConfigurationProfileScopeExclusionsUsersUserItem {
		n := name
		return proclassic.MobileDeviceConfigurationProfileScopeExclusionsUsersUserItem{Name: &n}
	})
	diags.Append(d...)
	if users != nil {
		e.Users = &proclassic.MobileDeviceConfigurationProfileScopeExclusionsUsers{User: users}
	}

	userGroups, d := scope.BuildNameSlice(ctx, m.DirectoryServiceUserGroupNames, func(name string) proclassic.IDName {
		n := name
		return proclassic.IDName{Name: &n}
	})
	diags.Append(d...)
	if userGroups != nil {
		e.UserGroups = &proclassic.MobileDeviceConfigurationProfileScopeExclusionsUserGroups{UserGroup: userGroups}
	}

	if e.MobileDevices == nil && e.MobileDeviceGroups == nil && e.Buildings == nil &&
		e.Departments == nil && e.JssUsers == nil && e.JssUserGroups == nil &&
		e.NetworkSegments == nil && e.Ibeacons == nil &&
		e.Users == nil && e.UserGroups == nil {
		return nil, diags
	}
	return e, diags
}

func buildSelfService(m *SelfServiceModel) (*proclassic.MobileDeviceConfigurationProfileSelfService, diag.Diagnostics) {
	var diags diag.Diagnostics
	ss := &proclassic.MobileDeviceConfigurationProfileSelfService{
		SelfServiceDescription: helpers.OptionalStringPointer(m.SelfServiceDescription),
		FeatureOnMainPage:      optionalBoolPointer(m.FeatureOnMainPage),
	}

	hasDisallowed := !m.RemovalDisallowed.IsNull() && !m.RemovalDisallowed.IsUnknown() && m.RemovalDisallowed.ValueString() != ""
	hasPassword := !m.AuthorizationPassword.IsNull() && !m.AuthorizationPassword.IsUnknown() && m.AuthorizationPassword.ValueString() != ""
	if hasDisallowed || hasPassword {
		sec := &proclassic.MobileDeviceConfigurationProfileSelfServiceSecurity{}
		if hasDisallowed {
			v := m.RemovalDisallowed.ValueString()
			sec.RemovalDisallowed = &v
		}
		if hasPassword {
			v := m.AuthorizationPassword.ValueString()
			sec.Password = &v
		}
		ss.Security = sec
	}

	if len(m.Categories) > 0 {
		items := make([]proclassic.Category, 0, len(m.Categories))
		for _, c := range m.Categories {
			item := proclassic.Category{}
			if id := optionalIntFromStringID(c.ID); id != nil {
				item.ID = id
			}
			items = append(items, item)
		}
		ss.SelfServiceCategories = &proclassic.MobileDeviceConfigurationProfileSelfServiceSelfServiceCategories{
			Category: &items,
		}
	}

	return ss, diags
}

func optionalBoolPointer(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := value.ValueBool()
	return &v
}

func optionalIntFromStringID(value types.String) *int {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	s := value.ValueString()
	if s == "" {
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &n
}
