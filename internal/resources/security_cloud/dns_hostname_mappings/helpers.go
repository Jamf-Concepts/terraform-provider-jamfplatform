// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package dns_hostname_mappings

import (
	"net/http"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/securitycloud"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

// Machine-readable error codes Jamf Security Cloud returns on the custom hostname
// mappings endpoint. Wire-probed against production EU on 2026-08-29. All three are
// in the SDK's generated ApiErrorItemCode enum, which the DNS namespace declares in
// its spec schema, so all three are taken from there rather than restated as
// literals — checked individually rather than reasoned about as a set, per
// STYLE_GUIDE §Enum values and error codes come from the SDK.
const (
	codeInvalidField     = securitycloud.ApiErrorItemCodeInvalidField
	codeListSizeExceeded = securitycloud.ApiErrorItemCodeListSizeExceeded
	codeNotEntitled      = securitycloud.ApiErrorItemCodeNotEntitled
)

// appendWriteDiagnostics turns a write failure into the most specific diagnostic the
// error body supports, and reports whether it recognised one.
//
// Every code here should have been prevented at plan time, so reaching apply means
// the provider's rules and the server's have diverged. Naming the attribute anyway is
// what makes that diagnosable: the raw bodies are a bare "Invalid field value." with
// a null field, and a size violation that sometimes names an indexed field and
// sometimes names nothing.
//
// The one failure this cannot help with is a duplicate host name across two mappings,
// which the endpoint answers with a 500 carrying an empty errors array. There is no
// code to match and nothing in the body to quote, which is precisely why the
// uniqueness check is a plan-time validator rather than a translated diagnostic.
func appendWriteDiagnostics(diags *diag.Diagnostics, err error) bool {
	apiErr := jamfplatform.AsAPIError(err)
	if apiErr == nil {
		return false
	}
	matched := false
	for _, detail := range apiErr.Details() {
		switch detail.Code {
		case codeInvalidField:
			diags.AddAttributeError(
				path.Root("mappings"),
				"Hostname mapping not accepted",
				"Jamf Security Cloud refuses one of these mappings. Check that each host name is a valid name "+
					"with no wildcard, that every `ipv4_addresses` entry is an IPv4 address and every "+
					"`ipv6_addresses` entry an IPv6 address, and that each mapping sets at least one of the two. "+
					"Reported by Jamf Security Cloud: "+detail.Description,
			)
		case codeListSizeExceeded:
			diags.AddAttributeError(
				path.Root("mappings"),
				"Hostname mapping collection size out of range",
				"A collection here is outside the size Jamf Security Cloud accepts: `mappings` takes 1 to 500 "+
					"entries, and each mapping takes at most 10 `ipv4_addresses` and 10 `ipv6_addresses`. "+
					"Reported by Jamf Security Cloud: "+detail.Description,
			)
		case codeNotEntitled:
			diags.AddError(
				"Tenant not entitled to Jamf Security Cloud custom DNS",
				"The credentials authenticated successfully but this tenant does not have the custom DNS surface "+
					"enabled. Contact Jamf to have it provisioned. Reported by Jamf Security Cloud: "+
					detail.Description,
			)
		default:
			continue
		}
		matched = true
	}
	return matched
}

// appendDuplicateHostnameHint adds guidance when a write fails with a bare 500 and no
// error details.
//
// A duplicate host name across two mappings is the one cause wire-probing found for
// that response, and the plan-time uniqueness validator should have caught it — so
// reaching here means either the validator missed a case or the endpoint has a second
// way of failing this way. Saying both is more use than surfacing the bare status.
func appendDuplicateHostnameHint(diags *diag.Diagnostics, err error) bool {
	apiErr := jamfplatform.AsAPIError(err)
	if apiErr == nil || !apiErr.HasStatus(http.StatusInternalServerError) || len(apiErr.Details()) > 0 {
		return false
	}
	diags.AddAttributeError(
		path.Root("mappings"),
		"Jamf Security Cloud rejected the hostname mappings without saying why",
		"The write failed with an internal server error carrying no detail. The known cause is the same host "+
			"name appearing in more than one mapping, which this provider normally catches before the write — so "+
			"check for a repeated `hostname`, and if there is none, this is worth reporting to Jamf. Underlying "+
			"error: "+err.Error(),
	)
	return true
}
