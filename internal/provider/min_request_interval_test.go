// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestResolveMinRequestInterval(t *testing.T) {
	tests := []struct {
		name    string
		attr    types.Int64
		env     *string
		want    time.Duration
		wantErr bool
	}{
		// The provider paces nothing unless asked to: a zero interval is always
		// passed to the client, overriding the SDK's own default.
		{name: "no pacing by default", attr: types.Int64Null(), want: 0},
		{name: "attribute set", attr: types.Int64Value(250), want: 250 * time.Millisecond},
		{name: "attribute explicitly zero", attr: types.Int64Value(0), want: 0},
		{name: "unknown attribute falls through to the environment", attr: types.Int64Unknown(), env: new("500"), want: 500 * time.Millisecond},
		{name: "environment set", attr: types.Int64Null(), env: new("500"), want: 500 * time.Millisecond},
		// The attribute is the operator's explicit intent, so it wins.
		{name: "attribute overrides environment", attr: types.Int64Value(250), env: new("500"), want: 250 * time.Millisecond},
		{name: "attribute zero overrides environment", attr: types.Int64Value(0), env: new("500"), want: 0},
		// An empty value is treated as unset, matching the other env variables.
		{name: "empty environment value is no pacing", attr: types.Int64Null(), env: new(""), want: 0},
		// Silently picking a different pace would misreport what was asked for.
		{name: "unparseable environment value errors", attr: types.Int64Null(), env: new("fast please"), wantErr: true},
		// A set attribute means the environment is never consulted, so even an
		// unparseable value is harmless.
		{name: "attribute set suppresses the environment error", attr: types.Int64Value(250), env: new("fast please"), want: 250 * time.Millisecond},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Always stamped, so a variable set in the ambient environment cannot
			// change the result of the unset cases.
			env := ""
			if tc.env != nil {
				env = *tc.env
			}
			t.Setenv(envMinRequestIntervalMs, env)

			got, err := resolveMinRequestInterval(tc.attr)
			if (err != nil) != tc.wantErr {
				t.Fatalf("resolveMinRequestInterval() error = %v, want error: %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if got != tc.want {
				t.Fatalf("resolveMinRequestInterval() = %v, want %v", got, tc.want)
			}
		})
	}
}
