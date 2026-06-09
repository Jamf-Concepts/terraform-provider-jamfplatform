// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package adcs

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// connector_mode enum values. They map to the wire `outbound` bool:
// INBOUND => outbound:false, OUTBOUND => outbound:true.
const (
	connectorModeInbound  = "INBOUND"
	connectorModeOutbound = "OUTBOUND"
)

// adcsTimeoutAttributeTypes defines the timeout attribute types for the AD CS
// resource operations.
var adcsTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// adcsCertDetailsAttributeTypes is the attribute-type map for the Computed
// *_details nested objects. Both server_certificate_details and
// client_certificate_details share this shape (filename / serial_number /
// subject / issuer / expiration_date — all server-echoed metadata strings).
var adcsCertDetailsAttributeTypes = map[string]attr.Type{
	"filename":        types.StringType,
	"serial_number":   types.StringType,
	"subject":         types.StringType,
	"issuer":          types.StringType,
	"expiration_date": types.StringType,
}
