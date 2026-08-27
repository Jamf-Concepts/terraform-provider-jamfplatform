// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_gateway

import (
	"strings"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/securitycloud"
)

// The accepted value sets all come straight from the SDK's generated enum
// helpers rather than being restated here, so the validators and the documented
// lists cannot drift from the API the provider was built against.
//
// Validating these at plan time is not cosmetic. Jamf Security Cloud rejects an
// unknown enum value with `400 [INVALID_FIELD] Request body is missing or
// malformed.` — no field name, no offending value. Wire-probed 2026-08-27 with a
// bad cipher and with `strongswan` in the wrong case, which is easy to write and
// impossible to diagnose from that message.

// datacenterValues returns the egress regions a gateway can be deployed to.
func datacenterValues() []string {
	return securitycloud.GatewayCreateRequestDatacenterValues()
}

// keyExchangeValues returns the accepted IKE versions.
func keyExchangeValues() []string {
	return securitycloud.GatewayIpSecRequestKeyExchangeValues()
}

// encryptionValues returns the accepted cipher-suite encryption algorithms.
func encryptionValues() []string {
	return securitycloud.CipherSuiteConfigEncryptionValues()
}

// integrityValues returns the accepted cipher-suite integrity algorithms.
func integrityValues() []string {
	return securitycloud.CipherSuiteConfigIntegrityValues()
}

// dhGroupValues returns the accepted Diffie-Hellman groups.
func dhGroupValues() []string {
	return securitycloud.CipherSuiteConfigDhGroupsValues()
}

// vendorValues returns the accepted remote-peer VPN vendors.
func vendorValues() []string {
	return securitycloud.ConnectionConfigRightRequestVendorValues()
}

// markdownList renders a value set as a comma-separated list of backticked
// literals, so a description and its validator are generated from one slice.
func markdownList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, v := range values {
		quoted = append(quoted, "`"+v+"`")
	}
	return strings.Join(quoted, ", ")
}
