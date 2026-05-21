// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package helpers

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestIntIDToString(t *testing.T) {
	tests := []struct {
		id   int64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{12345, "12345"},
		{-1, "-1"},
	}
	for _, tc := range tests {
		got := IntIDToString(tc.id)
		if got.ValueString() != tc.want {
			t.Errorf("IntIDToString(%d) = %q, want %q", tc.id, got.ValueString(), tc.want)
		}
	}
}

func TestStringValueFromIntPtr(t *testing.T) {
	zero := 0
	pos := 12345
	neg := -1

	tests := []struct {
		name     string
		in       *int
		wantNull bool
		want     string
	}{
		{"nil", nil, true, ""},
		{"zero", &zero, false, "0"},
		{"positive", &pos, false, "12345"},
		{"negative", &neg, false, "-1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := StringValueFromIntPtr(tc.in)
			if tc.wantNull {
				if !got.IsNull() {
					t.Fatalf("expected null types.String, got %#v", got)
				}
				return
			}
			if got.IsNull() {
				t.Fatalf("expected non-null types.String for %v, got null", tc.in)
			}
			if got.ValueString() != tc.want {
				t.Errorf("got %q, want %q", got.ValueString(), tc.want)
			}
		})
	}
}

func TestStringToIntID(t *testing.T) {
	tests := []struct {
		name    string
		input   types.String
		want    int64
		wantErr bool
	}{
		{"valid", types.StringValue("12345"), 12345, false},
		{"zero", types.StringValue("0"), 0, false},
		{"negative", types.StringValue("-1"), -1, false},
		{"null", types.StringNull(), 0, true},
		{"unknown", types.StringUnknown(), 0, true},
		{"non-numeric", types.StringValue("abc"), 0, true},
		{"empty", types.StringValue(""), 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := StringToIntID(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("got = %d, want %d", got, tc.want)
			}
		})
	}
}
