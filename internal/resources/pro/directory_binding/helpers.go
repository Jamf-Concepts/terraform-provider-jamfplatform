// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package directory_binding

// Wire enum values accepted by the Jamf ProClassic /directorybindings
// endpoint for the top-level `<type>` element. Audit reference:
// local-testing/directorybindings/AUDIT_FINDINGS.md.
//
// Note: the Jamf Pro admin UI labels the Open Directory option as "Apple
// Open Directory" but the wire value is the bare string "Open Directory" —
// the schema documentation calls this out so users do not paste the UI label.
const (
	typeActiveDirectory = "Active Directory"
	typeOpenDirectory   = "Open Directory"
	typePowerBroker     = "PowerBroker Identity Services"
	typeADmitMac        = "ADmitMac"
	typeCentrify        = "Centrify"
)

// allDirectoryBindingTypes lists every accepted `type` wire value. Kept
// alphabetised for diff-stability; used by stringvalidator.OneOf in the
// resource and data source schemas.
var allDirectoryBindingTypes = []string{
	typeADmitMac,
	typeActiveDirectory,
	typeCentrify,
	typeOpenDirectory,
	typePowerBroker,
}

// typePowerBrokerCreateAlias is the legacy product name the Jamf Pro
// classic /directorybindings create endpoint requires for PowerBroker
// bindings. Background: "Likewise" was the product's original name
// before the BeyondTrust acquisition renamed it to "PowerBroker
// Identity Services". The create path validates `type` against the
// legacy name and rejects the modern one with HTTP 409 "Problem with
// directory binding type"; the read path always returns the modern
// name. The provider translates one-way at the input boundary
// (mapType) so TF state stays on the canonical wire value the server
// emits, and users only ever see the modern name.
const typePowerBrokerCreateAlias = "Likewise"

// mapType translates the TF `type` value to the wire value accepted by
// the Jamf Pro classic /directorybindings create / update endpoints.
// Only PowerBroker requires aliasing — every other type passes through
// unchanged. Returns the TF value unchanged when no alias applies.
func mapType(tfType string) string {
	if tfType == typePowerBroker {
		return typePowerBrokerCreateAlias
	}
	return tfType
}
