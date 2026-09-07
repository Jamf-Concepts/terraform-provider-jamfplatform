// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package helpers

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
		{"numeric", types.StringValue("42"), new(42)},
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
	if got := StringFromIntPtr(new(7)); got == nil || *got != "7" {
		t.Errorf("StringFromIntPtr(7) = %v, want \"7\"", got)
	}
}

func TestInt64FromIntPtr(t *testing.T) {
	if got := Int64FromIntPtr(nil); !got.IsNull() {
		t.Errorf("Int64FromIntPtr(nil) should be null")
	}
	if got := Int64FromIntPtr(new(9)); got.IsNull() || got.ValueInt64() != 9 {
		t.Errorf("Int64FromIntPtr(9) = %v, want 9", got)
	}
}

func TestStickyIgnoringDriftString(t *testing.T) {
	// Configured current always wins, even when the API echoes a different value.
	if got := StickyIgnoringDriftString(new("api"), types.StringValue("cfg")); got.ValueString() != "cfg" {
		t.Errorf("configured current should win, got %v", got)
	}
	// Unset current adopts the API value.
	if got := StickyIgnoringDriftString(new("api"), types.StringNull()); got.ValueString() != "api" {
		t.Errorf("unset current should adopt API value, got %v", got)
	}
	// Nil API with unset current is null.
	if got := StickyIgnoringDriftString(nil, types.StringNull()); !got.IsNull() {
		t.Errorf("nil API + unset current should be null, got %v", got)
	}
}

func TestStickyIgnoringDriftBool(t *testing.T) {
	if got := StickyIgnoringDriftBool(new(false), types.BoolValue(true)); got.ValueBool() != true {
		t.Errorf("configured current should win, got %v", got)
	}
	if got := StickyIgnoringDriftBool(new(true), types.BoolNull()); got.ValueBool() != true {
		t.Errorf("unset current should adopt API value, got %v", got)
	}
	if got := StickyIgnoringDriftBool(nil, types.BoolNull()); !got.IsNull() {
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

// TestWireWhenPresentString covers the conditionally-echoed read: adopt the
// wire when it speaks, keep state when it does not, and never propagate an
// Unknown into state. Promoted from mobile_device_app, where the same shape was
// written for `general.os_type`.
func TestWireWhenPresentString(t *testing.T) {
	if got := WireWhenPresentString(new("tvOS"), types.StringValue("iOS")); got.ValueString() != "tvOS" {
		t.Errorf("wire present: got %q want tvOS", got.ValueString())
	}
	if got := WireWhenPresentString(new(""), types.StringValue("iOS")); got.ValueString() != "iOS" {
		t.Errorf("wire empty counts as absent: got %q want iOS", got.ValueString())
	}
	if got := WireWhenPresentString(nil, types.StringValue("iOS")); got.ValueString() != "iOS" {
		t.Errorf("wire absent, known current: got %q want iOS", got.ValueString())
	}
	if got := WireWhenPresentString(nil, types.StringNull()); !got.IsNull() {
		t.Errorf("wire absent, null current: got %q want null", got.ValueString())
	}
	if got := WireWhenPresentString(nil, types.StringUnknown()); got.IsUnknown() || !got.IsNull() {
		t.Errorf("wire absent, unknown current: unknown=%v null=%v want null", got.IsUnknown(), got.IsNull())
	}
}

// TestWireWhenPresentBool is the bool sibling. There is no empty form, so only
// a nil wire pointer counts as absent — a wire false must win over a state
// true, which is the whole point of the helper.
func TestWireWhenPresentBool(t *testing.T) {
	if got := WireWhenPresentBool(new(false), types.BoolValue(true)); got.ValueBool() {
		t.Error("wire present false must win over state true")
	}
	if got := WireWhenPresentBool(nil, types.BoolValue(true)); !got.ValueBool() {
		t.Error("wire absent, known current: must keep true")
	}
	if got := WireWhenPresentBool(nil, types.BoolNull()); !got.IsNull() {
		t.Error("wire absent, null current: must stay null")
	}
	if got := WireWhenPresentBool(nil, types.BoolUnknown()); got.IsUnknown() || !got.IsNull() {
		t.Errorf("wire absent, unknown current: unknown=%v null=%v want null", got.IsUnknown(), got.IsNull())
	}
}

// TestWireWhenPresentInt64 is the Int64 sibling, taking the *int the ProClassic
// SDK uses for integer wire fields.
func TestWireWhenPresentInt64(t *testing.T) {
	if got := WireWhenPresentInt64(new(7), types.Int64Value(3)); got.ValueInt64() != 7 {
		t.Errorf("wire present: got %d want 7", got.ValueInt64())
	}
	if got := WireWhenPresentInt64(new(0), types.Int64Value(3)); got.ValueInt64() != 0 {
		t.Errorf("wire zero is a value, not absence: got %d want 0", got.ValueInt64())
	}
	if got := WireWhenPresentInt64(nil, types.Int64Value(3)); got.ValueInt64() != 3 {
		t.Errorf("wire absent, known current: got %d want 3", got.ValueInt64())
	}
	if got := WireWhenPresentInt64(nil, types.Int64Unknown()); got.IsUnknown() || !got.IsNull() {
		t.Errorf("wire absent, unknown current: unknown=%v null=%v want null", got.IsUnknown(), got.IsNull())
	}
}
