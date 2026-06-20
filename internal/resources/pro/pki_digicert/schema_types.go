// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pki_digicert

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// digicertTimeoutAttributeTypes defines the timeout attribute types for the
// resource operations.
var digicertTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// clientCertificateDetailsAttrTypes defines the attribute types for the Computed
// client_certificate_details object (server-echo certificate metadata). Modelled
// as a types.Object — a Computed nested object backed by a Go struct trips an
// Unknown-at-plan conversion error under acceptance apply.
var clientCertificateDetailsAttrTypes = map[string]attr.Type{
	"filename":        types.StringType,
	"serial_number":   types.StringType,
	"subject":         types.StringType,
	"issuer":          types.StringType,
	"expiration_date": types.StringType,
}
