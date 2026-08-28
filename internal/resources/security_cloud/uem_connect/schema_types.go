// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package uem_connect

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// uemConnectTimeoutAttributeTypes defines the timeout attribute types for the UEM
// Connect resource operations.
var uemConnectTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// emailMappingAttributeTypes defines the object attribute types for the email
// derivation rule.
var emailMappingAttributeTypes = map[string]attr.Type{
	"source":                types.StringType,
	"prefix":                types.StringType,
	"suffix":                types.StringType,
	"only_if_email_missing": types.BoolType,
}

// dataFieldMappingAttributeTypes defines the object attribute types for the
// device field mapping block, as the data source returns it.
var dataFieldMappingAttributeTypes = map[string]attr.Type{
	"device_name":  types.StringType,
	"user_name":    types.StringType,
	"user_id":      types.StringType,
	"phone_number": types.StringType,
	"email":        types.ObjectType{AttrTypes: emailMappingAttributeTypes},
}

// groupMappingEntryAttributeTypes defines the object attribute types for one
// group mapping entry.
var groupMappingEntryAttributeTypes = map[string]attr.Type{
	"uem_group_id":            types.StringType,
	"security_cloud_group_id": types.StringType,
}

// groupMappingEntryObjectType is the element type of the group mappings
// collection.
var groupMappingEntryObjectType = types.ObjectType{AttrTypes: groupMappingEntryAttributeTypes}

// groupMappingAttributeTypes defines the object attribute types for the group
// mapping block, as the data source returns it.
var groupMappingAttributeTypes = map[string]attr.Type{
	"enabled":                         types.BoolType,
	"default_security_cloud_group_id": types.StringType,
	"mappings":                        types.ListType{ElemType: groupMappingEntryObjectType},
}

// latestSyncAttributeTypes defines the object attribute types for the most recent
// sync summary, which the data source exposes and the resource does not.
var latestSyncAttributeTypes = map[string]attr.Type{
	"transaction_id":    types.StringType,
	"status":            types.StringType,
	"trigger":           types.StringType,
	"started":           types.StringType,
	"finished":          types.StringType,
	"error_reason":      types.StringType,
	"error_description": types.StringType,
}
