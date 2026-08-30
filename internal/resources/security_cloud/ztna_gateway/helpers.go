// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_gateway

import (
	"net/http"
	"strings"

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

	codeReferencedByAccessPolicies  = "GATEWAY_REFERENCED_BY_ACCESS_POLICIES"
	codeReferencedByDNSZones        = "GATEWAY_REFERENCED_BY_DNS_ZONES"
	codeReferencedByGroupedGateways = "GATEWAY_REFERENCED_BY_GROUPED_GATEWAYS"
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
// Jamf Security Cloud refuses to delete a gateway that anything still references,
// with a `409`. Terraform will happily plan that destroy when the config dependency
// edge disappears in the same apply that removes the reference, so this is worth
// spelling out rather than passing through. In every case the remedy is the same
// two-apply sequence: drop the reference, apply, then destroy the gateway.
//
// All three referrers name themselves. Wire-probed against production EU on
// 2026-08-30 by creating a dedicated gateway, pointing each referrer at it in turn
// and deleting it: an access policy answers GATEWAY_REFERENCED_BY_ACCESS_POLICIES, a
// custom DNS zone name server GATEWAY_REFERENCED_BY_DNS_ZONES, and grouped-gateway
// membership GATEWAY_REFERENCED_BY_GROUPED_GATEWAYS, each with a description naming
// the referrer class and the fix. Releasing the reference and repeating the delete
// answered 204 in all three, so the reference is the sole cause. That is why each
// code gets its own diagnostic naming the one thing to go and look at, rather than
// the operator being handed all three possibilities.
//
// The generic fallback stays, and is not dead code. The 2026-08-27 probe of the zone
// and grouped-gateway cases recorded a bare 409 carrying no structured detail, which
// today's probe contradicts — whether that was a misread then or a server change
// since, an endpoint that has been seen to answer both shapes gets code-keyed
// diagnostics for the shape that carries a code and the three-way explanation for the
// shape that does not.
//
// A ZTNA access policy is a reference Terraform manages: the application's
// `routing.gateway_id` and `routing_overrides[].routing.gateway_id` each name a
// gateway, so moving an application between gateways and destroying the old one in a
// single apply is exactly the ordering trap above. Only an access policy created
// outside Terraform is beyond its reach, and then the reference does live in the
// admin UI.
//
// None of the three codes is an SDK constant: the generated `ApiErrorItemCode` enum
// is declared from the DNS schema, and these appear in the ZTNA spec only as response
// examples, which the generator does not emit. enum_literals_test.go exempts each by
// name so an SDK release that starts generating one fails the guard.
func appendDeleteDiagnostics(diags *diag.Diagnostics, err error) bool {
	apiErr := jamfplatform.AsAPIError(err)
	if apiErr == nil || !apiErr.HasStatus(http.StatusConflict) {
		return false
	}
	for _, detail := range apiErr.Details() {
		summary, referrer, ok := referencedByDetail(detail.Code)
		if !ok {
			continue
		}
		diags.AddError(
			summary,
			"Jamf Security Cloud will not delete a gateway while "+referrer+" still points at it. Remove the "+
				"reference first in a separate apply, then destroy the gateway: dropping the reference and the "+
				"gateway in one apply lets Terraform sequence the destroy before the update that would have "+
				"released it."+reportedDetails(apiErr),
		)
		return true
	}
	diags.AddError(
		"Gateway is still referenced",
		"Jamf Security Cloud refuses to delete a gateway that something still points at: a ZTNA access policy, a "+
			"custom DNS zone name server, or membership of a grouped gateway. It did not say which, so check each "+
			"in turn. Access policies created outside Terraform are invisible to it, so check the Jamf Security "+
			"Cloud admin UI if no `jamfplatform_security_cloud_ztna_app`, zone or grouped gateway in your "+
			"configuration names this gateway. For a reference Terraform does manage, remove it first in a "+
			"separate apply, then destroy the gateway: dropping the reference and the gateway in one apply lets "+
			"Terraform sequence the destroy before the update that would have released it."+reportedDetails(apiErr),
	)
	return true
}

// referencedByDetail maps a referenced-by error code to its diagnostic summary and
// the phrase naming what holds the reference, reporting whether the code is one of
// the three.
func referencedByDetail(code string) (summary, referrer string, ok bool) {
	switch code {
	case codeReferencedByAccessPolicies:
		return "Gateway is still referenced by an access policy application",
			"an access policy application — a `jamfplatform_security_cloud_ztna_app` whose " +
				"`routing.gateway_id` or `routing_overrides[].routing.gateway_id` names this gateway, or one " +
				"created outside Terraform, which appears only in the admin UI —", true
	case codeReferencedByDNSZones:
		return "Gateway is still referenced by a custom DNS zone",
			"a custom DNS zone name server — a `jamfplatform_security_cloud_dns_zone` whose " +
				"`name_servers[].gateway_id` names this gateway —", true
	case codeReferencedByGroupedGateways:
		return "Gateway is still a member of a grouped gateway",
			"a grouped gateway — a `jamfplatform_security_cloud_ztna_grouped_gateway` whose `gateway_ids` " +
				"includes this gateway —", true
	}
	return "", "", false
}

// reportedDetails renders whatever structured detail an error carries, for the
// diagnostics that cannot assume one is present.
//
// The delete conflict is the case: the probed referrer cases answered with a bare
// 409, while the spec documents per-referrer codes. Appending what is there costs
// nothing when there is nothing, and stops the diagnostic contradicting the body
// if the endpoint starts sending one.
func reportedDetails(apiErr *jamfplatform.APIResponseError) string {
	var b strings.Builder
	for _, detail := range apiErr.Details() {
		if detail.Description == "" {
			continue
		}
		b.WriteString(" Reported by Jamf Security Cloud: ")
		b.WriteString(detail.Description)
	}
	return b.String()
}
