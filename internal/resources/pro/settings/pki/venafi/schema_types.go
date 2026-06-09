// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package venafi

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// pkiVenafiTimeoutAttributeTypes defines the timeout attribute types for the
// Venafi CA resource operations.
var pkiVenafiTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}
