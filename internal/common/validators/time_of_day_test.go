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

// runTimeOfDayHHMMSS invokes TimeOfDayHHMMSS(allowEmpty) against a single
// ConfigValue and returns the diagnostic summaries. The validator reads only
// req.ConfigValue (no companion lookups), so no synthesised config is needed.
func runTimeOfDayHHMMSS(v types.String, allowEmpty bool) []string {
	resp := &validator.StringResponse{}
	TimeOfDayHHMMSS(allowEmpty).ValidateString(context.Background(), validator.StringRequest{
		Path:        path.Root("time_of_day"),
		ConfigValue: v,
	}, resp)
	out := []string{}
	for _, d := range resp.Diagnostics {
		out = append(out, d.Summary())
	}
	return out
}

// TestTimeOfDayHHMMSS pins the canonical HH:MM:SS shape: 24-hour canonical
// times pass; short forms (which the server would canonicalize, perma-diffing
// config against state) and invalid times fail; "" is gated by allowEmpty
// (the full-replace clear sentinel).
func TestTimeOfDayHHMMSS(t *testing.T) {
	tests := []struct {
		name       string
		value      types.String
		allowEmpty bool
		wantErr    bool
	}{
		{"midnight", types.StringValue("00:00:00"), true, false},
		{"morning", types.StringValue("05:30:00"), true, false},
		{"evening", types.StringValue("17:30:00"), true, false},
		{"last second of day", types.StringValue("23:59:59"), true, false},
		{"clear sentinel allowed", types.StringValue(""), true, false},
		{"clear sentinel rejected when disallowed", types.StringValue(""), false, true},
		{"valid time with allowEmpty false", types.StringValue("08:30:00"), false, false},
		{"short HH:MM rejected (server canonicalizes to HH:MM:SS)", types.StringValue("05:30"), true, true},
		{"single-digit hour rejected", types.StringValue("5:30:00"), true, true},
		{"hour 24 rejected", types.StringValue("24:00:00"), true, true},
		{"minute 60 rejected", types.StringValue("23:60:00"), true, true},
		{"second 60 rejected", types.StringValue("23:00:60"), true, true},
		{"garbage rejected", types.StringValue("banana"), true, true},
		{"trailing space rejected", types.StringValue("17:30:00 "), true, true},
		{"leading space rejected", types.StringValue(" 17:30:00"), true, true},
		{"no separators rejected", types.StringValue("173000"), true, true},
		{"null skipped", types.StringNull(), false, false},
		{"unknown skipped", types.StringUnknown(), false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := runTimeOfDayHHMMSS(tc.value, tc.allowEmpty)
			if tc.wantErr && len(got) == 0 {
				t.Fatalf("expected a validation error, got none")
			}
			if !tc.wantErr && len(got) != 0 {
				t.Fatalf("expected no error, got %v", got)
			}
		})
	}
}
