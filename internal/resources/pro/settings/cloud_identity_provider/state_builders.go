// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package cloud_identity_provider

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// assignGoogleState folds a Cloud LDAP GET response into the resource model.
//
// The keystore `wo_version` rotation trigger and the WriteOnly `file` /
// `password` are not returned by the server. `wo_version` is preserved from
// the prior model (`priorWoVersion`) so a refresh does not null it out;
// `file` / `password` are WriteOnly and excluded from state by the framework.
func assignGoogleState(state *CloudIdentityProviderResourceModel, resp *pro.LdapConfigurationResponse, priorWoVersion types.Int64) {
	if resp == nil {
		return
	}

	// Capture the prior mappings shape before rebuilding `google`. `mappings`
	// (and its sub-blocks) are Optional, not Computed: the server always
	// returns generated mappings, but if the user did not author the block we
	// must keep it null in state — otherwise Terraform reports "planned null,
	// got object". So we only surface the sub-blocks the user actually
	// configured.
	var priorMappings *cloudLdapMappingsModel
	if state.Google != nil {
		priorMappings = state.Google.Mappings
	}

	if resp.CloudIDPCommon != nil {
		if resp.CloudIDPCommon.ID != "" {
			state.ID = types.StringValue(resp.CloudIDPCommon.ID)
		}
		state.DisplayName = types.StringValue(resp.CloudIDPCommon.DisplayName)
		state.ProviderName = types.StringValue(providerNameFromWire(resp.CloudIDPCommon.ProviderName))
	}

	google := &cloudIdentityProviderGoogleModel{}

	if resp.Server != nil {
		s := resp.Server
		server := &cloudLdapServerModel{
			ServerURL:                                types.StringValue(s.ServerURL),
			DomainName:                               types.StringValue(s.DomainName),
			Port:                                     types.Int64Value(int64(s.Port)),
			ConnectionType:                           types.StringValue(s.ConnectionType),
			ConnectionTimeout:                        types.Int64Value(int64(s.ConnectionTimeout)),
			SearchTimeout:                            types.Int64Value(int64(s.SearchTimeout)),
			UseWildcards:                             types.BoolValue(s.UseWildcards),
			Enabled:                                  types.BoolValue(s.Enabled),
			MembershipCalculationOptimizationEnabled: types.BoolValue(s.MembershipCalculationOptimizationEnabled),
			Keystore:                                 assignKeystoreState(s.Keystore, priorWoVersion),
		}
		google.Server = server
	}

	google.Mappings = assignMappingsState(resp.Mappings, priorMappings)

	state.Google = google
}

// assignKeystoreState builds the keystore echo model. file / password are
// WriteOnly (never in state); wo_version is carried from the prior model.
func assignKeystoreState(k *pro.CloudLdapKeystore, priorWoVersion types.Int64) *cloudLdapKeystoreModel {
	out := &cloudLdapKeystoreModel{
		File:      types.StringNull(),
		Password:  types.StringNull(),
		WoVersion: priorWoVersion,
	}
	if k == nil {
		out.FileName = types.StringNull()
		out.Type = types.StringNull()
		out.Subject = types.StringNull()
		out.ExpirationDate = types.StringNull()
		return out
	}
	out.FileName = types.StringValue(k.FileName)
	out.Type = types.StringValue(k.Type)
	out.Subject = types.StringValue(k.Subject)
	out.ExpirationDate = stringPtrValueOrNull(k.ExpirationDate)
	return out
}

// assignMappingsState builds the mappings model from a GET response, scoped to
// the sub-blocks the user actually authored (`prior`). The server always
// returns generated mappings, but `mappings` is Optional (not Computed), so
// surfacing a block the user did not configure would trip a "planned null, got
// object" consistency error. Returns nil when the user did not author mappings.
func assignMappingsState(m *pro.CloudLdapMappingsResponse, prior *cloudLdapMappingsModel) *cloudLdapMappingsModel {
	if prior == nil || m == nil {
		return nil
	}
	out := &cloudLdapMappingsModel{}
	if prior.UserMappings != nil && m.UserMappings != nil {
		u := m.UserMappings
		out.UserMappings = &cloudLdapUserMappingsModel{
			ObjectClassLimitation: types.StringValue(u.ObjectClassLimitation),
			ObjectClasses:         types.StringValue(u.ObjectClasses),
			SearchBase:            types.StringValue(u.SearchBase),
			SearchScope:           types.StringValue(u.SearchScope),
			AdditionalSearchBase:  stringPtrValueOrNull(u.AdditionalSearchBase),
			UserID:                types.StringValue(u.UserID),
			Username:              types.StringValue(u.Username),
			RealName:              types.StringValue(u.RealName),
			EmailAddress:          types.StringValue(u.EmailAddress),
			Department:            types.StringValue(u.Department),
			Building:              types.StringValue(u.Building),
			Room:                  types.StringValue(u.Room),
			Phone:                 types.StringValue(u.Phone),
			Position:              types.StringValue(u.Position),
			UserUUID:              types.StringValue(u.UserUUID),
		}
	}
	if prior.GroupMappings != nil && m.GroupMappings != nil {
		g := m.GroupMappings
		out.GroupMappings = &cloudLdapGroupMappingsModel{
			ObjectClassLimitation: types.StringValue(g.ObjectClassLimitation),
			ObjectClasses:         types.StringValue(g.ObjectClasses),
			SearchBase:            types.StringValue(g.SearchBase),
			SearchScope:           types.StringValue(g.SearchScope),
			GroupID:               types.StringValue(g.GroupID),
			GroupName:             types.StringValue(g.GroupName),
			GroupUUID:             types.StringValue(g.GroupUUID),
		}
	}
	if prior.MembershipMappings != nil && m.MembershipMappings != nil {
		out.MembershipMappings = &cloudLdapMembershipMappingsModel{
			GroupMembershipMapping: types.StringValue(m.MembershipMappings.GroupMembershipMapping),
		}
	}
	return out
}

// stringPtrValueOrNull converts a *string into a TF String, mapping nil to
// null. Deliberately NOT helpers.StringPointerValueOrNull: that shared helper
// also collapses a non-nil empty string ("") to null, which would break an
// Optional+Computed field the user explicitly set to "" (plan "" vs state null
// → inconsistent result). Here a non-nil "" is preserved as StringValue("").
func stringPtrValueOrNull(p *string) types.String {
	if p == nil {
		return types.StringNull()
	}
	return types.StringValue(*p)
}
