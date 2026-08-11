// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestImpactAlertsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		attr     types.Bool
		env      *string
		want     bool
		wantWarn bool
	}{
		{name: "off by default", attr: types.BoolNull(), want: false},
		{name: "attribute true", attr: types.BoolValue(true), want: true},
		{name: "attribute false", attr: types.BoolValue(false), want: false},
		{name: "unknown attribute falls through to the environment", attr: types.BoolUnknown(), env: new("true"), want: true},
		{name: "environment true", attr: types.BoolNull(), env: new("true"), want: true},
		{name: "environment 1", attr: types.BoolNull(), env: new("1"), want: true},
		{name: "environment false", attr: types.BoolNull(), env: new("false"), want: false},
		{name: "environment whitespace tolerated", attr: types.BoolNull(), env: new("  true  "), want: true},
		// An unparseable value must not stop a plan that would otherwise succeed,
		// but it must not silently do nothing either.
		{name: "unparseable environment value is off with a warning", attr: types.BoolNull(), env: new("yes please"), want: false, wantWarn: true},
		// An empty value is treated as unset, matching the other env variables.
		{name: "empty environment value is off without a warning", attr: types.BoolNull(), env: new(""), want: false},
		{name: "blank environment value is off without a warning", attr: types.BoolNull(), env: new("   "), want: false},
		// The attribute is the operator's explicit intent, so it wins.
		{name: "attribute false overrides environment true", attr: types.BoolValue(false), env: new("true"), want: false},
		// A set attribute means the environment is never consulted, so even an
		// unparseable value earns no warning.
		{name: "attribute set suppresses the environment warning", attr: types.BoolValue(true), env: new("yes please"), want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.env != nil {
				t.Setenv(envImpactAlerts, *tc.env)
			}
			got, warn := impactAlertsEnabled(tc.attr)
			if got != tc.want {
				t.Fatalf("impactAlertsEnabled() = %v, want %v", got, tc.want)
			}
			if (warn != "") != tc.wantWarn {
				t.Fatalf("impactAlertsEnabled() warning = %q, want warning: %v", warn, tc.wantWarn)
			}
			if tc.wantWarn {
				if tc.env == nil || !strings.Contains(warn, *tc.env) {
					t.Fatalf("warning %q does not name the offending value", warn)
				}
				if !strings.Contains(warn, envImpactAlerts) {
					t.Fatalf("warning %q does not name the environment variable", warn)
				}
			}
		})
	}
}
