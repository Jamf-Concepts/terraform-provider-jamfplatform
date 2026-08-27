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
		{"prefix too short for 172", types.StringValue("172.16.0.0/11"), true},
		{"prefix too short for 192", types.StringValue("192.168.0.0/15"), true},
		{"public range", types.StringValue("8.8.8.0/24"), true},
		{"default route", types.StringValue("0.0.0.0/0"), true},
		{"bare address", types.StringValue("10.0.0.1"), true},
		{"ipv6", types.StringValue("fd00::/8"), true},
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

func TestIPv4Address(t *testing.T) {
	cases := []struct {
		name    string
		value   types.String
		wantErr bool
	}{
		{"dotted quad", types.StringValue("3.9.67.90"), false},
		{"ipv6", types.StringValue("2001:db8::1"), true},
		{"cidr", types.StringValue("3.9.67.90/32"), true},
		{"nonsense", types.StringValue("nope"), true},
		{"null defers to server", types.StringNull(), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("availability_zones"),
				ConfigValue: tc.value,
			}
			var resp validator.StringResponse
			ipv4Address().ValidateString(context.Background(), req, &resp)

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
		"ipv4Address": ipv4Address(),
	} {
		if v.Description(context.Background()) == "" {
			t.Errorf("%s must describe itself", name)
		}
	}
}
