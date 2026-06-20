// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package jamf_teacher_settings

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// jamfTeacherSettingsTimeoutAttributeTypes defines the timeout attribute types.
var jamfTeacherSettingsTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// safelistedAppObjectType is the attr.Type for one safelisted_apps element.
var safelistedAppObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name":      types.StringType,
		"bundle_id": types.StringType,
	},
}
