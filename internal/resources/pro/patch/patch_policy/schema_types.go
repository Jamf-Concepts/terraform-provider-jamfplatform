// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_policy

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// patchPolicyTimeoutAttributeTypes defines the timeout attribute types for the
// patch policy resource operations.
var patchPolicyTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// killAppAttrTypes describes the element object type for the computed-only
// kill_apps list. Server-derived from the target_version's patch definition.
var killAppAttrTypes = map[string]attr.Type{
	"kill_app_name":      types.StringType,
	"kill_app_bundle_id": types.StringType,
}
