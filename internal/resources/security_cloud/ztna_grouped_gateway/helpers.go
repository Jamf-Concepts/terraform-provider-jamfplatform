// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_grouped_gateway

import (
	"net/http"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

// Machine-readable error codes the grouped-gateway endpoints return. Wire-probed
// against production EU on 2026-08-27.
const (
	codeMixedTunnelTypes    = "MIXED_TUNNEL_TYPES"
	codeSharedGatewayMember = "SHARED_GATEWAY_MEMBER"
	codeNotEntitled         = "NOT_ENTITLED"
	codeBadRequest          = "BAD_REQUEST"
)

// appendWriteDiagnostics turns a create/update failure into the most specific
// diagnostic the error body supports, and reports whether it recognised one.
//
// The two membership codes are the ones worth translating, because both describe
// a property of the *members* rather than of the group being written, and neither
// message says which member is at fault.
func appendWriteDiagnostics(diags *diag.Diagnostics, err error) bool {
	apiErr := jamfplatform.AsAPIError(err)
	if apiErr == nil {
		return false
	}
	matched := false
	for _, detail := range apiErr.Details() {
		switch detail.Code {
		case codeMixedTunnelTypes:
			diags.AddAttributeError(
				path.Root("gateway_ids"),
				"Member gateways have different forms",
				"Every member of a grouped gateway must be the same form — all dedicated IPsec gateways, or all "+
					"dedicated internet gateways. A gateway is an IPsec gateway when it has an `ipsec` block. "+
					"Reported by Jamf Security Cloud: "+detail.Description,
			)
		case codeSharedGatewayMember:
			diags.AddAttributeError(
				path.Root("gateway_ids"),
				"Shared gateway cannot be a group member",
				"Members must be your own dedicated gateways. One of these IDs names a Jamf-managed shared gateway "+
					"— those cannot be grouped, and are read with "+
					"`jamfplatform_security_cloud_ztna_shared_gateways`. Reported by Jamf Security Cloud: "+
					detail.Description,
			)
		case codeBadRequest:
			diags.AddAttributeError(
				path.Root("tenant_ids"),
				"Unknown tenant ID",
				"Jamf Security Cloud could not resolve one of the supplied tenant IDs. Every tenant must belong to "+
					"the same organization as the credentials the provider is configured with. Reported by Jamf "+
					"Security Cloud: "+detail.Description,
			)
		case codeNotEntitled:
			diags.AddError(
				"Tenant not entitled to Jamf Security Cloud ZTNA",
				"The credentials authenticated successfully but this tenant does not have the ZTNA surface enabled. "+
					"Contact Jamf to have it provisioned. Reported by Jamf Security Cloud: "+detail.Description,
			)
		default:
			continue
		}
		matched = true
	}
	return matched
}

// appendDeleteDiagnostics explains the delete refusal, which is an ordering
// problem rather than a configuration mistake.
//
// A grouped gateway that something still points at — a custom DNS zone's name
// server, say — is refused with a bare `409 CONFLICT` carrying no detail about the
// referrer.
func appendDeleteDiagnostics(diags *diag.Diagnostics, err error) bool {
	apiErr := jamfplatform.AsAPIError(err)
	if apiErr == nil || !apiErr.HasStatus(http.StatusConflict) {
		return false
	}
	diags.AddError(
		"Grouped gateway is still referenced",
		"Jamf Security Cloud refuses to delete a grouped gateway that something still points at, such as a custom "+
			"DNS zone name server. It does not say which. Remove the reference first, in a separate apply, then "+
			"destroy the group: dropping the reference and the group in one apply lets Terraform sequence the "+
			"destroy before the update that would have released it.",
	)
	return true
}
