// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package activation_profile

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/securitycloud"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// Machine-readable error codes Jamf Security Cloud returns on the activation
// profile endpoints, wire-probed against the EU gateway on 2026-09-01.
//
// Each code was checked individually against the SDK's generated ApiErrorItemCode
// enum — which is the Security Cloud DNS namespace's error schema rather than a
// namespace-wide one — rather than reasoning about the set, per STYLE_GUIDE §Enum
// values and error codes come from the SDK. NOT_ENTITLED is in it. STATE_CONFLICT
// is genuinely absent and is declared as an exemption in enum_literals_test.go,
// where a later SDK release that adds it fails the test. INVALID_FIELD is in the
// enum but is deliberately not referenced here — see appendWriteDiagnostics.
const (
	codeNotEntitled = securitycloud.ApiErrorItemCodeNotEntitled

	// codeStateConflict is returned by pause and resume against a code that has
	// already been deleted. The SDK carries no constant for it.
	codeStateConflict = "STATE_CONFLICT"
)

// appendWriteDiagnostics turns a create or state-assertion failure into the most
// specific diagnostic the error body supports, and reports whether it recognised
// one.
//
// Two codes are worth translating. NOT_ENTITLED is the Security Cloud staple: the
// credentials are valid and the tenant simply does not hold the surface, which a
// bare 403 hides. STATE_CONFLICT is specific to this construct and names a
// situation Terraform cannot see for itself — the profile was deleted out of
// band, which the read surface reports as a healthy profile because deletion here
// is a soft delete that GET does not reflect.
//
// INVALID_FIELD is deliberately not translated. Every field constraint this
// surface enforces is checked at plan time, so an INVALID_FIELD reaching apply
// means the server enforces something the schema does not yet model, and the
// server's own field-attributed message is more informative than anything this
// function could add.
func appendWriteDiagnostics(diags *diag.Diagnostics, err error) bool {
	apiErr := jamfplatform.AsAPIError(err)
	if apiErr == nil {
		return false
	}
	matched := false
	for _, detail := range apiErr.Details() {
		switch detail.Code {
		case codeNotEntitled:
			diags.AddError(
				"Jamf Security Cloud activation profiles not available for this tenant",
				"The credentials are valid, but this tenant is not entitled to Jamf Security Cloud activation "+
					"profiles. Confirm the tenant holds Jamf Security Cloud and that the API integration carries "+
					"the activation profile privileges. Reported by Jamf Security Cloud: "+detail.Description,
			)
		case codeStateConflict:
			diags.AddError(
				"Activation profile has already been deleted",
				"Jamf Security Cloud reports this activation profile as deleted. Deleting an activation profile is "+
					"a soft delete that reads back as a healthy profile, so Terraform cannot detect it during a "+
					"refresh — the profile was most likely deleted outside Terraform. Remove it from state with "+
					"`terraform state rm` and apply again to mint a replacement. Reported by Jamf Security Cloud: "+
					detail.Description,
			)
		default:
			continue
		}
		matched = true
	}
	return matched
}
