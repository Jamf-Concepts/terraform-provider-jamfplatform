// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package account_group

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/accountprivileges"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// buildAccountGroupInput converts the Terraform plan into an SDK Group payload.
// The classic /accounts/groupid PUT is a category-level merge: fields and
// privilege categories that are not sent are retained, so null members /
// null privilege categories are omitted (retained) and an explicitly empty
// members set clears membership. Privileges are only emitted when the privilege
// set is Custom (Jamf Pro ignores them otherwise).
func buildAccountGroupInput(ctx context.Context, plan AccountGroupResourceModel) (*proclassic.Group, diag.Diagnostics) {
	var diags diag.Diagnostics

	group := &proclassic.Group{
		Name:         helpers.OptionalStringPointer(plan.DisplayName),
		AccessLevel:  helpers.OptionalStringPointer(plan.AccessLevel),
		PrivilegeSet: helpers.OptionalStringPointer(plan.PrivilegeSet),
	}

	if id := helpers.OptionalInt64Pointer(plan.SiteID); id != nil {
		group.Site = &proclassic.Site{ID: id}
	}
	if id := helpers.OptionalInt64Pointer(plan.LdapServerID); id != nil {
		group.LdapServer = &proclassic.GroupLdapServer{ID: id}
	}

	// Members: null => omit (retain server/directory membership); set (incl.
	// empty) => send, where empty clears.
	if !plan.Members.IsNull() && !plan.Members.IsUnknown() {
		var ids []int64
		diags.Append(plan.Members.ElementsAs(ctx, &ids, false)...)
		if diags.HasError() {
			return nil, diags
		}
		users := make([]proclassic.IDName, 0, len(ids))
		for _, id := range ids {
			id := int(id)
			users = append(users, proclassic.IDName{ID: &id})
		}
		group.Members = &proclassic.GroupMembers{User: &users}
	}

	// Privileges: only when Custom and the block is present with content.
	if plan.PrivilegeSet.ValueString() == proclassic.GroupPrivilegeSetCustom && plan.Privileges != nil && !plan.Privileges.IsEmpty() {
		privMap, d := plan.Privileges.ToMap(ctx)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		group.Privileges = accountprivileges.ToGroupPrivileges(privMap)
	}

	return group, diags
}
