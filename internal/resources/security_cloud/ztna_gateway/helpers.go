// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_gateway

import (
	"net/http"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/securitycloud"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

// Machine-readable error codes the ZTNA gateway endpoints return. Wire-probed
// against production EU on 2026-08-27, except codeDedicatedIPsLimit, which is
// taken from the spec's shared 409 catalogue (SDK spec v1807).
//
// Codes the SDK does not carry as constants. Only the DNS namespace declares its
// error codes in a schema enum, so `securitycloud.ApiErrorItemCode*` covers that
// vocabulary and nothing else — the ZTNA codes below appear in the spec only as
// response examples, which the generator does not emit. Referenced from the SDK
// wherever it has the constant, and declared here where it does not.
const (
	codeNotEntitled = securitycloud.ApiErrorItemCodeNotEntitled

	codeGatewayTypeChangeNotSupported = "GATEWAY_TYPE_CHANGE_NOT_SUPPORTED"
	codeIPSecSecretClearNotSupported  = "IPSEC_SECRET_CLEAR_NOT_SUPPORTED"
	codeDedicatedIPsLimit             = "DEDICATED_IPS_LIMIT"
	codeBadRequest                    = "BAD_REQUEST"
)

// appendWriteDiagnostics turns a create/update failure into the most specific
// diagnostic the error body supports, and reports whether it recognised one.
//
// Three of the four codes worth translating share a property: the message names
// a mechanism rather than a fix. `GATEWAY_TYPE_CHANGE_NOT_SUPPORTED` should not
// normally be reachable — the `ipsec` block's plan modifier replaces the gateway
// instead — so seeing it means something got past that, and the diagnostic says
// as much. `BAD_REQUEST` "No mapping found for one of the supplied ids" is
// exclusively about `tenant_ids`, and says nothing about which id or why.
//
// `DEDICATED_IPS_LIMIT` is the odd one out: nothing in the configuration is
// wrong, the account has simply used every dedicated IP address it is allotted.
// It gets no attribute path because the addresses are provisioned by Jamf and
// surface as the computed `dedicated_egress_ip_addresses` — there is no input to
// point at, and pointing at one would imply an edit that cannot fix it.
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
				path.Root("ipsec").AtName("jamf_side").AtName("authentication_secret"),
				"IPsec pre-shared key cannot be cleared",
				"The pre-shared key can be rotated but never removed. Supply a new `authentication_secret` and "+
					"bump `authentication_secret_wo_version`, or leave both alone to keep the stored key. Reported by Jamf "+
					"Security Cloud: "+detail.Description,
			)
		case codeDedicatedIPsLimit:
			diags.AddError(
				"Dedicated IP address limit reached",
				"This account has no dedicated IP addresses left to assign, so Jamf Security Cloud cannot "+
					"provision another dedicated gateway. Nothing in this configuration is wrong: destroy a "+
					"dedicated gateway you no longer need, or contact Jamf to raise the allotment. Reported by "+
					"Jamf Security Cloud: "+detail.Description,
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
