// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_gateway

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestPrivateCIDR pins the Jamf-side subnet rules the server stated in its own
// rejection message (wire-probed 2026-08-27), plus the shape rules the shared CIDR
// grammar contributes. `::ffff:10.0.0.0/104` is the case that matters most: it parses,
// and its `To4` is non-nil, so the gateway's own former parse let it through to be
// reported as a `/104` prefix out of range — true, but for the wrong reason.
func TestPrivateCIDR(t *testing.T) {
	cases := []struct {
		name    string
		value   types.String
		wantErr bool
	}{
		{"ten eight", types.StringValue("10.0.0.0/8"), false},
		{"ten thirty", types.StringValue("10.1.2.0/30"), false},
		{"one seventy two twelve", types.StringValue("172.16.0.0/12"), false},
		{"one ninety two sixteen", types.StringValue("192.168.0.0/16"), false},
		{"one ninety two twenty four", types.StringValue("192.168.100.0/24"), false},
		{"prefix too long", types.StringValue("10.1.2.0/31"), true},
		{"host bits set", types.StringValue("172.16.0.0/11"), true},
		{"prefix too short for 192", types.StringValue("192.168.0.0/15"), true},
		{"public range", types.StringValue("8.8.8.0/24"), true},
		{"default route", types.StringValue("0.0.0.0/0"), true},
		{"bare address", types.StringValue("10.0.0.1"), true},
		{"ipv6", types.StringValue("fd00::/8"), true},
		{"ipv4-mapped ipv6 block", types.StringValue("::ffff:10.0.0.0/104"), true},
		{"nonsense", types.StringValue("not-a-subnet"), true},
		{"null defers to server", types.StringNull(), false},
		{"unknown defers to server", types.StringUnknown(), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("ipsec").AtName("jamf_side").AtName("subnet"),
				ConfigValue: tc.value,
			}
			var resp validator.StringResponse
			privateCIDR().ValidateString(context.Background(), req, &resp)

			if got := resp.Diagnostics.HasError(); got != tc.wantErr {
				t.Errorf("HasError() = %v, want %v (diagnostics: %v)", got, tc.wantErr, resp.Diagnostics)
			}
		})
	}
}

// TestCIDRBlock pins the customer-side subnet grammar, which is the shared
// commonvalidators.IPv4CIDR: no IPv4-mapped IPv6 block, and no range carrying host
// bits, since `subnets` is Required and Terraform holds the provider to the value it
// planned.
func TestCIDRBlock(t *testing.T) {
	cases := []struct {
		name    string
		value   types.String
		wantErr bool
	}{
		{"default route is allowed on the customer side", types.StringValue("0.0.0.0/0"), false},
		{"public range is allowed on the customer side", types.StringValue("203.0.113.0/24"), false},
		{"private range", types.StringValue("10.10.0.0/16"), false},
		{"bare address", types.StringValue("10.10.0.1"), true},
		{"ipv6", types.StringValue("2001:db8::/32"), true},
		{"ipv4-mapped ipv6 block", types.StringValue("::ffff:10.0.0.0/104"), true},
		{"host bits set is accepted; canonicalisation is unprobed on this endpoint", types.StringValue("10.10.0.1/16"), false},
		{"nonsense", types.StringValue("nope"), true},
		{"null defers to server", types.StringNull(), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("ipsec").AtName("customer_side").AtName("subnets"),
				ConfigValue: tc.value,
			}
			var resp validator.StringResponse
			cidrBlock().ValidateString(context.Background(), req, &resp)

			if got := resp.Diagnostics.HasError(); got != tc.wantErr {
				t.Errorf("HasError() = %v, want %v (diagnostics: %v)", got, tc.wantErr, resp.Diagnostics)
			}
		})
	}
}

func TestValidatorDescriptions(t *testing.T) {
	for name, v := range map[string]validator.String{
		"privateCIDR": privateCIDR(),
		"cidrBlock":   cidrBlock(),
	} {
		if v.Description(context.Background()) == "" {
			t.Errorf("%s must describe itself", name)
		}
	}
}
