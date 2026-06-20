// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package location

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// volumePurchasingLocationTimeoutAttributeTypes defines the timeout attribute
// types for the volume purchasing location resource operations. Used by Read
// during import-time refresh to construct a null Timeouts value before state
// has been populated.
var volumePurchasingLocationTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}
