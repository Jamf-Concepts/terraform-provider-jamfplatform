// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestEncodeNoExecuteTime pins the arithmetic against the wire observations in
// no_execute.go. Every `want` below was verified live: the value is what Jamf
// Pro 11.31.1 must be sent for the `in` column to come back from a GET.
func TestEncodeNoExecuteTime(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ in, want string }{
		// Midnight is hour 0, so it encodes to 48 — and it is the case a
		// 24-hour offset gets wrong, because 24:00 is exactly 1440 minutes and
		// the server stores only above that.
		{"12:00 AM", "48:00 AM"},
		{"12:01 AM", "48:01 AM"},
		{"12:59 AM", "48:59 AM"},
		{"1:00 AM", "49:00 AM"},
		{"11:59 AM", "59:59 AM"},
		// Noon stays hour 12 rather than becoming 24: the other end of the
		// 12-hour conversion, and the other easy mistake.
		{"12:00 PM", "60:00 AM"},
		{"12:30 PM", "60:30 AM"},
		{"1:00 PM", "61:00 AM"},
		{"5:15 PM", "65:15 AM"},
		{"11:59 PM", "71:59 AM"},
		// A value outside the wire shape is passed through untouched rather
		// than mangled — the schema validator makes it unreachable from
		// configuration, but a total function is easier to reason about.
		{"", ""},
		{"25:00 AM", "25:00 AM"},
		{"1:00", "1:00"},
		{"nonsense", "nonsense"},
	} {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := encodeNoExecuteTime(tc.in); got != tc.want {
				t.Errorf("encodeNoExecuteTime(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestEncodeNoExecuteTime_AlwaysClearsTheThreshold is the property the whole
// encoding rests on: every minute of the day must encode to a total strictly
// greater than 1440, because that is the condition under which Jamf Pro stores
// the value at all. A 24-hour offset would fail this at exactly midnight.
func TestEncodeNoExecuteTime_AlwaysClearsTheThreshold(t *testing.T) {
	t.Parallel()

	for h := range 24 {
		for _, m := range []int{0, 1, 30, 59} {
			in := renderTwelveHour(h, m)
			encoded := encodeNoExecuteTime(in)
			hour, minute, ok := parseEncoded(encoded)
			if !ok {
				t.Fatalf("encodeNoExecuteTime(%q) = %q, which is not h:MM AM", in, encoded)
			}
			total := hour*60 + minute
			if total <= 1440 {
				t.Errorf("encodeNoExecuteTime(%q) = %q = %d minutes, which Jamf Pro discards (needs > 1440)", in, encoded, total)
			}
			// And it must wrap back to the minute of day it came from.
			if got := total % 1440; got != h*60+m {
				t.Errorf("encodeNoExecuteTime(%q) = %q wraps to minute %d, want %d", in, encoded, got, h*60+m)
			}
		}
	}
}

// TestNoExecuteTimePointer covers the omission contract. Omitting the element
// is what preserves a window configured outside Terraform, so a null or unknown
// attribute must produce nil rather than an empty string.
func TestNoExecuteTimePointer(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		in   types.String
		want *string
	}{
		{"null omits the element", types.StringNull(), nil},
		{"unknown omits the element", types.StringUnknown(), nil},
		{"empty omits the element", types.StringValue(""), nil},
		{"a set value is encoded", types.StringValue("5:15 PM"), new("65:15 AM")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := noExecuteTimePointer(tc.in)
			switch {
			case tc.want == nil && got != nil:
				t.Errorf("want nil, got %q", *got)
			case tc.want != nil && got == nil:
				t.Errorf("want %q, got nil", *tc.want)
			case tc.want != nil && *got != *tc.want:
				t.Errorf("want %q, got %q", *tc.want, *got)
			}
		})
	}
}

// renderTwelveHour turns a 24-hour hour and a minute into the `h:MM AM` form
// the schema accepts.
//
//go:fix inline
func renderTwelveHour(h, m int) string {
	meridiem := "AM"
	if h >= 12 {
		meridiem = "PM"
	}
	display := h % 12
	if display == 0 {
		display = 12
	}
	return fmt.Sprintf("%d:%02d %s", display, m, meridiem)
}

// parseEncoded reads back the `H:MM AM` form encodeNoExecuteTime emits, where
// the hour may exceed 24.
func parseEncoded(s string) (hour, minute int, ok bool) {
	var meridiem string
	n, err := fmt.Sscanf(s, "%d:%d %s", &hour, &minute, &meridiem)
	if err != nil || n != 3 || meridiem != "AM" {
		return 0, 0, false
	}
	return hour, minute, true
}
