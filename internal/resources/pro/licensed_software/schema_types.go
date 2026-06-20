// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package licensed_software

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// licensedSoftwareTimeoutAttributeTypes defines the timeout attribute types for
// the licensed_software resource operations.
var licensedSoftwareTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// computerObjectType / attachmentObjectType are the element object types for the
// Computed-only `computers` and `licenses[].attachments` collections. These are
// modeled as types.List (not Go typed slices) because a Computed attribute is
// Unknown at plan time, and a typed slice cannot carry an unknown value.
var computerObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"id":   types.StringType,
	"name": types.StringType,
	"udid": types.StringType,
}}

var attachmentObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"id":       types.StringType,
	"filename": types.StringType,
	"uri":      types.StringType,
}}
