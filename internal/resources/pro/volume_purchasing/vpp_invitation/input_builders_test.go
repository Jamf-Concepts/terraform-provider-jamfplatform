// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package vpp_invitation

import (
	"context"
	"net/url"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func mustStringSet(t *testing.T, vals ...string) types.Set {
	t.Helper()
	elems := make([]types.String, 0, len(vals))
	for _, v := range vals {
		elems = append(elems, types.StringValue(v))
	}
	set, diags := types.SetValueFrom(context.Background(), types.StringType, elems)
	if diags.HasError() {
		t.Fatalf("set build diags: %v", diags)
	}
	return set
}

func TestBuildInput_GeneralAndAccount(t *testing.T) {
	plan := VPPInvitationResourceModel{
		Name:                     types.StringValue("VPP Invite"),
		VPPAccountID:             types.StringValue("3"),
		DistributionMethod:       types.StringValue("Make available in Self Service only"),
		AutoRegisterManagedUsers: types.BoolValue(true),
	}
	in, diags := buildVPPInvitationInput(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if in.General.Name == nil || *in.General.Name != "VPP Invite" {
		t.Errorf("name not carried: %v", in.General.Name)
	}
	if in.General.DistributionMethod == nil || *in.General.DistributionMethod != "Make available in Self Service only" {
		t.Errorf("distribution_method not carried: %v", in.General.DistributionMethod)
	}
	if in.General.VppAccount == nil || in.General.VppAccount.ID == nil || *in.General.VppAccount.ID != 3 {
		t.Errorf("vpp_account id not carried: %+v", in.General.VppAccount)
	}
	if in.General.AutoRegisterManagedUsers == nil || !*in.General.AutoRegisterManagedUsers {
		t.Error("auto_register_managed_users not carried")
	}
	if in.Scope != nil {
		t.Errorf("nil scope model must omit <scope>, got %+v", in.Scope)
	}
}

func TestBuildInput_EmailFields(t *testing.T) {
	plan := VPPInvitationResourceModel{
		Name:               types.StringValue("x"),
		VPPAccountID:       types.StringValue("3"),
		DistributionMethod: types.StringValue("Send emails"),
		SenderName:         types.StringValue("IT"),
		SenderEmailAddress: types.StringValue("it@example.com"),
		Subject:            types.StringValue("Register"),
		Message:            types.StringValue("link %@"),
		RequireLogin:       types.BoolValue(true),
	}
	in, _ := buildVPPInvitationInput(context.Background(), plan)
	if in.General.SenderName == nil || *in.General.SenderName != "IT" {
		t.Error("sender_name not carried")
	}
	// message is form-URL-encoded on the wire so the server's form-decode
	// round-trips it verbatim (incl. the %@ placeholder).
	if in.General.Message == nil || *in.General.Message != "link+%25%40" {
		t.Errorf("message not form-URL-encoded, got %v", in.General.Message)
	}
	if in.General.RequireLogin == nil || !*in.General.RequireLogin {
		t.Error("require_login not carried")
	}
}

func TestEncodedMessagePointer(t *testing.T) {
	if encodedMessagePointer(types.StringNull()) != nil {
		t.Error("null message must omit")
	}
	if encodedMessagePointer(types.StringUnknown()) != nil {
		t.Error("unknown message must omit")
	}
	// %@ + newline + percent + space must encode so the server's form-decode
	// reproduces the original exactly.
	got := encodedMessagePointer(types.StringValue("Click %@\n100% a b"))
	if got == nil {
		t.Fatal("expected encoded message")
	}
	want := "Click+%25%40%0A100%25+a+b"
	if *got != want {
		t.Errorf("encoded = %q, want %q", *got, want)
	}
	if dec, err := url.QueryUnescape(*got); err != nil || dec != "Click %@\n100% a b" {
		t.Errorf("round-trip decode = %q (err %v), want original", dec, err)
	}
}

// TestBuildScope_AlwaysEmitsSkeleton verifies the always-emit contract: when the
// scope block is declared, every collection wrapper is present (empty inner slice
// when the set is null/empty) so the server full-replaces and clears.
func TestBuildScope_AlwaysEmitsSkeleton(t *testing.T) {
	plan := VPPInvitationResourceModel{
		Name:               types.StringValue("x"),
		VPPAccountID:       types.StringValue("3"),
		DistributionMethod: types.StringValue("Make available in Self Service only"),
		Scope:              &VPPInvitationScopeModel{}, // empty block, all sets null
	}
	in, _ := buildVPPInvitationInput(context.Background(), plan)
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
	plan := VPPInvitationResourceModel{
		Name:               types.StringValue("x"),
		VPPAccountID:       types.StringValue("3"),
		DistributionMethod: types.StringValue("Make available in Self Service only"),
		Scope: &VPPInvitationScopeModel{
			AllJSSUsers:     types.BoolValue(false),
			JSSUserGroupIDs: mustStringSet(t, "1"),
			Limitations: &VPPInvitationScopeLimitationsModel{
				DirectoryServiceUserGroupNames: mustStringSet(t, "LDAP Admins"),
			},
			Exclusions: &VPPInvitationScopeExclusionsModel{
				JSSUserIDs:                     mustStringSet(t, "5"),
				DirectoryServiceUserGroupNames: mustStringSet(t, "Excluded LDAP Group"),
			},
		},
	}
	in, _ := buildVPPInvitationInput(context.Background(), plan)
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
