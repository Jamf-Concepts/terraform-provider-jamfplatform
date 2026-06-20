// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package account

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/accountprivileges"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// assignProBaseFields populates the base (Pro-owned) fields from a Pro v1
// UserAccount response, translating Pro wire enum spellings back to the UI
// values. Privileges are NOT touched here (they come from the classic side).
func assignProBaseFields(state *AccountResourceModel, a *pro.UserAccount) {
	if a == nil {
		return
	}
	if a.ID != nil {
		state.ID = helpers.StringPointerValueOrNull(a.ID)
	}
	if a.Username != nil {
		state.Username = helpers.StringPointerValueOrNull(a.Username)
	}
	state.FullName = helpers.StringPointerValueOrNull(a.Realname)
	state.EmailAddress = helpers.StringPointerValueOrNull(a.Email)
	if a.AccessLevel != nil {
		state.AccessLevel = types.StringValue(translate(accessLevelFromWire, *a.AccessLevel))
	}
	if a.PrivilegeLevel != nil {
		state.PrivilegeSet = types.StringValue(translate(privilegeSetFromWire, *a.PrivilegeLevel))
	}
	state.AccessStatus = helpers.StringPointerValueOrNull(a.AccountStatus)
	state.AccountType = helpers.StringPointerValueOrNull(a.AccountType)
	state.LdapServerID = int64OrNull(a.LdapServerID)
	state.SiteID = int64OrNull(a.SiteID)
	state.ForcePasswordChange = helpers.BoolPointerValueOrNull(a.ChangePasswordOnNextLogin)
}

// assignClassicPrivileges reconciles the Custom privilege grid from a ProClassic
// Account response using intersect-on-read (same semantics as account_group):
// declared ∩ server per managed category, null categories stay null, import
// materialises the full grid.
func assignClassicPrivileges(ctx context.Context, state *AccountResourceModel, a *proclassic.Account, isImport bool) diag.Diagnostics {
	var diags diag.Diagnostics
	var serverPriv map[string][]string
	if a != nil {
		serverPriv = accountprivileges.FromAccountPrivileges(a.Privileges)
	}
	switch {
	case isImport:
		out, d := accountprivileges.IntersectIntoState(ctx, nil, serverPriv)
		diags.Append(d...)
		if out.IsEmpty() {
			state.Privileges = nil
		} else {
			state.Privileges = &out
		}
	case state.Privileges != nil:
		out, d := accountprivileges.IntersectIntoState(ctx, state.Privileges, serverPriv)
		diags.Append(d...)
		state.Privileges = &out
	}
	return diags
}

// int64OrNull converts a *int to a types.Int64 (null when nil).
func int64OrNull(p *int) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*p))
}
