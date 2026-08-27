// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package validators

import (
	"context"
	"net"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// ipv4AddressValidator checks that a string attribute holds an IPv4 address in
// dotted-quad form.
type ipv4AddressValidator struct{}

// IPv4Address returns a validator.String enforcing dotted-quad IPv4 form.
//
// Extracted at its second consumer per STYLE_GUIDE §Shared abstractions: the Jamf
// Security Cloud custom DNS zone and ZTNA gateway resources both need it, and two
// copies of an address check is two places for the next correction to be missed.
//
// Why it exists at all: the Security Cloud endpoints collapse several distinct
// mistakes into one opaque `400 [INVALID_FIELD] … Invalid field value.` — a
// non-address string, an IPv6 literal, and a dotted quad with leading zeros were
// each wire-probed to that same message on 2026-08-27. Catching the shape here
// names the attribute and says what form is expected.
//
// `net.ParseIP` carries most of the weight, including the leading-zero case, which
// it has rejected since Go 1.17. What it does not cover is the IPv4-mapped IPv6
// literal: `::ffff:203.0.113.53` parses, and its `To4` is non-nil, so the parse
// alone accepts a form no Jamf endpoint takes. Counting dots does not rule it out
// either — it has exactly three. Rejecting any colon does.
//
// Null and unknown values defer to the server, per STYLE_GUIDE §Config-time
// validators.
func IPv4Address() validator.String {
	return ipv4AddressValidator{}
}

// Description returns a plain-text description of the validator.
func (ipv4AddressValidator) Description(_ context.Context) string {
	return "must be an IPv4 address in dotted-quad form, with no leading zeros in any octet"
}

// MarkdownDescription returns the markdown description of the validator.
func (v ipv4AddressValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString implements validator.String.
func (v ipv4AddressValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if !strings.ContainsRune(value, ':') {
		if ip := net.ParseIP(value); ip != nil && ip.To4() != nil {
			return
		}
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid IPv4 address",
		"Expected an IPv4 address in dotted-quad form, with no leading zeros in any octet. Got: "+value,
	)
}
