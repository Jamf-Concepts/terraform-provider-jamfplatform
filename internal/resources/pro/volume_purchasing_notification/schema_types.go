// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package volume_purchasing_notification

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// volumePurchasingNotificationTimeoutAttributeTypes defines the timeout attribute
// types for the resource operations.
var volumePurchasingNotificationTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// externalRecipientAttrTypes is the object attribute schema for one external
// recipient. Shared by the resource Set and the data source List element types.
var externalRecipientAttrTypes = map[string]attr.Type{
	"email": types.StringType,
	"name":  types.StringType,
}

// externalRecipientObjectType is the element type for the external_recipients
// collection.
var externalRecipientObjectType = types.ObjectType{AttrTypes: externalRecipientAttrTypes}
