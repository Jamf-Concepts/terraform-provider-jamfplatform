// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package dns_hostname_mappings

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// hostnameMappingsTimeoutAttributeTypes defines the timeout attribute types for the
// hostname mappings resource operations.
var hostnameMappingsTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// mappingAttributeTypes defines the object attribute types for one hostname mapping
// in the resource schema.
var mappingAttributeTypes = map[string]attr.Type{
	"hostname":              types.StringType,
	"ipv4_addresses":        types.SetType{ElemType: types.StringType},
	"ipv6_addresses":        types.SetType{ElemType: types.StringType},
	"connect_to_ztna":       types.BoolType,
	"connect_to_secure_dns": types.BoolType,
}

// mappingObjectType is the element type of the mappings collection.
var mappingObjectType = types.ObjectType{AttrTypes: mappingAttributeTypes}
