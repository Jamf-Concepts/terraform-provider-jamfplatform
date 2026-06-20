// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package jamf_parent_settings

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// jamfParentSettingsTimeoutAttributeTypes defines the timeout attribute types.
var jamfParentSettingsTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// restrictedTimeObjectType is the attr.Type for one restricted_times element.
var restrictedTimeObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"begin_time": types.StringType,
		"end_time":   types.StringType,
	},
}

// safelistedAppObjectType is the attr.Type for one safelisted_apps element.
var safelistedAppObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name":      types.StringType,
		"bundle_id": types.StringType,
	},
}
