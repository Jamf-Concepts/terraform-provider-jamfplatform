// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ldap_server

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// assignLdapServerResourceModel populates a resource model from a GET
// response. The server nests the allocated ID under `<connection><id>`; we
// only overwrite state.ID when that value is non-nil so a transient GET that
// drops it does not clobber the value Create persisted from the top-level id.
//
// The bind password is never round-tripped: `password` is WriteOnly (the
// framework excludes it from state) and the GET returns only the masked
// `password_sha256` sentinel, which carries no drift signal and is dropped.
// `password_wo_version` is preserved verbatim from the prior state.
//
// gateMappingsToDeclared controls how the (always-superset) server mappings
// are reduced into state:
//   - true  (Create / Update / refresh-Read): keep only the sub-blocks present
//     in the prior model so the result matches the plan/config — the schema
//     models the blocks as Optional (not Optional+Computed) because the
//     framework cannot decode an unknown object into a typed model pointer, so
//     undeclared blocks must be dropped to avoid "inconsistent result after
//     apply".
//   - false (import): there is no config to be consistent with, so populate
//     every sub-block the server returns for full-fidelity import.
func assignLdapServerResourceModel(state *LdapServerResourceModel, s *proclassic.LdapServer, gateMappingsToDeclared bool) diag.Diagnostics {
	var diags diag.Diagnostics
	if s == nil {
		return diags
	}
	if s.Connection != nil && s.Connection.ID != nil {
		state.ID = helpers.StringValueFromIntPtr(s.Connection.ID)
	}

	var existingAccount *ldapAccountModel
	if state.Connection != nil {
		existingAccount = state.Connection.Account
	}
	state.Connection = assignConnectionModel(s.Connection, existingAccount)

	full := assignMappingsModel(s.MappingsForUsers)
	if gateMappingsToDeclared {
		state.MappingsForUsers = gateMappings(state.MappingsForUsers, full)
	} else {
		state.MappingsForUsers = full
	}
	return diags
}

// gateMappings reduces the fully-populated server mappings to only the
// sub-blocks the user declared (declared != nil). Returns nil when the user
// did not declare mappings_for_users at all.
func gateMappings(declared, full *ldapMappingsModel) *ldapMappingsModel {
	if declared == nil {
		return nil
	}
	if full == nil {
		return declared
	}
	out := &ldapMappingsModel{}
	if declared.UserMappings != nil {
		out.UserMappings = full.UserMappings
	}
	if declared.UserGroupMappings != nil {
		out.UserGroupMappings = full.UserGroupMappings
	}
	if declared.UserGroupMembershipMappings != nil {
		out.UserGroupMembershipMappings = full.UserGroupMembershipMappings
	}
	return out
}

// assignLdapServerDataSourceModel populates a data source model from a GET
// response. Symmetric with the resource builder; the bind account's password
// and rotation companion are never populated (the data source is read-only
// and Jamf Pro never echoes the plaintext).
func assignLdapServerDataSourceModel(state *LdapServerDataSourceModel, s *proclassic.LdapServer) diag.Diagnostics {
	var diags diag.Diagnostics
	if s == nil {
		return diags
	}
	if s.Connection != nil {
		if s.Connection.ID != nil {
			state.ID = helpers.StringValueFromIntPtr(s.Connection.ID)
		}
		if s.Connection.Name != nil {
			state.Name = helpers.StringPointerValueOrNull(s.Connection.Name)
		}
	}
	state.Connection = assignConnectionModel(s.Connection, nil)
	state.MappingsForUsers = assignMappingsModel(s.MappingsForUsers)
	return diags
}

// assignConnectionModel decodes the connection block. `existingAccount`
// carries the prior state's account model so password_wo_version survives a
// refresh.
func assignConnectionModel(c *proclassic.LdapServerConnection, existingAccount *ldapAccountModel) *ldapConnectionModel {
	if c == nil {
		return nil
	}
	return &ldapConnectionModel{
		DisplayName:        helpers.StringPointerValueOrNull(c.Name),
		DirectoryService:   helpers.StringPointerValueOrNull(c.ServerType),
		Hostname:           helpers.StringPointerValueOrNull(c.Hostname),
		Port:               helpers.Int64FromIntPtr(c.Port),
		UseSSL:             helpers.BoolPointerValueOrNull(c.UseSsl),
		AuthenticationType: helpers.StringPointerValueOrNull(c.AuthenticationType),
		Account:            assignAccountModel(c.Account, existingAccount),
		ConnectionTimeout:  helpers.Int64FromIntPtr(c.OpenCloseTimeout),
		SearchTimeout:      helpers.Int64FromIntPtr(c.SearchTimeout),
		ReferralResponse:   helpers.StringPointerValueOrNull(c.ReferralResponse),
		UseWildcards:       helpers.BoolPointerValueOrNull(c.UseWildcards),
		IsEnabled:          helpers.BoolPointerValueOrNull(c.IsEnabled),
		MigratedToID:       helpers.Int64FromIntPtr(c.MigratedToID),
		CertificatesUsed:   helpers.StringPointerValueOrNull(c.CertificatesUsed),
	}
}

// assignAccountModel decodes the bind account. An empty distinguished username
// means anonymous bind (authentication_type = "none") — return the prior
// account (nil for a fresh anonymous server) so a planned-null account stays
// null. Otherwise carry password_wo_version over from the prior state.
func assignAccountModel(a *proclassic.LdapServerConnectionAccount, existing *ldapAccountModel) *ldapAccountModel {
	dn := ""
	if a != nil && a.DistinguishedUsername != nil {
		dn = *a.DistinguishedUsername
	}
	if dn == "" {
		return existing
	}
	out := &ldapAccountModel{
		DistinguishedUsername: types.StringValue(dn),
		Password:              types.StringNull(),
		PasswordWoVersion:     types.Int64Null(),
	}
	if existing != nil {
		out.PasswordWoVersion = existing.PasswordWoVersion
	}
	return out
}

// assignMappingsModel decodes the mappings_for_users block and its three
// sub-blocks. The server echoes all three regardless of what the user set, so
// we always populate whatever the response carries; the Optional+Computed
// sub-block plan modifiers keep omitted blocks stable across plans.
func assignMappingsModel(m *proclassic.LdapServerMappingsForUsers) *ldapMappingsModel {
	if m == nil {
		return nil
	}
	out := &ldapMappingsModel{}
	if u := m.UserMappings; u != nil {
		out.UserMappings = &ldapUserMappingsModel{
			ObjectClassLimitation: helpers.StringPointerValueOrNull(u.MapObjectClassToAnyOrAll),
			ObjectClasses:         helpers.StringPointerValueOrNull(u.ObjectClasses),
			SearchBase:            helpers.StringPointerValueOrNull(u.SearchBase),
			SearchScope:           helpers.StringPointerValueOrNull(u.SearchScope),
			UserID:                helpers.StringPointerValueOrNull(u.MapUserID),
			Username:              helpers.StringPointerValueOrNull(u.MapUsername),
			RealName:              helpers.StringPointerValueOrNull(u.MapRealname),
			EmailAddress:          helpers.StringPointerValueOrNull(u.MapEmailAddress),
			AppendToEmailResults:  helpers.StringPointerValueOrNull(u.AppendToEmailResults),
			Department:            helpers.StringPointerValueOrNull(u.MapDepartment),
			Building:              helpers.StringPointerValueOrNull(u.MapBuilding),
			Room:                  helpers.StringPointerValueOrNull(u.MapRoom),
			Phone:                 helpers.StringPointerValueOrNull(u.MapPhone),
			Position:              helpers.StringPointerValueOrNull(u.MapPosition),
			UserUUID:              helpers.StringPointerValueOrNull(u.MapUserUUID),
		}
	}
	if g := m.UserGroupMappings; g != nil {
		out.UserGroupMappings = &ldapUserGroupMappingsModel{
			ObjectClassLimitation: helpers.StringPointerValueOrNull(g.MapObjectClassToAnyOrAll),
			ObjectClasses:         helpers.StringPointerValueOrNull(g.ObjectClasses),
			SearchBase:            helpers.StringPointerValueOrNull(g.SearchBase),
			SearchScope:           helpers.StringPointerValueOrNull(g.SearchScope),
			GroupID:               helpers.StringPointerValueOrNull(g.MapGroupID),
			GroupName:             helpers.StringPointerValueOrNull(g.MapGroupName),
			GroupUUID:             helpers.StringPointerValueOrNull(g.MapGroupUUID),
		}
	}
	if b := m.UserGroupMembershipMappings; b != nil {
		out.UserGroupMembershipMappings = &ldapMembershipMappingsModel{
			MembershipLocation:                helpers.StringPointerValueOrNull(b.UserGroupMembershipStoredIn),
			MemberUserMapping:                 helpers.StringPointerValueOrNull(b.MapGroupMembershipToUserField),
			GroupMembershipMapping:            helpers.StringPointerValueOrNull(b.MapUserMembershipToGroupField),
			AppendToUsername:                  helpers.StringPointerValueOrNull(b.AppendToUsername),
			UseDN:                             helpers.BoolPointerValueOrNull(b.UseDn),
			UseLDAPCompare:                    helpers.BoolPointerValueOrNull(b.UserGroupMembershipUseLdapCompare),
			RecursiveLookups:                  helpers.BoolPointerValueOrNull(b.RecursiveLookups),
			MapUserMembershipUseDN:            helpers.BoolPointerValueOrNull(b.MapUserMembershipUseDn),
			MembershipCalculationOptimization: helpers.BoolPointerValueOrNull(b.MembershipScopingOptimization),
			ObjectClassLimitation:             helpers.StringPointerValueOrNull(b.MapObjectClassToAnyOrAll),
			ObjectClasses:                     helpers.StringPointerValueOrNull(b.ObjectClasses),
			SearchBase:                        helpers.StringPointerValueOrNull(b.SearchBase),
			SearchScope:                       helpers.StringPointerValueOrNull(b.SearchScope),
			UsernameMapping:                   helpers.StringPointerValueOrNull(b.Username),
			GroupIDMapping:                    helpers.StringPointerValueOrNull(b.GroupID),
			UseMemberFieldForSelectQueries:    helpers.BoolPointerValueOrNull(b.GroupMembershipEnabledWhenUserMembershipSelected),
		}
	}
	return out
}
