// Copyright 2026 Jamf Software LLC.

package helpers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestStringValueOrNull(t *testing.T) {
	tests := []struct {
		input    string
		wantNull bool
	}{
		{"", true},
		{"hello", false},
	}
	for _, tc := range tests {
		result := StringValueOrNull(tc.input)
		if tc.wantNull && !result.IsNull() {
			t.Errorf("StringValueOrNull(%q) should be null", tc.input)
		}
		if !tc.wantNull && result.ValueString() != tc.input {
			t.Errorf("StringValueOrNull(%q) = %q", tc.input, result.ValueString())
		}
	}
}

func TestStringPointerValueOrNull(t *testing.T) {
	s := "hello"
	empty := ""

	if result := StringPointerValueOrNull(nil); !result.IsNull() {
		t.Error("nil pointer should produce null")
	}
	if result := StringPointerValueOrNull(&empty); !result.IsNull() {
		t.Error("empty string pointer should produce null")
	}
	if result := StringPointerValueOrNull(&s); result.ValueString() != "hello" {
		t.Errorf("expected 'hello', got %q", result.ValueString())
	}
}

func TestBoolPointerValueOrNull(t *testing.T) {
	b := true
	f := false

	if result := BoolPointerValueOrNull(nil); !result.IsNull() {
		t.Error("nil pointer should produce null")
	}
	if result := BoolPointerValueOrNull(&b); result.ValueBool() != true {
		t.Error("expected true")
	}
	if result := BoolPointerValueOrNull(&f); result.ValueBool() != false {
		t.Error("expected false")
	}
}

func TestIsConfiguredValue(t *testing.T) {
	if IsConfiguredValue(types.StringNull()) {
		t.Error("null should not be configured")
	}
	if IsConfiguredValue(types.StringUnknown()) {
		t.Error("unknown should not be configured")
	}
	if !IsConfiguredValue(types.StringValue("x")) {
		t.Error("concrete value should be configured")
	}
	if !IsConfiguredValue(types.StringValue("")) {
		t.Error("empty string should be configured")
	}
}

func TestStringPointerValue(t *testing.T) {
	if result := StringPointerValue(types.StringNull()); result != nil {
		t.Error("null should produce nil pointer")
	}
	result := StringPointerValue(types.StringValue("test"))
	if result == nil || *result != "test" {
		t.Error("expected pointer to 'test'")
	}
	result = StringPointerValue(types.StringValue(""))
	if result == nil || *result != "" {
		t.Error("expected pointer to empty string")
	}
}

func TestBoolPointerValue(t *testing.T) {
	if result := BoolPointerValue(types.BoolNull()); result != nil {
		t.Error("null should produce nil pointer")
	}
	result := BoolPointerValue(types.BoolValue(true))
	if result == nil || *result != true {
		t.Error("expected pointer to true")
	}
}

func TestReconcileOptionalBool(t *testing.T) {
	if result := ReconcileOptionalBool(true, types.BoolNull()); !result.IsNull() {
		t.Error("unmanaged field should stay null")
	}
	if result := ReconcileOptionalBool(true, types.BoolValue(false)); result.ValueBool() != true {
		t.Error("managed field should take API value")
	}
}

func TestReconcileOptionalInt(t *testing.T) {
	if result := ReconcileOptionalInt(42, types.Int64Null()); !result.IsNull() {
		t.Error("unmanaged field should stay null")
	}
	if result := ReconcileOptionalInt(42, types.Int64Value(0)); result.ValueInt64() != 42 {
		t.Error("managed field should take API value")
	}
}

func TestReconcileOptionalString(t *testing.T) {
	if result := ReconcileOptionalString("", types.StringNull()); !result.IsNull() {
		t.Error("empty API + null current should produce null")
	}
	if result := ReconcileOptionalString("", types.StringValue("")); result.ValueString() != "" {
		t.Error("empty API + explicit empty should preserve empty")
	}
	if result := ReconcileOptionalString("hello", types.StringNull()); result.ValueString() != "hello" {
		t.Error("non-empty API should produce value")
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		err      error
		expected bool
	}{
		{nil, false},
		{errors.New("some error"), false},
		{errors.New("status 404"), true},
		{errors.New("resource was not found"), true},
		{errors.New("NOT_FOUND"), true},
	}
	for _, tc := range tests {
		if result := IsNotFoundError(tc.err); result != tc.expected {
			t.Errorf("IsNotFoundError(%v) = %v, want %v", tc.err, result, tc.expected)
		}
	}
}

func TestNormalizedFilterString(t *testing.T) {
	tests := []struct {
		input    types.String
		wantStr  string
		wantBool bool
	}{
		{types.StringNull(), "", false},
		{types.StringUnknown(), "", false},
		{types.StringValue(""), "", false},
		{types.StringValue("  "), "", false},
		{types.StringValue(" hello "), "hello", true},
		{types.StringValue("test"), "test", true},
	}
	for _, tc := range tests {
		str, ok := NormalizedFilterString(tc.input)
		if str != tc.wantStr || ok != tc.wantBool {
			t.Errorf("NormalizedFilterString(%v) = (%q, %v), want (%q, %v)", tc.input, str, ok, tc.wantStr, tc.wantBool)
		}
	}
}

func TestPollUntil_ImmediateSuccess(t *testing.T) {
	err := PollUntil(context.Background(), time.Millisecond, func(ctx context.Context) (bool, error) {
		return true, nil
	})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestPollUntil_ErrorPropagation(t *testing.T) {
	expected := errors.New("check failed")
	err := PollUntil(context.Background(), time.Millisecond, func(ctx context.Context) (bool, error) {
		return false, expected
	})
	if !errors.Is(err, expected) {
		t.Errorf("expected %v, got %v", expected, err)
	}
}

func TestPollUntil_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := PollUntil(ctx, time.Millisecond, func(ctx context.Context) (bool, error) {
		return false, nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestPollUntil_EventualSuccess(t *testing.T) {
	calls := 0
	err := PollUntil(context.Background(), time.Millisecond, func(ctx context.Context) (bool, error) {
		calls++
		return calls >= 3, nil
	})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if calls < 3 {
		t.Errorf("expected at least 3 calls, got %d", calls)
	}
}
