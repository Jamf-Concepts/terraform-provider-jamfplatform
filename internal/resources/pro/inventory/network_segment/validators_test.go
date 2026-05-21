// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package network_segment

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestIPv4Validator_AcceptsIPv4(t *testing.T) {
	for _, in := range []string{"10.0.0.0", "192.168.1.255", "127.0.0.1", "0.0.0.0", "255.255.255.255"} {
		t.Run(in, func(t *testing.T) {
			req := validator.StringRequest{Path: path.Root("starting_address"), ConfigValue: types.StringValue(in)}
			var resp validator.StringResponse
			ipv4Address().ValidateString(context.Background(), req, &resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("expected no diagnostics, got: %v", resp.Diagnostics)
			}
		})
	}
}

func TestIPv4Validator_RejectsNonIP(t *testing.T) {
	for _, in := range []string{"", "not-an-ip", "10.0.0", "10.0.0.0.0", "999.0.0.0", "10.0.0.0/24"} {
		t.Run(in, func(t *testing.T) {
			req := validator.StringRequest{Path: path.Root("starting_address"), ConfigValue: types.StringValue(in)}
			var resp validator.StringResponse
			ipv4Address().ValidateString(context.Background(), req, &resp)
			if !resp.Diagnostics.HasError() {
				t.Errorf("expected diagnostics for %q", in)
			}
		})
	}
}

func TestIPv4Validator_RejectsIPv6(t *testing.T) {
	for _, in := range []string{"::1", "fe80::1", "2001:db8::1"} {
		t.Run(in, func(t *testing.T) {
			req := validator.StringRequest{Path: path.Root("starting_address"), ConfigValue: types.StringValue(in)}
			var resp validator.StringResponse
			ipv4Address().ValidateString(context.Background(), req, &resp)
			if !resp.Diagnostics.HasError() {
				t.Errorf("expected IPv6 input %q to be rejected", in)
			}
		})
	}
}

func TestIPv4Validator_SkipsNullAndUnknown(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   types.String
	}{
		{"null", types.StringNull()},
		{"unknown", types.StringUnknown()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.StringRequest{Path: path.Root("starting_address"), ConfigValue: tc.in}
			var resp validator.StringResponse
			ipv4Address().ValidateString(context.Background(), req, &resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("expected no diagnostics for %s, got: %v", tc.name, resp.Diagnostics)
			}
		})
	}
}
