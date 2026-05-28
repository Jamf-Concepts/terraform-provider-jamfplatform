// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package enrollment_customization

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// enrollmentCustomizationTimeoutAttributeTypes defines the timeout attribute
// types for the resource's CRUD operations. Used by Read during import-time
// refresh to construct a null Timeouts value before state has been populated.
var enrollmentCustomizationTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}
