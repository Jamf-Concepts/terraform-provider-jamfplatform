// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package printer

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestStringPtrFromBool(t *testing.T) {
	cases := []struct {
		name    string
		in      types.Bool
		want    *string
		wantNil bool
	}{
		{"true", types.BoolValue(true), new("true"), false},
		{"false", types.BoolValue(false), new("false"), false},
		{"null", types.BoolNull(), nil, true},
		{"unknown", types.BoolUnknown(), nil, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := stringPtrFromBool(c.in)
			if c.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %q", *got)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected %q, got nil", *c.want)
			}
			if *got != *c.want {
				t.Errorf("expected %q, got %q", *c.want, *got)
			}
		})
	}
}

func TestBoolValueFromStringPtr(t *testing.T) {
	cases := []struct {
		name string
		in   *string
		want types.Bool
	}{
		{"true", new("true"), types.BoolValue(true)},
		{"false", new("false"), types.BoolValue(false)},
		{"empty", new(""), types.BoolNull()},
		{"nil", nil, types.BoolNull()},
		{"unknown_wire_value_falls_to_false", new("xyz"), types.BoolValue(false)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := boolValueFromStringPtr(c.in)
			if got.IsNull() != c.want.IsNull() {
				t.Errorf("null-ness: expected %v, got %v", c.want.IsNull(), got.IsNull())
				return
			}
			if !got.IsNull() && got.ValueBool() != c.want.ValueBool() {
				t.Errorf("value: expected %v, got %v", c.want.ValueBool(), got.ValueBool())
			}
		})
	}
}

func TestStringPtrEmitAlways(t *testing.T) {
	cases := []struct {
		name string
		in   types.String
		want string
	}{
		{"value", types.StringValue("Printers"), "Printers"},
		{"empty", types.StringValue(""), ""},
		{"null", types.StringNull(), ""},
		{"unknown", types.StringUnknown(), ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := stringPtrEmitAlways(c.in)
			if got == nil {
				t.Fatalf("expected non-nil pointer regardless of null state")
			}
			if *got != c.want {
				t.Errorf("expected %q, got %q", c.want, *got)
			}
		})
	}
}

func TestDecodeCategory(t *testing.T) {
	sentinel := categoryUnassignedSentinel
	empty := ""
	value := "Printers"

	cases := []struct {
		name     string
		in       *string
		wantNull bool
		wantVal  string
	}{
		{"nil", nil, true, ""},
		{"empty", &empty, true, ""},
		{"sentinel", &sentinel, true, ""},
		{"value", &value, false, "Printers"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := decodeCategory(c.in)
			if got.IsNull() != c.wantNull {
				t.Errorf("null-ness: expected %v, got %v (value=%q)", c.wantNull, got.IsNull(), got.ValueString())
				return
			}
			if !got.IsNull() && got.ValueString() != c.wantVal {
				t.Errorf("expected %q, got %q", c.wantVal, got.ValueString())
			}
		})
	}
}

func TestDerefString(t *testing.T) {
	if derefString(nil) != "" {
		t.Errorf("nil deref must be empty string")
	}
	v := "hello"
	if derefString(&v) != "hello" {
		t.Errorf("expected hello, got %q", derefString(&v))
	}
}
