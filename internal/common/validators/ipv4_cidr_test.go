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
// `2001:db8::/32` were both refused (400, `field: bareIps[]`). A range carrying host
// bits is refused here despite being accepted on the wire, because the same probe run
// showed the endpoint rewriting it — `10.99.43.7/24` was stored as `10.99.43.0/24`
// and `10.99.43.7/0` as `0.0.0.0/0`, both after a `204` — which fails the apply on a
// planned value Terraform holds the provider to. The remaining cases are shapes the
// same parse rules settle.
func TestIPv4CIDR(t *testing.T) {
	cases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"class A block", "10.20.30.0/24", false},
		{"single host", "10.0.0.1/32", false},
		{"whole space", "0.0.0.0/0", false},

		{"bare address", "10.11.12.13", true},
		{"host bits set", "10.1.2.3/24", true},
		{"host bits set on a zero prefix", "10.1.2.3/0", true},
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
// IPv6 block that is already in CIDR format. The host-bits case has to name the range
// it actually describes: the whole point of refusing it is that the operator cannot
// tell `10.1.2.3/24` apart from the `10.1.2.0/24` the server would store.
func TestIPv4CIDRMessages(t *testing.T) {
	cases := map[string]string{
		"2001:db8::/32": "IPv6",
		"10.11.12.13":   "prefix length",
		"":              "required",
		"10.0.0.0/33":   "CIDR notation",
		"10.1.2.3/24":   "10.1.2.0/24",
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

// TestParseIPv4CIDR pins the exported parse to the shape rules plus the prefix length
// its callers need. It deliberately does NOT apply the network-address rule: that rule
// is a wire fact about the ZTNA app's `bareIps` endpoint, and ParseIPv4CIDR serves
// `ztna_gateway`, whose IPsec endpoint has never been probed for canonicalisation. See
// IPv4CIDRAllowingHostBits.
func TestParseIPv4CIDR(t *testing.T) {
	cases := []struct {
		value      string
		wantNet    string
		wantPrefix int
		wantErr    string
	}{
		{value: "10.0.0.0/8", wantNet: "10.0.0.0/8", wantPrefix: 8},
		{value: "192.168.100.0/24", wantNet: "192.168.100.0/24", wantPrefix: 24},
		{value: "0.0.0.0/0", wantNet: "0.0.0.0/0", wantPrefix: 0},
		{value: "10.1.2.3/24", wantNet: "10.1.2.0/24", wantPrefix: 24},
		{value: "::ffff:10.0.0.0/104", wantErr: "IPv6"},
		{value: "10.0.0.1", wantErr: "prefix length"},
		{value: "", wantErr: "required"},
	}
	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			network, prefix, err := ParseIPv4CIDR(tc.value)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("ParseIPv4CIDR(%q) = %v, want an error mentioning %q", tc.value, network, tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("ParseIPv4CIDR(%q) error = %q, want it to mention %q", tc.value, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseIPv4CIDR(%q) unexpected error: %s", tc.value, err)
			}
			if got := network.String(); got != tc.wantNet {
				t.Fatalf("ParseIPv4CIDR(%q) network = %q, want %q", tc.value, got, tc.wantNet)
			}
			if prefix != tc.wantPrefix {
				t.Fatalf("ParseIPv4CIDR(%q) prefix = %d, want %d", tc.value, prefix, tc.wantPrefix)
			}
		})
	}
}
