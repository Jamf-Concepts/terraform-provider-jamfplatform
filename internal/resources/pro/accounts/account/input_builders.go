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

// optTranslatedString returns a pointer to the wire-translated value of a
// UI-facing enum string, or nil when the value is null/unknown.
func optTranslatedString(m map[string]string, v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	out := translate(m, v.ValueString())
	return &out
}

// buildProUserAccount projects the plan into a Pro v1 UserAccount payload for
// create/update of the base fields. UI enum values are translated to Pro wire
// spellings. password is supplied separately (WriteOnly, read from config) and
// is sent only when non-nil.
func buildProUserAccount(plan AccountResourceModel, password *string) *pro.UserAccount {
	acct := &pro.UserAccount{
		Username:                  helpers.OptionalStringPointer(plan.Username),
		Realname:                  helpers.OptionalStringPointer(plan.FullName),
		Email:                     helpers.OptionalStringPointer(plan.EmailAddress),
		AccessLevel:               optTranslatedString(accessLevelToWire, plan.AccessLevel),
		PrivilegeLevel:            optTranslatedString(privilegeSetToWire, plan.PrivilegeSet),
		AccountStatus:             helpers.OptionalStringPointer(plan.AccessStatus),
		AccountType:               helpers.OptionalStringPointer(plan.AccountType),
		LdapServerID:              helpers.OptionalInt64Pointer(plan.LdapServerID),
		SiteID:                    helpers.OptionalInt64Pointer(plan.SiteID),
		ChangePasswordOnNextLogin: helpers.OptionalBoolPointer(plan.ForcePasswordChange),
	}
	if password != nil && *password != "" {
		acct.PlainPassword = password
	}
	return acct
}

// buildClassicPrivileges projects the Custom privilege grid into a ProClassic
// Account payload carrying only <privileges>. The classic PUT merges, so other
// account fields are intentionally omitted (they are owned by the Pro side).
// Returns nil when there are no privileges to send.
func buildClassicPrivileges(ctx context.Context, plan AccountResourceModel) (*proclassic.Account, diag.Diagnostics) {
	var diags diag.Diagnostics
	if plan.Privileges == nil || plan.Privileges.IsEmpty() {
		return nil, diags
	}
	privMap, d := plan.Privileges.ToMap(ctx)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	grid := accountprivileges.ToAccountPrivileges(privMap)
	if grid == nil {
		return nil, diags
	}
	return &proclassic.Account{Privileges: grid}, diags
}
