// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package validators

import (
	"context"
	"net"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// ipv4CIDRValidator checks that a string attribute holds an IPv4 CIDR block.
type ipv4CIDRValidator struct{}

// IPv4CIDR returns a validator.String enforcing IPv4 CIDR notation.
//
// Why it exists: the Jamf Security Cloud ZTNA app endpoint refuses a bare address
// and an IPv6 block with the same message — `400 [INVALID_FIELD] bareIps[]: IP
// address range must be in CIDR format.` — which is accurate for `10.11.12.13` and
// misleading for `2001:db8::/32`, a perfectly well-formed CIDR block that the
// endpoint simply does not accept (both wire-probed 2026-08-30). Catching the shape
// at plan time lets the second case say what is actually wrong.
//
// A prefix length is required: `10.0.0.0` is refused, `10.0.0.0/32` accepted.
// Colons are rejected outright rather than relying on the parse, for the same
// reason IPv4Address rejects them — `::ffff:10.0.0.0/104` parses and its `To4` is
// non-nil, so the parse alone would let an IPv4-mapped IPv6 block through.
//
// Null and unknown values defer to the server, per STYLE_GUIDE §Config-time
// validators.
func IPv4CIDR() validator.String {
	return ipv4CIDRValidator{}
}

// Description returns a plain-text description of the validator.
func (ipv4CIDRValidator) Description(_ context.Context) string {
	return "must be an IPv4 address range in CIDR notation, for example 10.1.2.0/24"
}

// MarkdownDescription returns the markdown description of the validator.
func (v ipv4CIDRValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString implements validator.String.
func (v ipv4CIDRValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if reason := ipv4CIDRProblem(value); reason != "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid IPv4 address range",
			reason+" Got: "+value,
		)
	}
}

// ipv4CIDRProblem returns a sentence describing what is wrong with value, or the
// empty string when it is an acceptable IPv4 CIDR block. Split out from
// ValidateString so the cases can be unit-tested without building a framework
// request for each one.
func ipv4CIDRProblem(value string) string {
	if value == "" {
		return "An address range is required."
	}
	if strings.Contains(value, ":") {
		return "IPv6 address ranges are not accepted; supply an IPv4 range in CIDR notation."
	}
	if !strings.Contains(value, "/") {
		return "An address range must carry a prefix length, as in `10.1.2.0/24`; a bare address is not accepted."
	}
	ip, _, err := net.ParseCIDR(value)
	if err != nil {
		return "An address range must be in CIDR notation, as in `10.1.2.0/24`."
	}
	if ip.To4() == nil {
		return "IPv6 address ranges are not accepted; supply an IPv4 range in CIDR notation."
	}
	return ""
}
