// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_app

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ztnaAppTimeoutAttributeTypes defines the timeout attribute types for the ZTNA
// app resource operations.
var ztnaAppTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// routingAttributeTypes defines the object attribute types for a routing block.
// One table serves both the app's default routing and each per-group override,
// because the wire sends the same object in both positions.
var routingAttributeTypes = map[string]attr.Type{
	"mode":         types.StringType,
	"gateway_id":   types.StringType,
	"routing_mode": types.StringType,
}

// routingObjectType is the type of a routing block.
var routingObjectType = types.ObjectType{AttrTypes: routingAttributeTypes}

// routingOverrideAttributeTypes defines the object attribute types for one
// per-group routing override.
var routingOverrideAttributeTypes = map[string]attr.Type{
	"device_group_ids": types.SetType{ElemType: types.StringType},
	"routing":          routingObjectType,
}

// routingOverrideObjectType is the element type of the routing_overrides
// collection on the resource.
var routingOverrideObjectType = types.ObjectType{AttrTypes: routingOverrideAttributeTypes}

// dsRoutingOverrideAttributeTypes mirrors routingOverrideAttributeTypes for the
// data sources, where every collection returning API data is a list.
var dsRoutingOverrideAttributeTypes = map[string]attr.Type{
	"device_group_ids": types.ListType{ElemType: types.StringType},
	"routing":          routingObjectType,
}

// dsRoutingOverrideObjectType is the element type of the data-source-side
// routing_overrides list.
var dsRoutingOverrideObjectType = types.ObjectType{AttrTypes: dsRoutingOverrideAttributeTypes}

// securityControlAttributeTypes defines the object attribute types for the two
// security cards that carry only a toggle and its notification setting.
var securityControlAttributeTypes = map[string]attr.Type{
	"enabled":                   types.BoolType,
	"device_push_notifications": types.BoolType,
}

// securityControlObjectType is the type of a plain security control block.
var securityControlObjectType = types.ObjectType{AttrTypes: securityControlAttributeTypes}

// deviceRiskAttributeTypes defines the object attribute types for the device risk
// card, which adds the risk level the denial starts at.
var deviceRiskAttributeTypes = map[string]attr.Type{
	"enabled":                   types.BoolType,
	"deny_at_risk_level":        types.StringType,
	"device_push_notifications": types.BoolType,
}

// deviceRiskObjectType is the type of the device risk control block.
var deviceRiskObjectType = types.ObjectType{AttrTypes: deviceRiskAttributeTypes}

// securityAttributeTypes defines the object attribute types for the security
// block, one member per card in the admin UI's Security tab.
var securityAttributeTypes = map[string]attr.Type{
	"managed_device": securityControlObjectType,
	"device_risk":    deviceRiskObjectType,
	"jamf_trust":     securityControlObjectType,
}
