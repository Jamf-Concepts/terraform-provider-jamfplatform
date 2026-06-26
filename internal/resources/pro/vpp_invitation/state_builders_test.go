// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package vpp_invitation

import (
	"context"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

func sampleAPI() *proclassic.VppInvitation {
	id, acct := 2, 3
	name := "Test"
	dm := "Make available in Self Service only"
	autoReg, allUsers := true, false
	groupID := 1
	groupName := "Group One"
	limName := "LDAP Admins"
	exclName := "Excluded LDAP"
	usageID := 3
	usageName := "user@example.com"
	usageStatus := "registered"
	epoch := 1778575794328

	return &proclassic.VppInvitation{
		ID: &id,
		General: &proclassic.VppInvitationGeneral{
			ID:                       &id,
			Name:                     &name,
			DistributionMethod:       &dm,
			AutoRegisterManagedUsers: &autoReg,
			VppAccount:               &proclassic.VppInvitationGeneralVppAccount{ID: &acct},
		},
		Scope: &proclassic.VppInvitationScope{
			AllJssUsers:   &allUsers,
			JssUserGroups: &proclassic.VppInvitationScopeJssUserGroups{UserGroup: &[]proclassic.IDName{{ID: &groupID, Name: &groupName}}},
			Limitations: &proclassic.VppInvitationScopeLimitations{
				UserGroups: &proclassic.VppInvitationScopeLimitationsUserGroups{UserGroup: &[]proclassic.IDName{{Name: &limName}}},
			},
			Exclusions: &proclassic.VppInvitationScopeExclusions{
				UserGroups: &proclassic.VppInvitationScopeExclusionsUserGroups{UserGroup: &[]proclassic.VppInvitationScopeExclusionsUserGroupsUserGroupItem{{Name: &exclName}}},
			},
		},
		InvitationUsages: &proclassic.VppInvitationInvitationUsages{
			Usage: &[]proclassic.VppInvitationInvitationUsagesUsageItem{
				{ID: &usageID, Name: &usageName, Status: &usageStatus, LastActionDateEpoch: &epoch},
			},
		},
	}
}

func TestAssignResourceModel_GeneralAndUsages(t *testing.T) {
	state := &VPPInvitationResourceModel{}
	assignVPPInvitationResourceModel(context.Background(), state, sampleAPI(), false)

	if state.ID.ValueString() != "2" {
		t.Errorf("id = %q", state.ID.ValueString())
	}
	if state.VPPAccountID.ValueString() != "3" {
		t.Errorf("vpp_account_id = %q", state.VPPAccountID.ValueString())
	}
	if !state.AutoRegisterManagedUsers.ValueBool() {
		t.Error("auto_register_managed_users not set")
	}
	// invitation_usages is always refreshed (Computed), even with nil scope state.
	if state.InvitationUsages.IsNull() || len(state.InvitationUsages.Elements()) != 1 {
		t.Errorf("invitation_usages = %v", state.InvitationUsages)
	}
}

func TestAssignResourceModel_ScopeOnlyWhenManaged(t *testing.T) {
	// nil Scope in state → scope not populated (server always echoes <scope>).
	state := &VPPInvitationResourceModel{}
	assignVPPInvitationResourceModel(context.Background(), state, sampleAPI(), false)
	if state.Scope != nil {
		t.Error("unmanaged scope block must stay nil")
	}

	// Managed scope (state.Scope non-nil) → refreshed, name-keyed groups by name.
	managed := &VPPInvitationResourceModel{Scope: &scope.UserScopeModel{
		Targets:     &scope.UserScopeTargetsModel{},
		Limitations: &scope.UserScopeLimitationsModel{},
		Exclusions:  &scope.UserScopeExclusionsModel{},
	}}
	assignVPPInvitationResourceModel(context.Background(), managed, sampleAPI(), false)
	if managed.Scope.Targets.JssUserGroupIDs.IsNull() || len(managed.Scope.Targets.JssUserGroupIDs.Elements()) != 1 {
		t.Errorf("jss_user_group_ids = %v", managed.Scope.Targets.JssUserGroupIDs)
	}
	if managed.Scope.Limitations.DirectoryServiceUserGroupNames.IsNull() {
		t.Error("limitations DS names should be populated by name")
	}
	if managed.Scope.Exclusions.DirectoryServiceUserGroupNames.IsNull() {
		t.Error("exclusions DS names should be populated by name")
	}
}

// TestAssignVPPInvitationResourceModel_IncludeUnmanagedHydratesFromScratch pins
// the config-generation contract: with includeUnmanaged set and an empty
// starting model, the wire-present scope is allocated and hydrated from the
// server.
func TestAssignVPPInvitationResourceModel_IncludeUnmanagedHydratesFromScratch(t *testing.T) {
	state := &VPPInvitationResourceModel{}
	assignVPPInvitationResourceModel(context.Background(), state, sampleAPI(), true)

	if state.Scope == nil || state.Scope.Targets == nil {
		t.Fatalf("expected scope.targets hydrated from scratch; got %+v", state.Scope)
	}
	if state.Scope.Targets.JssUserGroupIDs.IsNull() || len(state.Scope.Targets.JssUserGroupIDs.Elements()) != 1 {
		t.Errorf("expected jss_user_group_ids hydrated; got %v", state.Scope.Targets.JssUserGroupIDs)
	}
	if state.Scope.Limitations == nil || state.Scope.Limitations.DirectoryServiceUserGroupNames.IsNull() {
		t.Fatalf("expected limitations hydrated when wire-present; got %+v", state.Scope.Limitations)
	}
	if state.Scope.Exclusions == nil || state.Scope.Exclusions.DirectoryServiceUserGroupNames.IsNull() {
		t.Fatalf("expected exclusions hydrated when wire-present; got %+v", state.Scope.Exclusions)
	}
	if state.InvitationUsages.IsNull() || len(state.InvitationUsages.Elements()) != 1 {
		t.Errorf("expected invitation_usages hydrated; got %v", state.InvitationUsages)
	}
}

func TestFlattenUsages_EmptyKnown(t *testing.T) {
	out := flattenUsages(nil)
	if out.IsNull() || out.IsUnknown() {
		t.Error("nil usages must yield a known empty list, not null/unknown")
	}
	if len(out.Elements()) != 0 {
		t.Errorf("expected 0 elements, got %d", len(out.Elements()))
	}
}

func TestAssignDataSourceModel_PopulatesScopeAndUsages(t *testing.T) {
	state := &VPPInvitationDataSourceModel{}
	assignVPPInvitationDataSourceModel(context.Background(), state, sampleAPI())
	if state.Scope == nil {
		t.Fatal("DS must always populate scope")
	}
	if state.Scope.Targets.JssUserGroupIDs.IsNull() {
		t.Error("DS scope jss_user_group_ids should be populated")
	}
	if len(state.InvitationUsages.Elements()) != 1 {
		t.Errorf("DS usages = %v", state.InvitationUsages)
	}
}
