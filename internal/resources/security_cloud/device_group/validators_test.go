// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestGroupName(t *testing.T) {
	tests := []struct {
		name        string
		value       types.String
		wantError   bool
		wantSummary string
	}{
		{
			name:  "plain name",
			value: types.StringValue("Executives"),
		},
		{
			name:  "internal whitespace is fine",
			value: types.StringValue("Field Staff EMEA"),
		},
		{
			name:  "non-ASCII is fine — the server stores it verbatim",
			value: types.StringValue("Führungskräfte 日本"),
		},
		{
			name:  "no length limit — the server accepted 65536 characters",
			value: types.StringValue(strings.Repeat("N", 5000)),
		},
		{
			name:  "null deferred to the server",
			value: types.StringNull(),
		},
		{
			name:  "unknown deferred to the server",
			value: types.StringUnknown(),
		},
		{
			name:        "leading whitespace",
			value:       types.StringValue(" Executives"),
			wantError:   true,
			wantSummary: "Group name has surrounding whitespace",
		},
		{
			name:        "trailing whitespace",
			value:       types.StringValue("Executives "),
			wantError:   true,
			wantSummary: "Group name has surrounding whitespace",
		},
		{
			name:        "whitespace only",
			value:       types.StringValue("   "),
			wantError:   true,
			wantSummary: "Group name has surrounding whitespace",
		},
		{
			name:        "reserved name exactly",
			value:       types.StringValue("Default Group"),
			wantError:   true,
			wantSummary: "Group name is reserved",
		},
		{
			name:        "reserved name lowercased — the server compares case-insensitively",
			value:       types.StringValue("default group"),
			wantError:   true,
			wantSummary: "Group name is reserved",
		},
		{
			name:        "reserved name uppercased",
			value:       types.StringValue("DEFAULT GROUP"),
			wantError:   true,
			wantSummary: "Group name is reserved",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("name"),
				ConfigValue: tc.value,
			}
			var resp validator.StringResponse

			GroupName().ValidateString(context.Background(), req, &resp)

			if got := resp.Diagnostics.HasError(); got != tc.wantError {
				t.Fatalf("HasError() = %v, want %v (diags: %v)", got, tc.wantError, resp.Diagnostics)
			}
			if !tc.wantError {
				return
			}
			if resp.Diagnostics[0].Summary() != tc.wantSummary {
				t.Errorf("summary = %q, want %q", resp.Diagnostics[0].Summary(), tc.wantSummary)
			}
		})
	}
}

// TestGroupName_ReservedCheckRunsAfterTrim pins the interaction between the two
// rules. The server trims first and then compares against the reserved name, so
// "Default Group " is refused for the reservation rather than accepted. The
// validator reports it as a whitespace problem, which is the more actionable of
// the two — but either way it must not pass.
func TestGroupName_ReservedCheckRunsAfterTrim(t *testing.T) {
	req := validator.StringRequest{
		Path:        path.Root("name"),
		ConfigValue: types.StringValue("Default Group "),
	}
	var resp validator.StringResponse

	GroupName().ValidateString(context.Background(), req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("\"Default Group \" must be refused — the server trims and then rejects it as reserved")
	}
}

// TestGroupName_DescriptionNamesBothRules keeps the generated documentation
// honest about what the validator enforces.
func TestGroupName_DescriptionNamesBothRules(t *testing.T) {
	got := GroupName().Description(context.Background())

	if !strings.Contains(got, "whitespace") {
		t.Errorf("description must mention whitespace, got %q", got)
	}
	if !strings.Contains(got, defaultGroupName) {
		t.Errorf("description must name %q, got %q", defaultGroupName, got)
	}
}
