// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package vpp_assignment

import (
	"context"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

// Pointer locals (v := …; &v) are used throughout instead of intp/strp helper
// funcs: a trivial `func intp(i int) *int { return &i }` attracts the `go fix`
// inliner (it rewrites the body to the invalid `return new(i)` and is not
// idempotent), so the helpers are deliberately omitted.

func sampleAPI() *proclassic.VppAssignment {
	id, acct := 2, 3
	name := "Test"
	acctName := "Apple Business Manager"
	allUsers := false
	groupID := 1
	groupName := "Group One"
	limName := "LDAP Admins"
	exclName := "Excluded LDAP"
	iosAdam, iosName := 6444539476, "Jamf Self Service"
	macAdam, macName := 409203825, "Numbers"

	return &proclassic.VppAssignment{
		ID: &id,
		General: &proclassic.VppAssignmentGeneral{
			ID:                  &id,
			Name:                &name,
			VppAdminAccountID:   &acct,
			VppAdminAccountName: &acctName,
		},
		IosApps: &proclassic.VppAssignmentIosApps{
			IosApp: &[]proclassic.VppAssignmentIosAppsIosAppItem{
				{AdamID: &iosAdam, Name: &iosName},
			},
		},
		MacApps: &proclassic.VppAssignmentMacApps{
			MacApp: &[]proclassic.VppAssignmentMacAppsMacAppItem{
				{AdamID: &macAdam, Name: &macName},
			},
		},
		Scope: &proclassic.VppAssignmentScope{
			AllJssUsers:   &allUsers,
			JssUserGroups: &proclassic.VppAssignmentScopeJssUserGroups{UserGroup: &[]proclassic.IDName{{ID: &groupID, Name: &groupName}}},
			Limitations: &proclassic.VppAssignmentScopeLimitations{
				UserGroups: &proclassic.VppAssignmentScopeLimitationsUserGroups{UserGroup: &[]proclassic.IDName{{Name: &limName}}},
			},
			Exclusions: &proclassic.VppAssignmentScopeExclusions{
				UserGroups: &proclassic.VppAssignmentScopeExclusionsUserGroups{UserGroup: &[]proclassic.VppAssignmentScopeExclusionsUserGroupsUserGroupItem{{Name: &exclName}}},
			},
		},
	}
}

func TestAssignResourceModel_General(t *testing.T) {
	state := &VPPAssignmentResourceModel{
		IosAppAdamIDs: types.SetNull(types.Int64Type),
		MacAppAdamIDs: types.SetNull(types.Int64Type),
		EbookAdamIDs:  types.SetNull(types.Int64Type),
	}
	assignVPPAssignmentResourceModel(context.Background(), state, sampleAPI())

	if state.ID.ValueString() != "2" {
		t.Errorf("id = %q", state.ID.ValueString())
	}
	if state.VPPAdminAccountID.ValueString() != "3" {
		t.Errorf("vpp_admin_account_id = %q", state.VPPAdminAccountID.ValueString())
	}
	if state.VPPAdminAccountName.ValueString() != "Apple Business Manager" {
		t.Errorf("vpp_admin_account_name = %q", state.VPPAdminAccountName.ValueString())
	}
}

// TestAssignResourceModel_ContentOptOut: an unmanaged (null) content set stays
// null even though the server returns the collection; a managed set is refreshed.
func TestAssignResourceModel_ContentOptOut(t *testing.T) {
	// All content unmanaged → stays null.
	unmanaged := &VPPAssignmentResourceModel{
		IosAppAdamIDs: types.SetNull(types.Int64Type),
		MacAppAdamIDs: types.SetNull(types.Int64Type),
		EbookAdamIDs:  types.SetNull(types.Int64Type),
	}
	assignVPPAssignmentResourceModel(context.Background(), unmanaged, sampleAPI())
	if !unmanaged.IosAppAdamIDs.IsNull() {
		t.Error("unmanaged ios_app_adam_ids must stay null (don't fabricate management)")
	}

	// ios managed, mac/ebook unmanaged → only ios refreshed.
	managed := &VPPAssignmentResourceModel{
		IosAppAdamIDs: mustInt64Set(t, 1), // any non-null marks it managed
		MacAppAdamIDs: types.SetNull(types.Int64Type),
		EbookAdamIDs:  types.SetNull(types.Int64Type),
	}
	assignVPPAssignmentResourceModel(context.Background(), managed, sampleAPI())
	if managed.IosAppAdamIDs.IsNull() || len(managed.IosAppAdamIDs.Elements()) != 1 {
		t.Errorf("managed ios_app_adam_ids should be refreshed: %v", managed.IosAppAdamIDs)
	}
	if !managed.MacAppAdamIDs.IsNull() {
		t.Error("unmanaged mac_app_adam_ids must stay null")
	}
}

// TestFlattenAdamSet_EmptyKnown: a managed set that the server returns empty must
// reconcile to a known empty Set (NOT null) — these are plain Optional attrs, so a
// null after an empty config triggers a "provider produced inconsistent result".
func TestFlattenAdamSet_EmptyKnown(t *testing.T) {
	out := flattenAdamSet(context.Background(), nil)
	if out.IsNull() || out.IsUnknown() {
		t.Error("empty content must yield a known empty Set, not null/unknown")
	}
	if len(out.Elements()) != 0 {
		t.Errorf("expected 0 elements, got %d", len(out.Elements()))
	}
}

func TestFlattenAdamSet_RoundTrip(t *testing.T) {
	a, b := 6444539476, 409203825
	out := flattenAdamSet(context.Background(), []*int{&a, nil, &b})
	if out.IsNull() || len(out.Elements()) != 2 {
		t.Errorf("adam set round-trip = %v (nil entries skipped)", out)
	}
}

func TestAssignResourceModel_ScopeOnlyWhenManaged(t *testing.T) {
	// nil Scope in state → scope not populated (server always echoes <scope>).
	state := &VPPAssignmentResourceModel{
		IosAppAdamIDs: types.SetNull(types.Int64Type),
		MacAppAdamIDs: types.SetNull(types.Int64Type),
		EbookAdamIDs:  types.SetNull(types.Int64Type),
	}
	assignVPPAssignmentResourceModel(context.Background(), state, sampleAPI())
	if state.Scope != nil {
		t.Error("unmanaged scope block must stay nil")
	}

	managed := &VPPAssignmentResourceModel{
		IosAppAdamIDs: types.SetNull(types.Int64Type),
		MacAppAdamIDs: types.SetNull(types.Int64Type),
		EbookAdamIDs:  types.SetNull(types.Int64Type),
		Scope: &scope.UserScopeModel{
			Limitations: &scope.UserScopeLimitationsModel{},
			Exclusions:  &scope.UserScopeExclusionsModel{},
		},
	}
	assignVPPAssignmentResourceModel(context.Background(), managed, sampleAPI())
	if managed.Scope.JssUserGroupIDs.IsNull() || len(managed.Scope.JssUserGroupIDs.Elements()) != 1 {
		t.Errorf("jss_user_group_ids = %v", managed.Scope.JssUserGroupIDs)
	}
	if managed.Scope.Limitations.DirectoryServiceUserGroupNames.IsNull() {
		t.Error("limitations DS names should be populated by name")
	}
	if managed.Scope.Exclusions.DirectoryServiceUserGroupNames.IsNull() {
		t.Error("exclusions DS names should be populated by name")
	}
}

func TestAssignDataSourceModel_PopulatesContentAndScope(t *testing.T) {
	state := &VPPAssignmentDataSourceModel{}
	assignVPPAssignmentDataSourceModel(context.Background(), state, sampleAPI())
	if state.Scope == nil {
		t.Fatal("DS must always populate scope")
	}
	if state.Scope.JssUserGroupIDs.IsNull() {
		t.Error("DS scope jss_user_group_ids should be populated")
	}
	// DS content lists always known (read-only).
	if state.IosApps.IsNull() || len(state.IosApps.Elements()) != 1 {
		t.Errorf("DS ios_apps = %v", state.IosApps)
	}
	if state.Ebooks.IsNull() || len(state.Ebooks.Elements()) != 0 {
		t.Errorf("DS ebooks must be a known empty list when absent, got %v", state.Ebooks)
	}
}

func TestIntStringOrNull_NegativeIsNull(t *testing.T) {
	neg, pos := -1, 3
	if !intStringOrNull(&neg).IsNull() {
		t.Error("vpp_admin_account_id = -1 (no valid account) must map to null")
	}
	if intStringOrNull(&pos).ValueString() != "3" {
		t.Error("valid account id must map to its decimal form")
	}
}
