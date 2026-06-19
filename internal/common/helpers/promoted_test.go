// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package helpers

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func intPtr(i int) *int         { return &i }
func strPtr(s string) *string   { return &s }
func boolPtrLocal(b bool) *bool { return &b }

func TestStringIDPtr(t *testing.T) {
	tests := []struct {
		name  string
		input types.String
		want  *int
	}{
		{"null", types.StringNull(), nil},
		{"unknown", types.StringUnknown(), nil},
		{"empty", types.StringValue(""), nil},
		{"non-numeric", types.StringValue("abc"), nil},
		{"numeric", types.StringValue("42"), intPtr(42)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := StringIDPtr(tc.input)
			if (got == nil) != (tc.want == nil) {
				t.Fatalf("StringIDPtr(%v) nilness = %v, want nil=%v", tc.input, got, tc.want == nil)
			}
			if got != nil && *got != *tc.want {
				t.Errorf("StringIDPtr(%v) = %d, want %d", tc.input, *got, *tc.want)
			}
		})
	}
}

func TestStringFromIntPtr(t *testing.T) {
	if got := StringFromIntPtr(nil); got != nil {
		t.Errorf("StringFromIntPtr(nil) = %v, want nil", got)
	}
	if got := StringFromIntPtr(intPtr(7)); got == nil || *got != "7" {
		t.Errorf("StringFromIntPtr(7) = %v, want \"7\"", got)
	}
}

func TestInt64FromIntPtr(t *testing.T) {
	if got := Int64FromIntPtr(nil); !got.IsNull() {
		t.Errorf("Int64FromIntPtr(nil) should be null")
	}
	if got := Int64FromIntPtr(intPtr(9)); got.IsNull() || got.ValueInt64() != 9 {
		t.Errorf("Int64FromIntPtr(9) = %v, want 9", got)
	}
}

func TestPreferCurrentStringPointer(t *testing.T) {
	// Configured current always wins, even when the API echoes a different value.
	if got := PreferCurrentStringPointer(strPtr("api"), types.StringValue("cfg")); got.ValueString() != "cfg" {
		t.Errorf("configured current should win, got %v", got)
	}
	// Unset current adopts the API value.
	if got := PreferCurrentStringPointer(strPtr("api"), types.StringNull()); got.ValueString() != "api" {
		t.Errorf("unset current should adopt API value, got %v", got)
	}
	// Nil API with unset current is null.
	if got := PreferCurrentStringPointer(nil, types.StringNull()); !got.IsNull() {
		t.Errorf("nil API + unset current should be null, got %v", got)
	}
}

func TestPreferCurrentBoolPointer(t *testing.T) {
	if got := PreferCurrentBoolPointer(boolPtrLocal(false), types.BoolValue(true)); got.ValueBool() != true {
		t.Errorf("configured current should win, got %v", got)
	}
	if got := PreferCurrentBoolPointer(boolPtrLocal(true), types.BoolNull()); got.ValueBool() != true {
		t.Errorf("unset current should adopt API value, got %v", got)
	}
	if got := PreferCurrentBoolPointer(nil, types.BoolNull()); !got.IsNull() {
		t.Errorf("nil API + unset current should be null, got %v", got)
	}
}

func TestProviderNotConfiguredError(t *testing.T) {
	summary, detail := ProviderNotConfiguredError()
	if summary != "Provider not configured" || detail == "" {
		t.Errorf("unexpected provider-not-configured message: %q / %q", summary, detail)
	}
}

func TestInitialSingletonID(t *testing.T) {
	if got := InitialSingletonID(); got.ValueString() != SingletonID {
		t.Errorf("InitialSingletonID() = %v, want %q", got, SingletonID)
	}
}
