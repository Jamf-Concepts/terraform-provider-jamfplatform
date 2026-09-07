// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
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

// TestDropPrintersLeaveExistingDefault pins the v1 → v2 migration. The
// attribute is gone from the schema, so a surplus key in prior state fails the
// decode outright — and the migration must leave every other value byte-exact,
// because it re-marshals the whole document to get there.
func TestDropPrintersLeaveExistingDefault(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		in   string
		// want is compared as parsed JSON, so key order does not matter.
		want string
	}{
		{
			name: "drops the key and keeps its siblings",
			in:   `{"id":"7029","printers":{"leave_existing_default":true,"printers":[{"id":"12","action":"install","make_default":true}]}}`,
			want: `{"id":"7029","printers":{"printers":[{"id":"12","action":"install","make_default":true}]}}`,
		},
		{
			name: "drops the key when it is the only one, leaving an empty block",
			in:   `{"printers":{"leave_existing_default":false}}`,
			want: `{"printers":{}}`,
		},
		{
			name: "state that never carried the key is untouched",
			in:   `{"printers":{"printers":[{"id":"12"}]}}`,
			want: `{"printers":{"printers":[{"id":"12"}]}}`,
		},
		{
			name: "a null printers block is untouched",
			in:   `{"id":"7029","printers":null}`,
			want: `{"id":"7029","printers":null}`,
		},
		{
			name: "an absent printers block is untouched",
			in:   `{"id":"7029","scripts":{"scripts":[]}}`,
			want: `{"id":"7029","scripts":{"scripts":[]}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := dropPrintersLeaveExistingDefault([]byte(tc.in))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			var gotAny, wantAny any
			if err := json.Unmarshal(got, &gotAny); err != nil {
				t.Fatalf("result is not valid JSON: %s", err)
			}
			if err := json.Unmarshal([]byte(tc.want), &wantAny); err != nil {
				t.Fatalf("fixture is not valid JSON: %s", err)
			}
			if !reflect.DeepEqual(gotAny, wantAny) {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

// TestDropPrintersLeaveExistingDefault_PreservesLargeIntegersVerbatim guards
// the hazard of a whole-document re-marshal: encoding/json turns a number into
// a float64 on the way through a map[string]any, which silently mangles a large
// Jamf Pro ID. The migration only ever unmarshals into json.RawMessage for that
// reason, and this test fails if someone loosens it.
func TestDropPrintersLeaveExistingDefault_PreservesLargeIntegersVerbatim(t *testing.T) {
	t.Parallel()

	const in = `{"printers":{"leave_existing_default":true,"printers":[{"id":9007199254740993}]}}`
	got, err := dropPrintersLeaveExistingDefault([]byte(in))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !strings.Contains(string(got), "9007199254740993") {
		t.Errorf("large integer was reformatted, got %s", got)
	}
	if strings.Contains(string(got), "leave_existing_default") {
		t.Errorf("leave_existing_default survived, got %s", got)
	}
}

// TestUpgradeStateCoversEveryPriorVersion pins that an upgrader exists for
// every schema version below the current one. Forgetting one makes every
// existing workspace fail on `terraform plan` with "state snapshot was created
// by a newer schema version", which no unit test would otherwise catch.
func TestUpgradeStateCoversEveryPriorVersion(t *testing.T) {
	t.Parallel()

	r := &PolicyResource{}
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	current := schemaResp.Schema.Version

	upgraders := r.UpgradeState(context.Background())
	for v := range current {
		if _, ok := upgraders[v]; !ok {
			t.Errorf("no state upgrader registered for schema version %d (current is %d)", v, current)
		}
	}
	if len(upgraders) != int(current) {
		t.Errorf("got %d upgraders for a v%d schema, want %d", len(upgraders), current, current)
	}
}
