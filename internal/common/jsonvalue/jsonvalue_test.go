// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package jsonvalue

import (
	"encoding/json"
	"math"
	"testing"
)

func TestNumeric(t *testing.T) {
	cases := []struct {
		name  string
		value any
		want  float64
		ok    bool
	}{
		{"float64", float64(1.5), 1.5, true},
		{"float32", float32(2), 2, true},
		{"int", 3, 3, true},
		{"int64", int64(4), 4, true},
		{"json.Number", json.Number("5.25"), 5.25, true},
		{"unparseable json.Number", json.Number("nope"), 0, false},
		{"string", "6", 0, false},
		{"bool", true, 0, false},
		{"nil", nil, 0, false},
		{"object", map[string]any{}, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := Numeric(c.value)
			if ok != c.ok || got != c.want {
				t.Errorf("Numeric(%#v) = %v, %v; want %v, %v", c.value, got, ok, c.want, c.ok)
			}
		})
	}
}

func TestFormatNumber(t *testing.T) {
	cases := []struct {
		number float64
		want   string
	}{
		{0, "0"},
		{1, "1"},
		{-1, "-1"},
		{2147483648, "2147483648"},
		{1e9, "1000000000"},
		{1.5, "1.5"},
		{-0.25, "-0.25"},
	}
	for _, c := range cases {
		if got := FormatNumber(c.number); got != c.want {
			t.Errorf("FormatNumber(%v) = %q, want %q", c.number, got, c.want)
		}
	}
}

// TestFormatNumberBeyondInt64 pins the bound that the pre-extraction round-trip comparison got
// wrong: an out-of-range float-to-int64 conversion is implementation-defined, so 1e30 must be
// formatted as a float rather than through int64.
func TestFormatNumberBeyondInt64(t *testing.T) {
	for _, number := range []float64{1e30, -1e30, math.MaxFloat64} {
		got := FormatNumber(number)
		if got == "" || got[0] == '-' && number > 0 {
			t.Errorf("FormatNumber(%v) = %q", number, got)
		}
		if got == "-9223372036854775808" || got == "9223372036854775807" {
			t.Errorf("FormatNumber(%v) = %q, want the value's own digits", number, got)
		}
	}
}

// TestIsWhole pins that whole-ness is a property of the value and carries no int64 bound: 1e30 is a
// whole number, and the round-trip comparison this replaced called it fractional on amd64, which is
// how an "integer"-typed 1e30 came to be reported as "expected an integer, found a whole number".
func TestIsWhole(t *testing.T) {
	cases := []struct {
		number float64
		want   bool
	}{
		{0, true},
		{3, true},
		{-3, true},
		{3.5, false},
		{-0.25, false},
		{1e30, true},
		{-1e30, true},
		{math.MaxFloat64, true},
		{math.Inf(1), false},
		{math.Inf(-1), false},
		{math.NaN(), false},
	}
	for _, c := range cases {
		if got := IsWhole(c.number); got != c.want {
			t.Errorf("IsWhole(%v) = %v, want %v", c.number, got, c.want)
		}
	}
}

func TestFormatNumberNonFinite(t *testing.T) {
	for _, number := range []float64{math.Inf(1), math.Inf(-1), math.NaN()} {
		if got := FormatNumber(number); got == "" {
			t.Errorf("FormatNumber(%v) returned empty", number)
		}
	}
}

func TestDescribe(t *testing.T) {
	cases := []struct {
		name  string
		value any
		want  string
	}{
		{"bool", true, "a boolean"},
		{"string", "x", "a string"},
		{"array", []any{1}, "an array"},
		{"object", map[string]any{"a": 1}, "an object"},
		{"null", nil, "null"},
		{"whole", float64(7), "a whole number"},
		{"fractional", 7.5, "a fractional number"},
		{"beyond int64 is still whole", 1e30, "a whole number"},
		{"unexpected", struct{}{}, "an unexpected value"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Describe(c.value); got != c.want {
				t.Errorf("Describe(%#v) = %q, want %q", c.value, got, c.want)
			}
		})
	}
}

func TestArticle(t *testing.T) {
	cases := map[string]string{
		"integer":    "an integer",
		"array":      "an array",
		"any":        "an any",
		"object":     "an object",
		"boolean":    "a boolean",
		"string":     "a string",
		"dictionary": "a dictionary",
		"real":       "a real",
		"number":     "a number",
		"null":       "a null",
		"":           "",
	}
	for name, want := range cases {
		if got := Article(name); got != want {
			t.Errorf("Article(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestRender(t *testing.T) {
	cases := []struct {
		name  string
		value any
		want  string
	}{
		{"string is quoted", "sonnet", `"sonnet"`},
		{"string with quote is escaped", `a"b`, `"a\"b"`},
		{"bool", false, "false"},
		{"null", nil, "null"},
		{"whole number", float64(42), "42"},
		{"fractional number", 0.5, "0.5"},
		{"json.Number", json.Number("42"), "42"},
		{"object falls back to its type", map[string]any{"a": 1}, "an object"},
		{"array falls back to its type", []any{1, 2}, "an array"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Render(c.value); got != c.want {
				t.Errorf("Render(%#v) = %q, want %q", c.value, got, c.want)
			}
		})
	}
}
