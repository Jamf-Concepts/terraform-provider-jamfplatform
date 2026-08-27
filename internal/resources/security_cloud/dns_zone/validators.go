// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package dns_zone

import (
	"context"
	"net"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// ipv4AddressValidator checks that a string attribute holds an IPv4 address in
// dotted-quad form.
type ipv4AddressValidator struct{}

// ipv4Address returns a validator.String enforcing dotted-quad IPv4 form.
//
// Why plan time: Jamf Security Cloud refuses anything else with
// `400 [INVALID_FIELD] nameServers[<n>].ip: Invalid field value.`, which names
// the offending element but not what is wrong with it. Wire-probed against
// production EU on 2026-08-27, three separate inputs collapse into that one
// message — a non-address string, an IPv6 address, and a dotted quad with
// leading zeros (`203.000.113.053`). Checking here turns all three into a
// diagnostic that points at the attribute and says which form is expected.
//
// Reserved-range rejection is deliberately NOT checked. A private or loopback
// address is refused separately, with `422 NAMESERVER_IP_RESTRICTED`, but the
// full restricted set is not published — guessing it would risk failing a plan
// over an address Jamf Security Cloud would have accepted, which is worse than
// letting the server answer. Null and unknown values defer to the server too.
func ipv4Address() validator.String {
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
func (v ipv4AddressValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if isDottedQuadIPv4(value) {
		return
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid name server IP address",
		"Jamf Security Cloud accepts only IPv4 name server addresses in dotted-quad form, with no leading zeros "+
			"in any octet. Got: "+value,
	)
}

// isDottedQuadIPv4 reports whether value is an IPv4 address written as four
// decimal octets. net.ParseIP accepts an IPv4-mapped IPv6 literal and, in older
// Go releases, octets with leading zeros; both are refused by Jamf Security
// Cloud, so the octets are checked directly rather than trusting the parse.
func isDottedQuadIPv4(value string) bool {
	octets := strings.Split(value, ".")
	if len(octets) != 4 {
		return false
	}
	for _, octet := range octets {
		if len(octet) == 0 || len(octet) > 3 {
			return false
		}
		if len(octet) > 1 && octet[0] == '0' {
			return false
		}
		for i := 0; i < len(octet); i++ {
			if octet[i] < '0' || octet[i] > '9' {
				return false
			}
		}
	}
	ip := net.ParseIP(value)
	return ip != nil && ip.To4() != nil
}
