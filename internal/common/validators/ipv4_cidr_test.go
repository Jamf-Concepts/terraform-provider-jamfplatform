// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package validators

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestIPv4CIDR pins the accepted form to what the ZTNA app endpoint answered on
// 2026-08-30: `10.20.30.0/24` was accepted (201), while `10.11.12.13` and
// `2001:db8::/32` were both refused (400, `field: bareIps[]`). The remaining cases
// are shapes the same parse rules settle.
func TestIPv4CIDR(t *testing.T) {
	cases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"class A block", "10.20.30.0/24", false},
		{"single host", "10.0.0.1/32", false},
		{"whole space", "0.0.0.0/0", false},
		{"host bits set", "10.1.2.3/24", false},

		{"bare address", "10.11.12.13", true},
		{"ipv6 block", "2001:db8::/32", true},
		{"ipv4-mapped ipv6 block", "::ffff:10.0.0.0/104", true},
		{"prefix out of range", "10.0.0.0/33", true},
		{"not an address", "not-a-cidr/24", true},
		{"leading zero octet", "010.0.0.0/8", true},
		{"empty", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			IPv4CIDR().ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("direct_ips_and_subnets"),
				ConfigValue: types.StringValue(tc.value),
			}, resp)
			if got := resp.Diagnostics.HasError(); got != tc.wantErr {
				t.Fatalf("IPv4CIDR(%q) error = %v, want %v (%s)", tc.value, got, tc.wantErr, resp.Diagnostics)
			}
		})
	}
}

// TestIPv4CIDRMessages pins each refusal to a message that names the actual
// mistake, because the endpoint's own message says "must be in CIDR format" for an
// IPv6 block that is already in CIDR format.
func TestIPv4CIDRMessages(t *testing.T) {
	cases := map[string]string{
		"2001:db8::/32": "IPv6",
		"10.11.12.13":   "prefix length",
		"":              "required",
		"10.0.0.0/33":   "CIDR notation",
	}
	for value, want := range cases {
		t.Run(value, func(t *testing.T) {
			got := ipv4CIDRProblem(value)
			if !strings.Contains(got, want) {
				t.Fatalf("ipv4CIDRProblem(%q) = %q, want it to mention %q", value, got, want)
			}
		})
	}
}

// TestIPv4CIDRDefersOnNullAndUnknown pins the config-time validator contract.
func TestIPv4CIDRDefersOnNullAndUnknown(t *testing.T) {
	for name, value := range map[string]types.String{
		"null":    types.StringNull(),
		"unknown": types.StringUnknown(),
	} {
		t.Run(name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			IPv4CIDR().ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("direct_ips_and_subnets"),
				ConfigValue: value,
			}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("expected no error for a %s value, got %s", name, resp.Diagnostics)
			}
		})
	}
}
