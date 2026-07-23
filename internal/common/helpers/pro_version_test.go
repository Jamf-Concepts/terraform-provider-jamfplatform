// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package helpers

import (
	"strings"
	"testing"
)

func TestRequireMinJamfProVersion(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		required string
		wantErr  bool
		errSub   string
	}{
		{"empty required skips", "11.5.0", "", false, ""},
		{"equal", "11.5.0", "11.5.0", false, ""},
		{"actual greater", "11.6.0", "11.5.0", false, ""},
		{"actual greater patch", "11.5.1", "11.5.0", false, ""},
		{"actual less minor", "11.4.0", "11.5.0", true, "below resource minimum"},
		{"actual less major", "10.9.0", "11.0.0", true, "below resource minimum"},
		{"actual with build suffix tolerated", "11.5.0-t1700000000", "11.5.0", false, ""},
		{"actual with plus suffix tolerated", "11.5.0+build", "11.5.0", false, ""},
		{"actual unparseable", "garbage", "11.5.0", true, "Unparseable"},
		{"required unparseable", "11.5.0", "garbage", true, "Invalid resource minimum"},
		{"actual empty", "", "11.5.0", true, "Unparseable"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			diags := RequireMinJamfProVersion(tc.actual, tc.required, "jamfplatform_pro_test")
			gotErr := diags.HasError()
			if gotErr != tc.wantErr {
				t.Fatalf("HasError = %v, want %v; diags=%v", gotErr, tc.wantErr, diags)
			}
			if tc.wantErr && tc.errSub != "" {
				found := false
				for _, d := range diags {
					if strings.Contains(d.Summary(), tc.errSub) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected summary substring %q in %v", tc.errSub, diags)
				}
			}
		})
	}
}

func TestAtLeastJamfProVersion(t *testing.T) {
	tests := []struct {
		name   string
		actual string
		min    string
		want   bool
	}{
		{"above min", "11.29.0", "11.28.0", true},
		{"at min", "11.29.0", "11.29.0", true},
		{"below min", "11.28.0", "11.29.0", false},
		{"patch below", "11.29.0", "11.29.1", false},
		{"build suffix at min", "11.29.0-t1700000000", "11.29.0", true},
		{"actual unparseable fails open", "garbage", "11.29.0", true},
		{"min unparseable fails open", "11.28.0", "garbage", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := AtLeastJamfProVersion(tc.actual, tc.min); got != tc.want {
				t.Fatalf("AtLeastJamfProVersion(%q, %q) = %v, want %v", tc.actual, tc.min, got, tc.want)
			}
		})
	}
}

// TestJamfProVersionInRange pins the half-open window semantics against the
// Jamf-group member-of workaround window [11.29.0, 11.30.1): the 11.29 regression,
// fixed in 11.30.1 (PI-1394).
func TestJamfProVersionInRange(t *testing.T) {
	tests := []struct {
		name string
		lo   string
		hi   string
		in   string
		want bool
	}{
		{"at lower bound (inclusive)", "11.29.0", "11.30.1", "11.29.0", true},
		{"inside window (minor)", "11.29.0", "11.30.1", "11.29.5", true},
		{"just below upper bound", "11.29.0", "11.30.1", "11.30.0", true},
		{"at upper bound (exclusive) -> fixed", "11.29.0", "11.30.1", "11.30.1", false},
		{"above upper bound -> fixed", "11.29.0", "11.30.1", "11.31.0", false},
		{"below lower bound -> pre-regression", "11.29.0", "11.30.1", "11.28.0", false},
		{"build suffix stripped, at lower bound", "11.29.0", "11.30.1", "11.29.0-t1700000000", true},
		{"build suffix stripped, at fixed version", "11.29.0", "11.30.1", "11.30.1-t1784555528405", false},
		{"actual unparseable fails open", "11.29.0", "11.30.1", "garbage", true},
		{"lower bound unparseable fails open", "garbage", "11.30.1", "11.30.1", true},
		{"upper bound unparseable fails open", "11.29.0", "garbage", "11.30.1", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := JamfProVersionInRange(tc.in, tc.lo, tc.hi); got != tc.want {
				t.Fatalf("JamfProVersionInRange(%q, %q, %q) = %v, want %v", tc.in, tc.lo, tc.hi, got, tc.want)
			}
		})
	}
}

func TestWarnIfBelowProviderFloor(t *testing.T) {
	tests := []struct {
		name    string
		actual  string
		floor   string
		wantNil bool
	}{
		{"floor empty", "11.5.0", "", true},
		{"at floor", "11.5.0", "11.5.0", true},
		{"above floor", "12.0.0", "11.5.0", true},
		{"below floor", "10.0.0", "11.5.0", false},
		{"actual unparseable returns warning", "garbage", "11.0.0", false},
		{"actual build suffix at floor", "11.0.0-t1700000000", "11.0.0", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := WarnIfBelowProviderFloor(tc.actual, tc.floor)
			gotNil := d == nil
			if gotNil != tc.wantNil {
				t.Fatalf("got nil = %v, want nil = %v (diagnostic = %v)", gotNil, tc.wantNil, d)
			}
		})
	}
}
