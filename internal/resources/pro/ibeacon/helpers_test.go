// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ibeacon

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Truth table for validateIbeaconPlan with two independent axes. The bool
// constructor matrix tests every combination of (IncludeAnyMajor, Major) for
// one axis; symmetric on minor. Mixed-axis combinations covered by the
// Combined tests below.

func TestValidateIbeaconPlan_AxisMajor(t *testing.T) {
	cases := []struct {
		name           string
		includeAny     types.Bool
		major          types.Int64
		expectError    bool
		errorSubstring string
	}{
		{"IncludeAny=true + major set → error", types.BoolValue(true), types.Int64Value(1), true, "include_any_major_value"},
		{"IncludeAny=true + major null → ok", types.BoolValue(true), types.Int64Null(), false, ""},
		{"IncludeAny=false + major set → ok", types.BoolValue(false), types.Int64Value(1), false, ""},
		{"IncludeAny=false + major null → error", types.BoolValue(false), types.Int64Null(), true, "major must be set"},
		{"IncludeAny=null + major set → ok (null treated as false)", types.BoolNull(), types.Int64Value(1), false, ""},
		{"IncludeAny=null + major null → error (null treated as false)", types.BoolNull(), types.Int64Null(), true, "major must be set"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			plan := &IbeaconResourceModel{
				IncludeAnyMajorValue: c.includeAny,
				Major:                c.major,
				IncludeAnyMinorValue: types.BoolValue(false),
				Minor:                types.Int64Value(0),
			}
			err := validateIbeaconPlan(plan)
			if c.expectError && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !c.expectError && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if c.expectError && c.errorSubstring != "" && err != nil && !contains(err.Error(), c.errorSubstring) {
				t.Errorf("expected error containing %q, got %q", c.errorSubstring, err.Error())
			}
		})
	}
}

func TestValidateIbeaconPlan_AxisMinor(t *testing.T) {
	cases := []struct {
		name           string
		includeAny     types.Bool
		minor          types.Int64
		expectError    bool
		errorSubstring string
	}{
		{"IncludeAny=true + minor set → error", types.BoolValue(true), types.Int64Value(2), true, "include_any_minor_value"},
		{"IncludeAny=true + minor null → ok", types.BoolValue(true), types.Int64Null(), false, ""},
		{"IncludeAny=false + minor set → ok", types.BoolValue(false), types.Int64Value(2), false, ""},
		{"IncludeAny=false + minor null → error", types.BoolValue(false), types.Int64Null(), true, "minor must be set"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			plan := &IbeaconResourceModel{
				IncludeAnyMajorValue: types.BoolValue(false),
				Major:                types.Int64Value(0),
				IncludeAnyMinorValue: c.includeAny,
				Minor:                c.minor,
			}
			err := validateIbeaconPlan(plan)
			if c.expectError && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !c.expectError && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if c.expectError && c.errorSubstring != "" && err != nil && !contains(err.Error(), c.errorSubstring) {
				t.Errorf("expected error containing %q, got %q", c.errorSubstring, err.Error())
			}
		})
	}
}

func TestValidateIbeaconPlan_Combined(t *testing.T) {
	cases := []struct {
		name        string
		plan        *IbeaconResourceModel
		expectError bool
	}{
		{
			"both axes any, both null",
			&IbeaconResourceModel{
				IncludeAnyMajorValue: types.BoolValue(true),
				IncludeAnyMinorValue: types.BoolValue(true),
				Major:                types.Int64Null(),
				Minor:                types.Int64Null(),
			},
			false,
		},
		{
			"any major + concrete minor",
			&IbeaconResourceModel{
				IncludeAnyMajorValue: types.BoolValue(true),
				IncludeAnyMinorValue: types.BoolValue(false),
				Major:                types.Int64Null(),
				Minor:                types.Int64Value(5),
			},
			false,
		},
		{
			"concrete major + any minor",
			&IbeaconResourceModel{
				IncludeAnyMajorValue: types.BoolValue(false),
				IncludeAnyMinorValue: types.BoolValue(true),
				Major:                types.Int64Value(42),
				Minor:                types.Int64Null(),
			},
			false,
		},
		{
			"both concrete",
			&IbeaconResourceModel{
				IncludeAnyMajorValue: types.BoolValue(false),
				IncludeAnyMinorValue: types.BoolValue(false),
				Major:                types.Int64Value(10),
				Minor:                types.Int64Value(20),
			},
			false,
		},
		{
			"both unset, both axes false → error",
			&IbeaconResourceModel{
				IncludeAnyMajorValue: types.BoolValue(false),
				IncludeAnyMinorValue: types.BoolValue(false),
				Major:                types.Int64Null(),
				Minor:                types.Int64Null(),
			},
			true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateIbeaconPlan(c.plan)
			if c.expectError && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !c.expectError && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
