// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package department

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// departmentTimeoutAttributeTypes defines the timeout attribute types for the department resource operations.
var departmentTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}
