// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_grouped_gateway

import (
	"net/http"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/securitycloud"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

// Machine-readable error codes the grouped-gateway endpoints return. Wire-probed
// against production EU on 2026-08-27, except codeGatewayNotFound, which is
// taken from the spec's shared 422 catalogue (SDK spec v1807).
//
// Codes the SDK does not carry as constants. Only the DNS namespace declares its
// error codes in a schema enum, so `securitycloud.ApiErrorItemCode*` covers that
// vocabulary and nothing else — the ZTNA codes below appear in the spec only as
// response examples, which the generator does not emit. Referenced from the SDK
// wherever it has the constant, and declared here where it does not.
const (
	codeGatewayNotFound = securitycloud.ApiErrorItemCodeGatewayNotFound
	codeNotEntitled     = securitycloud.ApiErrorItemCodeNotEntitled

	codeMixedTunnelTypes    = "MIXED_TUNNEL_TYPES"
	codeSharedGatewayMember = "SHARED_GATEWAY_MEMBER"
	codeBadRequest          = "BAD_REQUEST"
)

// appendWriteDiagnostics turns a create/update failure into the most specific
// diagnostic the error body supports, and reports whether it recognised one.
//
// The membership codes are the ones worth translating, because each describes a
// property of the *members* rather than of the group being written, and none of
// the messages says which member is at fault.
//
// `GATEWAY_NOT_FOUND` is the same ordering trap the DNS zone has for its name
// servers: a member gateway must exist before the group can name it, and
// Terraform will plan the group's create alongside the member's when the
// dependency edge is only implicit. It also covers a gateway that exists but
// belongs to another customer, which the spec folds into the same code, so the
// diagnostic names both readings rather than asserting the id is absent.
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
		case codeGatewayNotFound:
			diags.AddAttributeError(
				path.Root("gateway_ids"),
				"Member gateway not found",
				"One of the IDs in `gateway_ids` does not name a gateway this account can reach — either it does "+
					"not exist, or it belongs to another customer. Every member must exist before the group can "+
					"reference it, so if the member is managed in the same configuration, make the dependency "+
					"explicit with `depends_on` or by referencing its `id` attribute. Reported by Jamf Security "+
					"Cloud: "+detail.Description,
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
