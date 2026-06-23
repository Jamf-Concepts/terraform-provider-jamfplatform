// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestRenameScriptParameterKeys verifies the v0→v1 key rename rewrites every
// parameterN key under scripts.scripts[*] to parameter_N, leaves all other
// state untouched, and is a no-op when the scripts block is absent or null.
func TestRenameScriptParameterKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string // canonical-compared as JSON; "" means equal to in
	}{
		{
			name: "renames populated parameters and preserves siblings",
			in: `{
				"id": "123",
				"general": {"name": "p", "frequency": "Ongoing"},
				"scripts": {"scripts": [
					{"id": "9", "priority": "Before", "parameter4": "a", "parameter11": "b"}
				]}
			}`,
			want: `{
				"id": "123",
				"general": {"name": "p", "frequency": "Ongoing"},
				"scripts": {"scripts": [
					{"id": "9", "priority": "Before", "parameter_4": "a", "parameter_11": "b"}
				]}
			}`,
		},
		{
			name: "preserves null-valued parameters as null under the new key",
			in: `{"scripts": {"scripts": [
				{"id": "9", "parameter4": null, "parameter5": "set"}
			]}}`,
			want: `{"scripts": {"scripts": [
				{"id": "9", "parameter_4": null, "parameter_5": "set"}
			]}}`,
		},
		{
			name: "handles multiple script items independently",
			in: `{"scripts": {"scripts": [
				{"id": "1", "parameter4": "x"},
				{"id": "2", "parameter10": "y"}
			]}}`,
			want: `{"scripts": {"scripts": [
				{"id": "1", "parameter_4": "x"},
				{"id": "2", "parameter_10": "y"}
			]}}`,
		},
		{
			name: "no-op when scripts is null",
			in:   `{"id": "1", "scripts": null}`,
			want: `{"id": "1", "scripts": null}`,
		},
		{
			name: "no-op when scripts.scripts is null",
			in:   `{"scripts": {"scripts": null}}`,
			want: `{"scripts": {"scripts": null}}`,
		},
		{
			name: "no-op when scripts block absent",
			in:   `{"id": "1", "general": {"name": "p"}}`,
			want: `{"id": "1", "general": {"name": "p"}}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := renameScriptParameterKeys([]byte(tc.in))
			if err != nil {
				t.Fatalf("renameScriptParameterKeys returned error: %s", err)
			}

			var gotVal, wantVal any
			if err := json.Unmarshal(got, &gotVal); err != nil {
				t.Fatalf("output is not valid JSON: %s", err)
			}
			if err := json.Unmarshal([]byte(tc.want), &wantVal); err != nil {
				t.Fatalf("want fixture is not valid JSON: %s", err)
			}
			if !reflect.DeepEqual(gotVal, wantVal) {
				t.Errorf("rewritten state mismatch:\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}
