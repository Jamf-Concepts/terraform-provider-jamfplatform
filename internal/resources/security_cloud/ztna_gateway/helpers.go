// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_gateway

import (
	"net/http"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

// Machine-readable error codes the ZTNA gateway endpoints return. Wire-probed
// against production EU on 2026-08-27.
const (
	codeGatewayTypeChangeNotSupported = "GATEWAY_TYPE_CHANGE_NOT_SUPPORTED"
	codeIPSecSecretClearNotSupported  = "IPSEC_SECRET_CLEAR_NOT_SUPPORTED"
	codeNotEntitled                   = "NOT_ENTITLED"
	codeBadRequest                    = "BAD_REQUEST"
)

// appendWriteDiagnostics turns a create/update failure into the most specific
// diagnostic the error body supports, and reports whether it recognised one.
//
// The three codes worth translating all share a property: the message names a
// mechanism rather than a fix. `GATEWAY_TYPE_CHANGE_NOT_SUPPORTED` should not
// normally be reachable — the `ipsec` block's plan modifier replaces the gateway
// instead — so seeing it means something got past that, and the diagnostic says
// as much. `BAD_REQUEST` "No mapping found for one of the supplied ids" is
// exclusively about `tenant_ids`, and says nothing about which id or why.
func appendWriteDiagnostics(diags *diag.Diagnostics, err error) bool {
	apiErr := jamfplatform.AsAPIError(err)
	if apiErr == nil {
		return false
	}
	matched := false
	for _, detail := range apiErr.Details() {
		switch detail.Code {
		case codeGatewayTypeChangeNotSupported:
			diags.AddError(
				"Gateway form cannot be changed in place",
				"Jamf Security Cloud does not convert a dedicated IPsec gateway into a dedicated internet gateway "+
					"or back. The gateway has to be replaced. Reported by Jamf Security Cloud: "+detail.Description,
			)
		case codeIPSecSecretClearNotSupported:
			diags.AddAttributeError(
				path.Root("ipsec").AtName("jamf_side").AtName("shared_secret"),
				"IPsec pre-shared key cannot be cleared",
				"The pre-shared key can be rotated but never removed. Supply a new `shared_secret` and bump "+
					"`shared_secret_wo_version`, or leave both alone to keep the stored key. Reported by Jamf "+
					"Security Cloud: "+detail.Description,
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

// appendDeleteDiagnostics explains the one delete failure that is not a mistake
// in the configuration but an ordering problem in the plan.
//
// Jamf Security Cloud refuses to delete a gateway that anything still references —
// a custom DNS zone's name server, or membership of a grouped gateway — with a
// bare `409 CONFLICT` and no structured detail naming the referrer (wire-probed
// 2026-08-27 from both directions). Terraform will happily plan that destroy when
// the config dependency edge disappears in the same apply that removes the
// reference, so this is worth spelling out rather than passing through.
func appendDeleteDiagnostics(diags *diag.Diagnostics, err error) bool {
	apiErr := jamfplatform.AsAPIError(err)
	if apiErr == nil || !apiErr.HasStatus(http.StatusConflict) {
		return false
	}
	diags.AddError(
		"Gateway is still referenced",
		"Jamf Security Cloud refuses to delete a gateway that something still points at — a custom DNS zone name "+
			"server, or membership of a grouped gateway. It does not say which. Remove the reference first, in a "+
			"separate apply, then destroy the gateway: dropping the reference and the gateway in one apply lets "+
			"Terraform sequence the destroy before the update that would have released it.",
	)
	return true
}
