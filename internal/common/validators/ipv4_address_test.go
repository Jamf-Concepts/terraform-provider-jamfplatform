// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package validators

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestIPv4Address covers the forms the Security Cloud endpoints reject with one
// indistinguishable message, plus the ones they accept.
func TestIPv4Address(t *testing.T) {
	cases := []struct {
		name    string
		value   types.String
		wantErr bool
	}{
		{"dotted quad", types.StringValue("203.0.113.53"), false},
		{"public resolver", types.StringValue("8.8.8.8"), false},
		{"zero octet", types.StringValue("10.0.0.53"), false},
		{"all zeros", types.StringValue("0.0.0.0"), false},
		{"broadcast", types.StringValue("255.255.255.255"), false},
		{"leading zeros", types.StringValue("203.000.113.053"), true},
		{"single leading zero", types.StringValue("010.1.1.1"), true},
		{"ipv6", types.StringValue("2001:db8::53"), true},
		{"ipv4-mapped ipv6", types.StringValue("::ffff:203.0.113.53"), true},
		{"cidr", types.StringValue("203.0.113.0/24"), true},
		{"three octets", types.StringValue("203.0.113"), true},
		{"five octets", types.StringValue("203.0.113.53.1"), true},
		{"octet out of range", types.StringValue("203.0.113.256"), true},
		{"trailing dot", types.StringValue("203.0.113.53."), true},
		{"leading space", types.StringValue(" 203.0.113.53"), true},
		{"not an address", types.StringValue("not-an-ip"), true},
		{"empty", types.StringValue(""), true},
		{"null defers to server", types.StringNull(), false},
		{"unknown defers to server", types.StringUnknown(), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("ip_address"),
				ConfigValue: tc.value,
			}
			var resp validator.StringResponse
			IPv4Address().ValidateString(context.Background(), req, &resp)

			if got := resp.Diagnostics.HasError(); got != tc.wantErr {
				t.Errorf("HasError() = %v, want %v (diagnostics: %v)", got, tc.wantErr, resp.Diagnostics)
			}
		})
	}
}

// TestIPv4AddressRejectsMappedIPv6 pins the one case net.ParseIP alone would let
// through: an IPv4-mapped IPv6 literal has a non-nil To4, so the dot count is what
// rules it out. No Jamf endpoint accepts that form.
func TestIPv4AddressRejectsMappedIPv6(t *testing.T) {
	req := validator.StringRequest{
		Path:        path.Root("ip_address"),
		ConfigValue: types.StringValue("::ffff:203.0.113.53"),
	}
	var resp validator.StringResponse
	IPv4Address().ValidateString(context.Background(), req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Error("an IPv4-mapped IPv6 literal must be rejected")
	}
}

func TestIPv4Address_Description(t *testing.T) {
	if IPv4Address().Description(context.Background()) == "" {
		t.Error("validator must describe itself")
	}
}
