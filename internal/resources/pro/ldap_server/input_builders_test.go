// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ldap_server

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func derefStr(t *testing.T, p *string, field string) string {
	t.Helper()
	if p == nil {
		t.Fatalf("%s: expected non-nil pointer", field)
	}
	return *p
}

func TestBuildLdapServerInput_FullSimple(t *testing.T) {
	plan := LdapServerResourceModel{
		Connection: &ldapConnectionModel{
			DisplayName:        types.StringValue("corp-ad"),
			DirectoryService:   types.StringValue(serverTypeActiveDirectory),
			Hostname:           types.StringValue("ldap.example.com"),
			Port:               types.Int64Value(636),
			UseSSL:             types.BoolValue(true),
			AuthenticationType: types.StringValue(authTypeSimple),
			Account: &ldapAccountModel{
				DistinguishedUsername: types.StringValue("CN=svc,DC=example,DC=com"),
				Password:              types.StringNull(),
				PasswordWoVersion:     types.Int64Value(1),
			},
			ConnectionTimeout: types.Int64Value(15),
			SearchTimeout:     types.Int64Value(60),
			ReferralResponse:  types.StringValue(referralDefault),
			UseWildcards:      types.BoolValue(true),
		},
		MappingsForUsers: &ldapMappingsModel{
			UserMappings: &ldapUserMappingsModel{
				ObjectClassLimitation: types.StringValue(objectClassAny),
				SearchScope:           types.StringValue(searchScopeAllSubtrees),
				Username:              types.StringValue("mail"),
				Phone:                 types.StringValue("telephoneNumber"),
			},
			UserGroupMembershipMappings: &ldapMembershipMappingsModel{
				MembershipLocation:     types.StringValue(membershipGroupObject),
				MemberUserMapping:      types.StringValue("member"),
				GroupMembershipMapping: types.StringValue("memberOf"),
				UseLDAPCompare:         types.BoolValue(true),
			},
		},
	}

	pw := "s3cr3t"
	in := buildLdapServerInput(plan, &pw)

	if in.Connection == nil {
		t.Fatal("expected connection")
	}
	if got := derefStr(t, in.Connection.Name, "connection.name"); got != "corp-ad" {
		t.Errorf("name=%q", got)
	}
	if got := derefStr(t, in.Connection.ServerType, "server_type"); got != serverTypeActiveDirectory {
		t.Errorf("server_type=%q", got)
	}
	if in.Connection.Port == nil || *in.Connection.Port != 636 {
		t.Errorf("port not 636")
	}
	if in.Connection.Account == nil {
		t.Fatal("expected account")
	}
	if got := derefStr(t, in.Connection.Account.Password, "account.password"); got != "s3cr3t" {
		t.Errorf("password=%q, want threaded value", got)
	}
	if got := derefStr(t, in.Connection.Account.DistinguishedUsername, "account.dn"); got != "CN=svc,DC=example,DC=com" {
		t.Errorf("dn=%q", got)
	}

	// map_phone is the SDK-fixed tag (was the wrong map_telephone).
	if got := derefStr(t, in.MappingsForUsers.UserMappings.MapPhone, "map_phone"); got != "telephoneNumber" {
		t.Errorf("map_phone=%q", got)
	}
	// map_user_membership_to_group_field is the SDK-fixed *string (was *bool).
	if got := derefStr(t, in.MappingsForUsers.UserGroupMembershipMappings.MapUserMembershipToGroupField, "map_user_membership_to_group_field"); got != "memberOf" {
		t.Errorf("map_user_membership_to_group_field=%q", got)
	}
	if got := derefStr(t, in.MappingsForUsers.UserGroupMembershipMappings.MapGroupMembershipToUserField, "map_group_membership_to_user_field"); got != "member" {
		t.Errorf("map_group_membership_to_user_field=%q", got)
	}
}

func TestBuildLdapServerInput_PasswordOmittedWhenNil(t *testing.T) {
	plan := LdapServerResourceModel{
		Connection: &ldapConnectionModel{
			DisplayName: types.StringValue("corp-ad"),
			Account: &ldapAccountModel{
				DistinguishedUsername: types.StringValue("CN=svc"),
			},
		},
	}
	in := buildLdapServerInput(plan, nil)
	if in.Connection.Account == nil {
		t.Fatal("expected account block (dn supplied)")
	}
	if in.Connection.Account.Password != nil {
		t.Errorf("password must be omitted (nil) when not rotating; got %q", *in.Connection.Account.Password)
	}
}

func TestBuildLdapServerInput_AnonNoAccount(t *testing.T) {
	plan := LdapServerResourceModel{
		Connection: &ldapConnectionModel{
			DisplayName:        types.StringValue("anon-ldap"),
			AuthenticationType: types.StringValue(authTypeNone),
		},
	}
	in := buildLdapServerInput(plan, nil)
	if in.Connection == nil {
		t.Fatal("expected connection")
	}
	if in.Connection.Account != nil {
		t.Errorf("anonymous bind must emit no account block")
	}
}
