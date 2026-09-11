// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"strings"
	"testing"
)

// firstKey parses a one-key schema file and returns that key, so a case is the YAML under test.
func firstKey(t *testing.T, payloadKeys string) *specKey {
	t.Helper()
	spec, err := parseSpec([]byte(specWith(payloadKeys)))
	if err != nil {
		t.Fatalf("parseSpec: %v", err)
	}
	if len(spec.PayloadKeys) != 1 {
		t.Fatalf("parsed %d keys, want 1", len(spec.PayloadKeys))
	}
	return spec.PayloadKeys[0]
}

// TestParseRangelistKeepsTheDeclaredElementType covers why the rangelist is not stringified: the
// wire distinguishes the integer 14 from the string "14", and a table that stringified one would
// report every valid authored value as out of range.
func TestParseRangelistKeepsTheDeclaredElementType(t *testing.T) {
	tests := []struct {
		name       string
		payloadKey string
		want       []any
	}{
		{
			name:       "integers stay integers",
			payloadKey: "- key: DiffieHellmanGroup\n  type: <integer>\n  rangelist:\n  - 2\n  - 14\n",
			want:       []any{int64(2), int64(14)},
		},
		{
			name:       "strings stay strings",
			payloadKey: "- key: Mode\n  type: <string>\n  rangelist:\n  - Allow\n  - Deny\n",
			want:       []any{"Allow", "Deny"},
		},
		{
			name:       "a quoted digit stays a string",
			payloadKey: "- key: Version\n  type: <string>\n  rangelist:\n  - '14'\n",
			want:       []any{"14"},
		},
		{
			name:       "reals and booleans keep their own types",
			payloadKey: "- key: Mixed\n  type: <any>\n  rangelist:\n  - 1.5\n  - true\n",
			want:       []any{1.5, true},
		},
		{
			name:       "an absent rangelist means unconstrained",
			payloadKey: "- key: Mode\n  type: <string>\n",
			want:       nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			captured := captureWarnings(t)
			got := firstKey(t, test.payloadKey).Rangelist
			if fmt.Sprintf("%#v", got) != fmt.Sprintf("%#v", test.want) {
				t.Errorf("rangelist = %#v, want %#v", got, test.want)
			}
			if captured.Len() != 0 {
				t.Errorf("a readable rangelist reported %q", captured.String())
			}
		})
	}
}

// TestParseRangeReadsEitherBoundAlone covers Apple declaring only one end of a range —
// `LoginFrequency` gives a minimum and no maximum — which is why both bounds are pointers.
func TestParseRangeReadsEitherBoundAlone(t *testing.T) {
	tests := []struct {
		name             string
		payloadKey       string
		wantMin, wantMax string
	}{
		{
			name:       "both bounds",
			payloadKey: "- key: Interval\n  type: <integer>\n  range:\n    min: 1\n    max: 60\n",
			wantMin:    "1",
			wantMax:    "60",
		},
		{
			name:       "a minimum alone leaves the maximum unbounded",
			payloadKey: "- key: LoginFrequency\n  type: <integer>\n  range:\n    min: 0\n",
			wantMin:    "0",
			wantMax:    "unbounded",
		},
		{
			name:       "a maximum alone leaves the minimum unbounded",
			payloadKey: "- key: Ceiling\n  type: <integer>\n  range:\n    max: 10\n",
			wantMin:    "unbounded",
			wantMax:    "10",
		},
		{
			name:       "no range at all",
			payloadKey: "- key: Free\n  type: <integer>\n",
			wantMin:    "unbounded",
			wantMax:    "unbounded",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			captured := captureWarnings(t)
			key := firstKey(t, test.payloadKey)
			if got := formatBound(key.Min); got != test.wantMin {
				t.Errorf("min = %s, want %s", got, test.wantMin)
			}
			if got := formatBound(key.Max); got != test.wantMax {
				t.Errorf("max = %s, want %s", got, test.wantMax)
			}
			if captured.Len() != 0 {
				t.Errorf("a readable range reported %q", captured.String())
			}
		})
	}
}

// TestUnreadableConstraintsAreReported covers the fail-soft paths. Each one degrades a constraint
// Apple declared into no constraint at all, and the degradation is invisible in the emitted table —
// an unread bound looks exactly like a key Apple never bounded — so the regeneration has to say so.
func TestUnreadableConstraintsAreReported(t *testing.T) {
	tests := []struct {
		name       string
		payloadKey string
		wantReport string
		assert     func(t *testing.T, key *specKey)
	}{
		{
			name:       "a bound that is not a number",
			payloadKey: "- key: Interval\n  type: <integer>\n  range:\n    min: unspecified\n    max: 60\n",
			wantReport: `range min is "unspecified"`,
			assert: func(t *testing.T, key *specKey) {
				if key.Min != nil {
					t.Errorf("min = %s, want unbounded", formatBound(key.Min))
				}
				if key.Max == nil || *key.Max != 60 {
					t.Errorf("max = %s, want 60", formatBound(key.Max))
				}
			},
		},
		{
			name:       "a bound in a shape the parser cannot read",
			payloadKey: "- key: Interval\n  type: <integer>\n  range:\n    max:\n    - 1\n    - 60\n",
			wantReport: "range max is not a scalar",
			assert: func(t *testing.T, key *specKey) {
				if key.Max != nil {
					t.Errorf("max = %s, want unbounded", formatBound(key.Max))
				}
			},
		},
		{
			name:       "a rangelist value whose tag contradicts its text",
			payloadKey: "- key: Group\n  type: <integer>\n  rangelist:\n  - !!int notanumber\n",
			wantReport: "is tagged !!int but does not parse as one",
			assert: func(t *testing.T, key *specKey) {
				if want := []any{"notanumber"}; fmt.Sprintf("%#v", key.Rangelist) != fmt.Sprintf("%#v", want) {
					t.Errorf("rangelist = %#v, want %#v", key.Rangelist, want)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			captured := captureWarnings(t)
			key := firstKey(t, test.payloadKey)
			test.assert(t, key)
			if !strings.Contains(captured.String(), test.wantReport) {
				t.Errorf("stderr = %q, want it to contain %q", captured.String(), test.wantReport)
			}
			if !strings.Contains(captured.String(), key.Key) {
				t.Errorf("stderr = %q, want it to name the key %q", captured.String(), key.Key)
			}
		})
	}
}
