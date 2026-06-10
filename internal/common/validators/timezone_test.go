// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package validators

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// runIANATimeZone invokes IANATimeZone() against a single ConfigValue and
// returns the diagnostic summaries. The validator reads only req.ConfigValue
// (no companion lookups), so no synthesised config is needed.
func runIANATimeZone(v types.String) []string {
	resp := &validator.StringResponse{}
	IANATimeZone().ValidateString(context.Background(), validator.StringRequest{
		Path:        path.Root("timezone"),
		ConfigValue: v,
	}, resp)
	out := []string{}
	for _, d := range resp.Diagnostics {
		out = append(out, d.Summary())
	}
	return out
}

func TestIANATimeZone(t *testing.T) {
	tests := []struct {
		name    string
		value   types.String
		wantErr bool
	}{
		{"UTC accepted (absent from /v1/time-zones but server-accepted)", types.StringValue("UTC"), false},
		{"region id", types.StringValue("America/Chicago"), false},
		{"Etc/UTC accepted (absent from /v1/time-zones but server-accepted)", types.StringValue("Etc/UTC"), false},
		{"GMT", types.StringValue("GMT"), false},
		{"typo rejected", types.StringValue("America/Chicagoo"), true},
		{"garbage rejected", types.StringValue("garbage"), true},
		{"legacy Java alias PST rejected (server rejects it too)", types.StringValue("PST"), true},
		{"Go-only Local alias rejected", types.StringValue("Local"), true},
		{"empty deferred to LengthAtLeast", types.StringValue(""), false},
		{"null skipped", types.StringNull(), false},
		{"unknown skipped", types.StringUnknown(), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := runIANATimeZone(tc.value)
			if tc.wantErr && len(got) == 0 {
				t.Fatalf("expected a validation error, got none")
			}
			if !tc.wantErr && len(got) != 0 {
				t.Fatalf("expected no error, got %v", got)
			}
		})
	}
}
