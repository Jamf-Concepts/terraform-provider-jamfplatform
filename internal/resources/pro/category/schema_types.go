// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package category

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// categoryTimeoutAttributeTypes defines the timeout attribute types for the category resource operations.
var categoryTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}
