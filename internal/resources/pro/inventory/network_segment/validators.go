// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package network_segment

import (
	"context"
	"fmt"
	"net"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// ipv4Address validates that a string attribute is a valid IPv4 address.
// Jamf Pro network segments are IPv4-only; IPv6 inputs are rejected even though
// net.ParseIP would accept them.
func ipv4Address() validator.String { return ipv4Validator{} }

type ipv4Validator struct{}

func (ipv4Validator) Description(context.Context) string {
	return "value must be a valid IPv4 address (e.g. 10.0.0.1)."
}

func (ipv4Validator) MarkdownDescription(context.Context) string {
	return "Value must be a valid IPv4 address (e.g. `10.0.0.1`)."
}

func (ipv4Validator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	v := req.ConfigValue.ValueString()
	if ip := net.ParseIP(v); ip == nil || ip.To4() == nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid IPv4 address",
			fmt.Sprintf("Expected a valid IPv4 address, got %q. Jamf Pro network segments are IPv4-only.", v),
		)
	}
}
