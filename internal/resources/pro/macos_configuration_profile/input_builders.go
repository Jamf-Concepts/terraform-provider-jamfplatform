// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package macos_configuration_profile

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/payloadhelpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

// buildInput projects a plan ResourceModel into a *proclassic.OsXConfigurationProfile
// suitable for POST / PUT.
//
// On Create, existingUUID is empty and the user's payload goes through as-is;
// the server assigns a fresh top-level PayloadUUID + PayloadIdentifier.
//
// On Update, existingUUID carries the server-canonical UUID sourced from
// state.General.UUID. The input builder substitutes that value into both
// the top-level PayloadUUID and PayloadIdentifier of the new payload before
// PUT, so the server preserves the profile's identity across updates.
// Without this step, the server would assign a new UUID and macOS devices
// would treat the update as a fresh install ("ghost profile").
//
// The Classic API uses identical lowercase UUIDs for the two top-level
// fields on Jamf-minted profiles, which is why the same string drives both.
func buildInput(ctx context.Context, plan ResourceModel, existingUUID string) (*proclassic.OsXConfigurationProfile, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := &proclassic.OsXConfigurationProfile{}

	if plan.General != nil {
		general, payloadBytes, d := buildGeneral(plan.General, existingUUID)
		diags.Append(d...)
		out.General = general
		_ = payloadBytes
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

// buildGeneral projects the GeneralModel into a *proclassic.OsXConfigurationProfileGeneral.
// `existingUUID` is the server-canonical UUID from state, used for top-level
// identifier injection on Update. Empty on Create.
//
// Returns the prepared payload bytes alongside the General struct so tests
// can inspect what would have been sent on the wire.
func buildGeneral(m *GeneralModel, existingUUID string) (*proclassic.OsXConfigurationProfileGeneral, []byte, diag.Diagnostics) {
	var diags diag.Diagnostics
	g := &proclassic.OsXConfigurationProfileGeneral{
		Name:             helpers.OptionalStringPointer(m.Name),
		Description:      helpers.OptionalStringPointer(m.Description),
		UserRemovable:    helpers.OptionalBoolPointer(m.UserRemovable),
		RedeployOnUpdate: helpers.OptionalStringPointer(m.RedeployOnUpdate),
	}

	if v := m.Level.ValueString(); !m.Level.IsNull() && !m.Level.IsUnknown() {
		wire := levelToWireWrite(v)
		g.Level = &wire
	}
	if v := m.DistributionMethod.ValueString(); !m.DistributionMethod.IsNull() && !m.DistributionMethod.IsUnknown() {
		dm := v
		g.DistributionMethod = &dm
	}

	if id := helpers.StringIDPtr(m.CategoryID); id != nil {
		g.Category = &proclassic.CategoryObject{ID: id}
	}
	if id := helpers.StringIDPtr(m.SiteID); id != nil {
		g.Site = &proclassic.SiteObject{ID: id}
	}

	// Payload preparation:
	// - On Update, substitute the server-canonical UUID into both the
	//   top-level PayloadUUID and PayloadIdentifier of the new payload.
	//   The Classic API uses the same lowercase UUID for both fields on
	//   Jamf-minted profiles (verified across the 200-profile roundtrip
	//   corpus).
	// - On Create, existingUUID is empty; the identifier injection is a no-op.
	// - Either way, structural whitespace is compacted before send (Classic
	//   API plist-parser phantom-<array/> bug — see PrepareWirePayload).
	if v := m.Payloads.ValueString(); !m.Payloads.IsNull() && !m.Payloads.IsUnknown() && v != "" {
		raw := []byte(v)
		prepared, err := payloadhelpers.PrepareWirePayload(raw, existingUUID, existingUUID)
		if err != nil {
			diags.AddError("Failed to inject server-canonical PayloadUUID/PayloadIdentifier into update payload", err.Error())
			return nil, nil, diags
		}
		s := proclassic.PayloadsXMLText(prepared)
		g.Payloads = &s
		return g, prepared, diags
	}
	return g, nil, diags
}

func buildScope(ctx context.Context, m *scope.ComputerScopeModel) (*proclassic.OsXConfigurationProfileScope, diag.Diagnostics) {
	var diags diag.Diagnostics
	t := m.TargetsOrZero()
	s := &proclassic.OsXConfigurationProfileScope{
		AllComputers: helpers.OptionalBoolPointer(t.AllComputers),
		AllJssUsers:  helpers.OptionalBoolPointer(t.AllJssUsers),
	}

	computers, d := scope.BuildIDSlice(ctx, t.ComputerIDs, func(id int) proclassic.OsXConfigurationProfileScopeComputersComputerItem {
		return proclassic.OsXConfigurationProfileScopeComputersComputerItem{ID: &id}
	})
	diags.Append(d...)
	if computers != nil {
		s.Computers = &proclassic.OsXConfigurationProfileScopeComputers{Computer: computers}
	}

	cgs, d := scope.BuildIDSlice(ctx, t.ComputerGroupIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if cgs != nil {
		s.ComputerGroups = &proclassic.OsXConfigurationProfileScopeComputerGroups{ComputerGroup: cgs}
	}

	buildings, d := scope.BuildIDSlice(ctx, t.BuildingIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if buildings != nil {
		s.Buildings = &proclassic.OsXConfigurationProfileScopeBuildings{Building: buildings}
	}

	departments, d := scope.BuildIDSlice(ctx, t.DepartmentIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if departments != nil {
		s.Departments = &proclassic.OsXConfigurationProfileScopeDepartments{Department: departments}
	}

	jssUsers, d := scope.BuildIDSlice(ctx, t.UserIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if jssUsers != nil {
		s.JssUsers = &proclassic.OsXConfigurationProfileScopeJssUsers{User: jssUsers}
	}

	jssUserGroups, d := scope.BuildIDSlice(ctx, t.UserGroupIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if jssUserGroups != nil {
		s.JssUserGroups = &proclassic.OsXConfigurationProfileScopeJssUserGroups{JssUserGroup: jssUserGroups}
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

	if s.AllComputers == nil && s.AllJssUsers == nil && s.Computers == nil &&
		s.ComputerGroups == nil && s.Buildings == nil && s.Departments == nil &&
		s.JssUsers == nil && s.JssUserGroups == nil &&
		s.Limitations == nil && s.Exclusions == nil {
		return nil, diags
	}
	return s, diags
}

func buildScopeLimitations(ctx context.Context, m *scope.ComputerScopeLimitationsModel) (*proclassic.OsXConfigurationProfileScopeLimitations, diag.Diagnostics) {
	var diags diag.Diagnostics
	l := &proclassic.OsXConfigurationProfileScopeLimitations{}

	segs, d := scope.BuildIDSlice(ctx, m.NetworkSegmentIDs, func(id int) proclassic.OsXConfigurationProfileScopeLimitationsNetworkSegmentsNetworkSegmentItem {
		return proclassic.OsXConfigurationProfileScopeLimitationsNetworkSegmentsNetworkSegmentItem{ID: &id}
	})
	diags.Append(d...)
	if segs != nil {
		l.NetworkSegments = &proclassic.OsXConfigurationProfileScopeLimitationsNetworkSegments{NetworkSegment: segs}
	}

	ibeacons, d := scope.BuildIDSlice(ctx, m.IbeaconIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if ibeacons != nil {
		l.Ibeacons = &proclassic.OsXConfigurationProfileScopeLimitationsIbeacons{Ibeacon: ibeacons}
	}

	users, d := scope.BuildNameSlice(ctx, m.DirectoryServiceOrLocalUserNames, func(name string) proclassic.IDName {
		n := name
		return proclassic.IDName{Name: &n}
	})
	diags.Append(d...)
	if users != nil {
		l.Users = &proclassic.OsXConfigurationProfileScopeLimitationsUsers{User: users}
	}

	userGroups, d := scope.BuildNameSlice(ctx, m.DirectoryServiceUserGroupNames, func(name string) proclassic.IDName {
		n := name
		return proclassic.IDName{Name: &n}
	})
	diags.Append(d...)
	if userGroups != nil {
		l.UserGroups = &proclassic.OsXConfigurationProfileScopeLimitationsUserGroups{UserGroup: userGroups}
	}

	// Always emit the block when the caller's model declares `limitations`. A
	// scope PUT replaces the whole subtree, so an explicit empty element is the
	// clear gesture for a declared-empty category; undeclared (null) categories
	// are preserved upstream by the read-merge-write update, which hands this
	// builder a fully non-null merged model.
	return l, diags
}

func buildScopeExclusions(ctx context.Context, m *scope.ComputerScopeExclusionsModel) (*proclassic.OsXConfigurationProfileScopeExclusions, diag.Diagnostics) {
	var diags diag.Diagnostics
	e := &proclassic.OsXConfigurationProfileScopeExclusions{}

	computers, d := scope.BuildIDSlice(ctx, m.ComputerIDs, func(id int) proclassic.OsXConfigurationProfileScopeExclusionsComputersComputerItem {
		return proclassic.OsXConfigurationProfileScopeExclusionsComputersComputerItem{ID: &id}
	})
	diags.Append(d...)
	if computers != nil {
		e.Computers = &proclassic.OsXConfigurationProfileScopeExclusionsComputers{Computer: computers}
	}

	cgs, d := scope.BuildIDSlice(ctx, m.ComputerGroupIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if cgs != nil {
		e.ComputerGroups = &proclassic.OsXConfigurationProfileScopeExclusionsComputerGroups{ComputerGroup: cgs}
	}

	buildings, d := scope.BuildIDSlice(ctx, m.BuildingIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if buildings != nil {
		e.Buildings = &proclassic.OsXConfigurationProfileScopeExclusionsBuildings{Building: buildings}
	}

	departments, d := scope.BuildIDSlice(ctx, m.DepartmentIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if departments != nil {
		e.Departments = &proclassic.OsXConfigurationProfileScopeExclusionsDepartments{Department: departments}
	}

	jssUsers, d := scope.BuildIDSlice(ctx, m.UserIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if jssUsers != nil {
		e.JssUsers = &proclassic.OsXConfigurationProfileScopeExclusionsJssUsers{User: jssUsers}
	}

	jssUserGroups, d := scope.BuildIDSlice(ctx, m.UserGroupIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if jssUserGroups != nil {
		e.JssUserGroups = &proclassic.OsXConfigurationProfileScopeExclusionsJssUserGroups{UserGroup: jssUserGroups}
	}

	segs, d := scope.BuildIDSlice(ctx, m.NetworkSegmentIDs, func(id int) proclassic.OsXConfigurationProfileScopeExclusionsNetworkSegmentsNetworkSegmentItem {
		return proclassic.OsXConfigurationProfileScopeExclusionsNetworkSegmentsNetworkSegmentItem{ID: &id}
	})
	diags.Append(d...)
	if segs != nil {
		e.NetworkSegments = &proclassic.OsXConfigurationProfileScopeExclusionsNetworkSegments{NetworkSegment: segs}
	}

	ibeacons, d := scope.BuildIDSlice(ctx, m.IbeaconIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if ibeacons != nil {
		e.Ibeacons = &proclassic.OsXConfigurationProfileScopeExclusionsIbeacons{Ibeacon: ibeacons}
	}

	users, d := scope.BuildNameSlice(ctx, m.DirectoryServiceOrLocalUserNames, func(name string) proclassic.OsXConfigurationProfileScopeExclusionsUsersUserItem {
		n := name
		return proclassic.OsXConfigurationProfileScopeExclusionsUsersUserItem{Name: &n}
	})
	diags.Append(d...)
	if users != nil {
		e.Users = &proclassic.OsXConfigurationProfileScopeExclusionsUsers{User: users}
	}

	userGroups, d := scope.BuildNameSlice(ctx, m.DirectoryServiceUserGroupNames, func(name string) proclassic.IDName {
		n := name
		return proclassic.IDName{Name: &n}
	})
	diags.Append(d...)
	if userGroups != nil {
		e.UserGroups = &proclassic.OsXConfigurationProfileScopeExclusionsUserGroups{UserGroup: userGroups}
	}

	// Always emit the block when declared — see buildScopeLimitations.
	return e, diags
}

func buildSelfService(m *SelfServiceModel) (*proclassic.OsXConfigurationProfileSelfService, diag.Diagnostics) {
	var diags diag.Diagnostics
	ss := &proclassic.OsXConfigurationProfileSelfService{
		SelfServiceDisplayName:      helpers.OptionalStringPointer(m.SelfServiceDisplayName),
		InstallButtonText:           helpers.OptionalStringPointer(m.InstallButtonText),
		SelfServiceDescription:      helpers.OptionalStringPointer(m.SelfServiceDescription),
		ForceUsersToViewDescription: helpers.OptionalBoolPointer(m.EnsureUsersViewDescription),
		FeatureOnMainPage:           helpers.OptionalBoolPointer(m.FeatureOnMainPage),
		NotificationMessage:         helpers.OptionalStringPointer(m.NotificationMessage),
		NotificationSubject:         helpers.OptionalStringPointer(m.NotificationSubject),
	}

	// Notification (dual <notification> wire element via NotificationValue custom marshaller).
	hasNotificationBool := !m.DisplayNotifications.IsNull() && !m.DisplayNotifications.IsUnknown()
	hasNotificationLoc := !m.NotificationLocation.IsNull() && !m.NotificationLocation.IsUnknown() && m.NotificationLocation.ValueString() != ""
	if hasNotificationBool || hasNotificationLoc {
		nv := &proclassic.NotificationValue{}
		if hasNotificationBool {
			b := m.DisplayNotifications.ValueBool()
			nv.Enabled = &b
		}
		if hasNotificationLoc {
			s := m.NotificationLocation.ValueString()
			nv.Method = &s
		}
		ss.Notification = nv
	}

	// Security (removal_disallowed only — Password companion not surfaced).
	if v := m.RemovalDisallowed.ValueString(); !m.RemovalDisallowed.IsNull() && !m.RemovalDisallowed.IsUnknown() && v != "" {
		ss.Security = &proclassic.OsXConfigurationProfileSelfServiceSecurity{
			RemovalDisallowed: &v,
		}
	}

	if len(m.Categories) > 0 {
		items := make([]proclassic.OsXConfigurationProfileSelfServiceSelfServiceCategoriesCategoryItem, 0, len(m.Categories))
		for _, c := range m.Categories {
			item := proclassic.OsXConfigurationProfileSelfServiceSelfServiceCategoriesCategoryItem{
				DisplayIn: helpers.OptionalBoolPointer(c.DisplayIn),
				FeatureIn: helpers.OptionalBoolPointer(c.FeatureIn),
			}
			if id := helpers.StringIDPtr(c.ID); id != nil {
				item.ID = id
			}
			items = append(items, item)
		}
		ss.SelfServiceCategories = &proclassic.OsXConfigurationProfileSelfServiceSelfServiceCategories{
			Category: &items,
		}
	}

	return ss, diags
}
