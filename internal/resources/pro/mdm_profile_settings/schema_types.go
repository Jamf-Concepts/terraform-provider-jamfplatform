// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mdm_profile_settings

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// mdmProfileSettingsTimeoutAttributeTypes defines the timeout attribute types for the resource.
var mdmProfileSettingsTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}
