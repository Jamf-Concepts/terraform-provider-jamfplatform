// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package dns_zone

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestIPv4Address covers the three inputs Jamf Security Cloud collapses into one
// opaque INVALID_FIELD message, plus the forms it accepts.
func TestIPv4Address(t *testing.T) {
	cases := []struct {
		name    string
		value   types.String
		wantErr bool
	}{
		{"dotted quad", types.StringValue("203.0.113.53"), false},
		{"public resolver", types.StringValue("8.8.8.8"), false},
		{"zero octet", types.StringValue("10.0.0.53"), false},
		{"leading zeros", types.StringValue("203.000.113.053"), true},
		{"ipv6", types.StringValue("2001:db8::53"), true},
		{"not an address", types.StringValue("not-an-ip"), true},
		{"three octets", types.StringValue("203.0.113"), true},
		{"five octets", types.StringValue("203.0.113.53.1"), true},
		{"octet out of range", types.StringValue("203.0.113.256"), true},
		{"trailing dot", types.StringValue("203.0.113.53."), true},
		{"empty", types.StringValue(""), true},
		{"null defers to server", types.StringNull(), false},
		{"unknown defers to server", types.StringUnknown(), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("name_servers").AtSetValue(types.StringValue("x")).AtName("ip"),
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

func TestIPv4Address_Description(t *testing.T) {
	if ipv4Address().Description(context.Background()) == "" {
		t.Error("validator must describe itself")
	}
}
