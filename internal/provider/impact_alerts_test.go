// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestImpactAlertsEnabled(t *testing.T) {
	tests := []struct {
		name string
		attr types.Bool
		env  *string
		want bool
	}{
		{name: "off by default", attr: types.BoolNull(), want: false},
		{name: "attribute true", attr: types.BoolValue(true), want: true},
		{name: "attribute false", attr: types.BoolValue(false), want: false},
		{name: "unknown attribute falls through to the environment", attr: types.BoolUnknown(), env: new("true"), want: true},
		{name: "environment true", attr: types.BoolNull(), env: new("true"), want: true},
		{name: "environment 1", attr: types.BoolNull(), env: new("1"), want: true},
		{name: "environment false", attr: types.BoolNull(), env: new("false"), want: false},
		{name: "environment whitespace tolerated", attr: types.BoolNull(), env: new("  true  "), want: true},
		// An unparseable value must not stop a plan that would otherwise succeed.
		{name: "unparseable environment value is off", attr: types.BoolNull(), env: new("yes please"), want: false},
		{name: "empty environment value is off", attr: types.BoolNull(), env: new(""), want: false},
		// The attribute is the operator's explicit intent, so it wins.
		{name: "attribute false overrides environment true", attr: types.BoolValue(false), env: new("true"), want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.env != nil {
				t.Setenv(envImpactAlerts, *tc.env)
			}
			if got := impactAlertsEnabled(tc.attr); got != tc.want {
				t.Fatalf("impactAlertsEnabled() = %v, want %v", got, tc.want)
			}
		})
	}
}
