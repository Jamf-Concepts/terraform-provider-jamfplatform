// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package account_group

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// accountGroupTimeoutAttributeTypes defines the timeout attribute types for the
// account group resource operations.
var accountGroupTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// accessLevelValues are the classic access_level enum values for a group
// (groups have no "Group Access"). UI: "Access Level".
var accessLevelValues = []string{"Full Access", "Site Access"}

// privilegeSetValues are the classic privilege_set enum values. UI: "Privilege Set".
var privilegeSetValues = []string{"Administrator", "Auditor", "Enrollment Only", "Custom"}
