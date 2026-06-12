// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package accountprivileges

import (
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SchemaBlock returns the Optional `privileges` single-nested attribute for a
// privilege-bearing resource (account / account_group). Each of the seven
// categories is an Optional Set of privilege strings. A category left unset is
// not managed by Terraform (omit-on-write; preserved on read). Privileges are
// only honoured by Jamf Pro when privilege_set = "Custom".
//
// The grid is intentionally plain Optional (not Optional+Computed): server-added
// dependency privileges are reconciled out by IntersectIntoState on read, and
// invalid strings are caught by the plan-time Validator, so there is no need for
// the framework to compute or carry forward server values.
func SchemaBlock() rschema.SingleNestedAttribute {
	attrs := make(map[string]rschema.Attribute, len(Categories))
	for _, c := range Categories {
		attrs[c.AttrName] = rschema.SetAttribute{
			ElementType:         types.StringType,
			Optional:            true,
			Description:         c.Desc,
			MarkdownDescription: c.Desc,
		}
	}
	return rschema.SingleNestedAttribute{
		Optional:            true,
		Attributes:          attrs,
		Description:         "Custom privilege grid. Only applied when privilege_set is \"Custom\". Jamf Pro silently adds dependency privileges and silently ignores unrecognised ones; the provider reconciles server-added extras out of state and validates declared privileges at plan time against the tenant's Administrator catalog.",
		MarkdownDescription: "Custom privilege grid. Only applied when `privilege_set` is `Custom`. Jamf Pro silently adds dependency privileges and silently ignores unrecognised ones; the provider reconciles server-added extras out of state and validates declared privileges at plan time against the tenant's Administrator catalog.",
	}
}
