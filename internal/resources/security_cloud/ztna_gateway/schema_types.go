// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_gateway

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// gatewayTimeoutAttributeTypes defines the timeout attribute types for the
// gateway resource operations.
var gatewayTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// contactAttributeTypes defines the object attribute types for the operational
// contact.
var contactAttributeTypes = map[string]attr.Type{
	"name":  types.StringType,
	"email": types.StringType,
}

// statusAttributeTypes defines the object attribute types for the read-only
// status block.
var statusAttributeTypes = map[string]attr.Type{
	"state":        types.StringType,
	"tunnel_state": types.StringType,
}

// cipherSuiteAttributeTypes defines the object attribute types for one
// cipher-suite phase.
var cipherSuiteAttributeTypes = map[string]attr.Type{
	"encryption":           types.StringType,
	"integrity":            types.StringType,
	"diffie_hellman_group": types.StringType,
	"sa_lifetime_seconds":  types.Int64Type,
}

// dsJamfSideAttributeTypes defines the data-source-side object attribute types
// for the Jamf-side endpoint. It carries no secret or rotation trigger: the
// pre-shared key is never returned, so a data source has nothing to report.
var dsJamfSideAttributeTypes = map[string]attr.Type{
	"host":          types.StringType,
	"ike_domain_id": types.StringType,
	"subnet":        types.StringType,
	"auth_method":   types.StringType,
}

// dsCustomerSideAttributeTypes defines the data-source-side object attribute
// types for the remote-peer endpoint.
var dsCustomerSideAttributeTypes = map[string]attr.Type{
	"host":          types.StringType,
	"ike_domain_id": types.StringType,
	"subnets":       types.ListType{ElemType: types.StringType},
	"vendor":        types.StringType,
	"auth_method":   types.StringType,
}

// dsIPSecAttributeTypes defines the data-source-side object attribute types for
// the IPsec block.
var dsIPSecAttributeTypes = map[string]attr.Type{
	"key_exchange_protocol": types.StringType,
	"phase_1":               types.ObjectType{AttrTypes: cipherSuiteAttributeTypes},
	"phase_2":               types.ObjectType{AttrTypes: cipherSuiteAttributeTypes},
	"jamf_side":             types.ObjectType{AttrTypes: dsJamfSideAttributeTypes},
	"customer_side":         types.ObjectType{AttrTypes: dsCustomerSideAttributeTypes},
}
