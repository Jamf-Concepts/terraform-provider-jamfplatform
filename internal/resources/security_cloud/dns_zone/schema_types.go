// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package dns_zone

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// dnsZoneTimeoutAttributeTypes defines the timeout attribute types for the DNS
// zone resource operations.
var dnsZoneTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// nameServerAttributeTypes defines the object attribute types for one
// authoritative name server entry.
var nameServerAttributeTypes = map[string]attr.Type{
	"ip_address": types.StringType,
	"gateway_id": types.StringType,
}

// nameServerObjectType is the element type of the authoritative_name_servers
// collection.
var nameServerObjectType = types.ObjectType{AttrTypes: nameServerAttributeTypes}
