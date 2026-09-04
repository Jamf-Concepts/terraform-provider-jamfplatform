// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_predefined_apps

import (
	"context"
	"testing"
)

// TestHostnameListValue pins the nil-to-empty normalisation. The failure it guards
// is not a Go panic but a Terraform plan error: types.ListValueFrom reflects a nil
// slice into a NULL list, and a `for` expression over a null list fails at plan
// time. A reviewer reverting to the plain ListValueFrom call would see every other
// assertion in this package still pass.
func TestHostnameListValue(t *testing.T) {
	tests := []struct {
		name      string
		hostnames []string
		wantNull  bool
		wantLen   int
	}{
		{name: "nil normalises to an empty list", hostnames: nil, wantNull: false, wantLen: 0},
		{name: "empty stays an empty list", hostnames: []string{}, wantNull: false, wantLen: 0},
		{name: "populated round-trips", hostnames: []string{"slack.com", "app.slack.com"}, wantNull: false, wantLen: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, diags := hostnameListValue(context.Background(), tt.hostnames)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if got.IsNull() != tt.wantNull {
				t.Errorf("IsNull() = %v, want %v", got.IsNull(), tt.wantNull)
			}
			if len(got.Elements()) != tt.wantLen {
				t.Errorf("element count = %d, want %d", len(got.Elements()), tt.wantLen)
			}
		})
	}
}
