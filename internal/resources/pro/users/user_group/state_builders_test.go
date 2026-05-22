// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package user_group

import (
	"context"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAssignUserGroupResourceModel_Static_PopulatesMembers(t *testing.T) {
	id := 3
	ug := &proclassic.UserGroup{
		ID:               &id,
		Name:             new("Excluded Users"),
		IsSmart:          new(false),
		IsNotifyOnChange: new(false),
		Site:             &proclassic.SiteObject{ID: new(-1), Name: new("NONE")},
		Users: &proclassic.UserGroupUsers{User: &[]proclassic.UserGroupUsersUserItem{
			{ID: new(9), Username: new("david@example.com"), FullName: new("David Norris"), EmailAddress: new("david@example.com")},
		}},
	}

	initialMembers, mDiags := types.SetValueFrom(context.Background(), types.StringType, []string{"99"})
	if mDiags.HasError() {
		t.Fatalf("initial members: %v", mDiags)
	}
	state := &UserGroupResourceModel{Members: initialMembers}
	diags := assignUserGroupResourceModel(context.Background(), state, ug, true)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	if state.ID.ValueString() != "3" {
		t.Errorf("ID expected 3, got %s", state.ID)
	}
	if state.GroupType.ValueString() != "static" {
		t.Errorf("GroupType expected static, got %s", state.GroupType)
	}
	if state.SiteID.ValueString() != "-1" {
		t.Errorf("SiteID expected -1, got %s", state.SiteID)
	}
	if state.SiteName.ValueString() != "NONE" {
		t.Errorf("SiteName expected NONE, got %s", state.SiteName)
	}
	if state.MemberCount.ValueInt64() != 1 {
		t.Errorf("MemberCount expected 1, got %d", state.MemberCount.ValueInt64())
	}
	if state.Members.IsNull() {
		t.Fatal("Members must be non-null when manageMembers=true on static group")
	}
}

func TestAssignUserGroupResourceModel_Smart_MembersAlwaysNull(t *testing.T) {
	id := 2
	ug := &proclassic.UserGroup{
		ID:      &id,
		Name:    new("Smart Group"),
		IsSmart: new(true),
		Site:    &proclassic.SiteObject{ID: new(-1), Name: new("NONE")},
		Criteria: &proclassic.UserGroupCriteria{Criterion: &[]proclassic.Criterion{
			{Name: new("User Group"), Priority: new(0), AndOr: new("and"), SearchType: new("member of"), Value: new("All Managed Apple IDs"), OpeningParen: new(false), ClosingParen: new(false)},
		}},
		Users: &proclassic.UserGroupUsers{User: &[]proclassic.UserGroupUsersUserItem{
			{ID: new(6), Username: new("a@b.com")},
			{ID: new(7), Username: new("c@d.com")},
		}},
	}

	state := &UserGroupResourceModel{}
	diags := assignUserGroupResourceModel(context.Background(), state, ug, true) // manageMembers irrelevant for smart
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	if state.GroupType.ValueString() != "smart" {
		t.Errorf("GroupType expected smart")
	}
	if !state.Members.IsNull() {
		t.Errorf("Members must be null for smart group, got %v", state.Members)
	}
	if state.MemberCount.ValueInt64() != 2 {
		t.Errorf("MemberCount still surfaces from <users> for smart groups: expected 2, got %d", state.MemberCount.ValueInt64())
	}
	if len(state.Criteria) != 1 {
		t.Errorf("Criteria expected 1, got %d", len(state.Criteria))
	}
}

func TestAssignUserGroupDataSourceModel_PopulatesUsersBlock(t *testing.T) {
	id := 2
	ug := &proclassic.UserGroup{
		ID:      &id,
		Name:    new("Smart Group"),
		IsSmart: new(true),
		Users: &proclassic.UserGroupUsers{User: &[]proclassic.UserGroupUsersUserItem{
			{ID: new(6), Username: new("a@b.com"), FullName: new("Person A"), EmailAddress: new("a@b.com")},
		}},
	}

	state := &UserGroupDataSourceModel{}
	assignUserGroupDataSourceModel(state, ug)
	if len(state.Users) != 1 {
		t.Fatalf("Users expected 1, got %d", len(state.Users))
	}
	if state.Users[0].ID.ValueString() != "6" {
		t.Errorf("Users[0].ID expected 6, got %s", state.Users[0].ID)
	}
	if state.Users[0].FullName.ValueString() != "Person A" {
		t.Errorf("Users[0].FullName expected 'Person A'")
	}
}

func TestGroupTypeFromIsSmart(t *testing.T) {
	tests := []struct {
		in   *bool
		want string
	}{
		{new(true), "smart"},
		{new(false), "static"},
	}
	for _, tt := range tests {
		got := groupTypeFromIsSmart(tt.in)
		if got.ValueString() != tt.want {
			t.Errorf("isSmart=%v: expected %q, got %q", tt.in, tt.want, got.ValueString())
		}
	}
	if !groupTypeFromIsSmart(nil).IsNull() {
		t.Errorf("nil IsSmart must yield null")
	}
}

func TestFlattenSite(t *testing.T) {
	id, name := flattenSite(&proclassic.SiteObject{ID: new(5), Name: new("Site5")})
	if id == nil || *id != "5" || name == nil || *name != "Site5" {
		t.Errorf("unexpected: id=%v name=%v", id, name)
	}
	id, name = flattenSite(nil)
	if id != nil || name != nil {
		t.Errorf("nil site must yield (nil, nil)")
	}
}
