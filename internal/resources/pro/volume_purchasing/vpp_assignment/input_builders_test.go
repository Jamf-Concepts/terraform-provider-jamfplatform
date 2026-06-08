// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package vpp_assignment

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

func mustInt64Set(t *testing.T, vals ...int64) types.Set {
	t.Helper()
	// A variadic nil slice would marshal to a NULL set; force a non-nil slice so
	// an explicit zero-arg call models the config `[]` (known empty), not omission.
	if vals == nil {
		vals = []int64{}
	}
	set, diags := types.SetValueFrom(context.Background(), types.Int64Type, vals)
	if diags.HasError() {
		t.Fatalf("int64 set build diags: %v", diags)
	}
	return set
}

func mustStringSet(t *testing.T, vals ...string) types.Set {
	t.Helper()
	set, diags := types.SetValueFrom(context.Background(), types.StringType, vals)
	if diags.HasError() {
		t.Fatalf("string set build diags: %v", diags)
	}
	return set
}

func TestBuildInput_GeneralAndAccount(t *testing.T) {
	plan := VPPAssignmentResourceModel{
		Name:              types.StringValue("VPP Assignment"),
		VPPAdminAccountID: types.StringValue("3"),
		IosAppAdamIDs:     types.SetNull(types.Int64Type),
		MacAppAdamIDs:     types.SetNull(types.Int64Type),
		EbookAdamIDs:      types.SetNull(types.Int64Type),
	}
	in, diags := buildVPPAssignmentInput(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if in.General.Name == nil || *in.General.Name != "VPP Assignment" {
		t.Errorf("name not carried: %v", in.General.Name)
	}
	if in.General.VppAdminAccountID == nil || *in.General.VppAdminAccountID != 3 {
		t.Errorf("vpp_admin_account_id not carried: %v", in.General.VppAdminAccountID)
	}
	// null content sets → wire blocks omitted (server retains).
	if in.IosApps != nil || in.MacApps != nil || in.Ebooks != nil {
		t.Error("null content sets must omit their wire blocks")
	}
	if in.Scope != nil {
		t.Errorf("nil scope model must omit <scope>, got %+v", in.Scope)
	}
}

// TestBuildInput_ContentOptOut is the correctness-critical test: null omits the
// block (retain), [] emits a non-nil-but-empty wrapper (clear), and a populated
// set emits the full list (replace). The three are independent.
func TestBuildInput_ContentOptOut(t *testing.T) {
	plan := VPPAssignmentResourceModel{
		Name:              types.StringValue("x"),
		VPPAdminAccountID: types.StringValue("3"),
		IosAppAdamIDs:     mustInt64Set(t, 6444539476),    // populated → replace
		MacAppAdamIDs:     mustInt64Set(t),                // empty []  → clear
		EbookAdamIDs:      types.SetNull(types.Int64Type), // null      → retain
	}
	in, diags := buildVPPAssignmentInput(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}

	// ios_apps: populated → wrapper present, list of one, adam_id carried, name nil.
	if in.IosApps == nil || in.IosApps.IosApp == nil || len(*in.IosApps.IosApp) != 1 {
		t.Fatalf("populated ios_apps must full-replace: %+v", in.IosApps)
	}
	item := (*in.IosApps.IosApp)[0]
	if item.AdamID == nil || *item.AdamID != 6444539476 {
		t.Errorf("ios adam_id not carried: %+v", item.AdamID)
	}
	if item.Name != nil {
		t.Error("content item name must NOT be sent (server resolves it)")
	}

	// mac_apps: empty [] → wrapper present, inner slice nil/empty (marshals as a
	// clearing element).
	if in.MacApps == nil {
		t.Fatal("empty mac_app_adam_ids must emit a (clearing) <mac_apps> wrapper")
	}
	if in.MacApps.MacApp != nil && len(*in.MacApps.MacApp) != 0 {
		t.Errorf("empty mac_app_adam_ids must yield an empty inner slice, got %+v", in.MacApps.MacApp)
	}

	// ebooks: null → omitted entirely (retain).
	if in.Ebooks != nil {
		t.Errorf("null ebook_adam_ids must omit <ebooks>, got %+v", in.Ebooks)
	}
}

// TestBuildScope_AlwaysEmitsSkeleton verifies the always-emit contract: when the
// scope block is declared, every collection wrapper is present (empty inner slice
// when the set is null/empty) so the server full-replaces and clears.
func TestBuildScope_AlwaysEmitsSkeleton(t *testing.T) {
	plan := VPPAssignmentResourceModel{
		Name:              types.StringValue("x"),
		VPPAdminAccountID: types.StringValue("3"),
		Scope:             &scope.UserScopeModel{}, // empty block, all sets null
	}
	in, _ := buildVPPAssignmentInput(context.Background(), plan)
	if in.Scope == nil {
		t.Fatal("declared scope must emit <scope>")
	}
	if in.Scope.JssUsers == nil || in.Scope.JssUserGroups == nil {
		t.Error("target wrappers must always be present")
	}
	if in.Scope.JssUsers.User != nil {
		t.Error("empty jss_user_ids must yield an empty (nil-slice) wrapper")
	}
	if in.Scope.Limitations == nil || in.Scope.Limitations.UserGroups == nil {
		t.Error("limitations wrapper must always be present")
	}
	if in.Scope.Exclusions == nil || in.Scope.Exclusions.JssUsers == nil ||
		in.Scope.Exclusions.JssUserGroups == nil || in.Scope.Exclusions.UserGroups == nil {
		t.Error("all exclusion wrappers must always be present")
	}
}

func TestBuildScope_PopulatedTargetsAndNameKeyedGroups(t *testing.T) {
	plan := VPPAssignmentResourceModel{
		Name:              types.StringValue("x"),
		VPPAdminAccountID: types.StringValue("3"),
		Scope: &scope.UserScopeModel{
			AllJssUsers:     types.BoolValue(false),
			JssUserGroupIDs: mustStringSet(t, "1"),
			Limitations: &scope.UserScopeLimitationsModel{
				DirectoryServiceUserGroupNames: mustStringSet(t, "LDAP Admins"),
			},
			Exclusions: &scope.UserScopeExclusionsModel{
				JssUserIDs:                     mustStringSet(t, "5"),
				DirectoryServiceUserGroupNames: mustStringSet(t, "Excluded LDAP Group"),
			},
		},
	}
	in, _ := buildVPPAssignmentInput(context.Background(), plan)
	if in.Scope.JssUserGroups.UserGroup == nil || len(*in.Scope.JssUserGroups.UserGroup) != 1 || *(*in.Scope.JssUserGroups.UserGroup)[0].ID != 1 {
		t.Errorf("jss_user_group_ids not id-carried: %+v", in.Scope.JssUserGroups.UserGroup)
	}
	// limitations.user_groups: NAME-keyed (Name set, ID nil).
	lim := in.Scope.Limitations.UserGroups.UserGroup
	if lim == nil || len(*lim) != 1 || (*lim)[0].Name == nil || *(*lim)[0].Name != "LDAP Admins" {
		t.Errorf("limitations DS group not name-carried: %+v", lim)
	}
	if (*lim)[0].ID != nil {
		t.Error("limitations DS group must be name-only (id nil)")
	}
	// exclusions.jss_users: id-keyed.
	exUsers := in.Scope.Exclusions.JssUsers.User
	if exUsers == nil || len(*exUsers) != 1 || *(*exUsers)[0].ID != 5 {
		t.Errorf("exclusions jss_user_ids not id-carried: %+v", exUsers)
	}
	// exclusions.user_groups: name-only item type.
	exDS := in.Scope.Exclusions.UserGroups.UserGroup
	if exDS == nil || len(*exDS) != 1 || (*exDS)[0].Name == nil || *(*exDS)[0].Name != "Excluded LDAP Group" {
		t.Errorf("exclusions DS group not name-carried: %+v", exDS)
	}
}

func TestStringIDPtr(t *testing.T) {
	if stringIDPtr(types.StringNull()) != nil {
		t.Error("null must be nil")
	}
	if stringIDPtr(types.StringValue("")) != nil {
		t.Error("empty must be nil")
	}
	if stringIDPtr(types.StringValue("abc")) != nil {
		t.Error("non-numeric must be nil")
	}
	if p := stringIDPtr(types.StringValue("42")); p == nil || *p != 42 {
		t.Errorf("42 must parse, got %v", p)
	}
}
