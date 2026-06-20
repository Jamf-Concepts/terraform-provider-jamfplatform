// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_enrollment_profile

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var enrollmentProfileTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// attachmentAttrTypes is the object shape for the Computed attachments list.
var attachmentAttrTypes = map[string]attr.Type{
	"id":       types.StringType,
	"filename": types.StringType,
	"uri":      types.StringType,
}

// attachmentObjectType is the element type of the attachments list.
var attachmentObjectType = types.ObjectType{AttrTypes: attachmentAttrTypes}
