// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_software_title

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// State upgraders are unit-tested rather than acceptance-tested: the acceptance
// harness always writes state at the current schema version, so the v0→v1 path
// is unreachable from it. These tests are the only coverage the upgrader gets.

// v0StateJSON is a realistic schema-v0 state object for one title, as written
// by the build that read the classic /patchsoftwaretitles endpoints: it still
// carries the category_name and site_name attributes the v1 schema has no home
// for, alongside every attribute that survived.
const v0StateJSON = `{
  "id": "147",
  "name": "8x8 Work",
  "name_id": "285",
  "source_id": 1,
  "category_id": "58",
  "category_name": "Productivity",
  "site_id": "-1",
  "site_name": "NONE",
  "web_notification": false,
  "email_notification": false,
  "version_packages": {"8.33.2.2": "1"},
  "available_versions": ["8.36.2.3", "8.35.2.6", "8.33.2.2"],
  "accept_extension_attributes": true,
  "extension_attributes": [
    {"ea_id": "jamf-patch-8x8-work", "display_name": "8x8 Work Bundle Version", "accepted": true}
  ],
  "timeouts": null
}`

// TestDropRemovedAttributes_RemovesOnlyTheWithdrawnKeys pins the rewrite at the
// heart of the upgrade: the two withdrawn keys go, and every other key keeps
// its raw JSON value byte for byte once insignificant whitespace is normalised.
// Comparing raw bytes rather than decoded values is the point — the rewrite
// carries each value across as json.RawMessage precisely so nothing is
// reinterpreted, and a comparison through Go types would hide a number
// reformatted, a string re-escaped or an object's keys reordered.
func TestDropRemovedAttributes_RemovesOnlyTheWithdrawnKeys(t *testing.T) {
	var before map[string]json.RawMessage
	if err := json.Unmarshal([]byte(v0StateJSON), &before); err != nil {
		t.Fatalf("v0 fixture is not valid JSON: %s", err)
	}

	got, err := dropRemovedAttributes([]byte(v0StateJSON), removedInV1)
	if err != nil {
		t.Fatalf("dropRemovedAttributes returned error: %s", err)
	}

	var after map[string]json.RawMessage
	if err := json.Unmarshal(got, &after); err != nil {
		t.Fatalf("rewritten state is not valid JSON: %s", err)
	}

	for _, gone := range removedInV1 {
		if _, ok := after[gone]; ok {
			t.Errorf("%q must be removed from upgraded state", gone)
		}
	}
	if len(after) != len(before)-len(removedInV1) {
		t.Errorf("expected %d keys after the rewrite, got %d", len(before)-len(removedInV1), len(after))
	}
	for k, want := range before {
		if slices.Contains(removedInV1, k) {
			continue
		}
		gotVal, ok := after[k]
		if !ok {
			t.Errorf("%q was dropped but only category_name and site_name should be", k)
			continue
		}
		if compactJSON(t, gotVal) != compactJSON(t, want) {
			t.Errorf("%q value changed: want %s, got %s", k, want, gotVal)
		}
	}
}

// compactJSON strips the insignificant whitespace a fixture carries for
// readability, so a value comparison is byte-exact on everything that matters:
// number formatting, string escaping and key order all survive it.
func compactJSON(t *testing.T, in []byte) string {
	t.Helper()
	var buf bytes.Buffer
	if err := json.Compact(&buf, in); err != nil {
		t.Fatalf("compact %s: %s", in, err)
	}
	return buf.String()
}

// TestDropRemovedAttributes_AbsentKeysAreNotAnError pins that state written by
// a build which never set the withdrawn attributes still upgrades. Terraform
// omits a null attribute from state, so most real v0 states carry neither key,
// and treating that as a failure would block the upgrade for the common case.
func TestDropRemovedAttributes_AbsentKeysAreNotAnError(t *testing.T) {
	in := `{"id": "147", "name": "8x8 Work"}`

	got, err := dropRemovedAttributes([]byte(in), removedInV1)
	if err != nil {
		t.Fatalf("dropRemovedAttributes returned error: %s", err)
	}

	var after map[string]json.RawMessage
	if err := json.Unmarshal(got, &after); err != nil {
		t.Fatalf("rewritten state is not valid JSON: %s", err)
	}
	if len(after) != 2 || string(after["id"]) != `"147"` {
		t.Errorf("expected the two surviving keys unchanged, got %s", got)
	}
}

// TestDropRemovedAttributes_RejectsMalformedState pins that a state body which
// is not a JSON object is reported rather than silently replaced with an empty
// one, which would destroy the resource's state.
func TestDropRemovedAttributes_RejectsMalformedState(t *testing.T) {
	if _, err := dropRemovedAttributes([]byte(`["not", "an", "object"]`), removedInV1); err == nil {
		t.Error("expected an error for a non-object state body")
	}
}

// TestUpgradeState_V0StateDecodesAtV1 drives the registered upgrader end to end
// over realistic v0 state and pins that it produces a value against the current
// schema with no diagnostics. This is the whole reason the upgrader exists: the
// framework's raw-state decode rejects an attribute the current schema does not
// declare, so without the rewrite every pre-migration state file fails to load
// with a framework error rather than a plannable diff.
func TestUpgradeState_V0StateDecodesAtV1(t *testing.T) {
	ctx := context.Background()
	r := NewPatchSoftwareTitleResource().(*PatchSoftwareTitleResource)

	upgraders := r.UpgradeState(ctx)
	u, ok := upgraders[0]
	if !ok {
		t.Fatalf("no upgrader registered for schema version 0, got versions %v", upgraders)
	}
	if u.StateUpgrader == nil {
		t.Fatal("the v0 entry needs a StateUpgrader — this upgrade rewrites raw state, not a decoded prior schema")
	}

	req := resource.UpgradeStateRequest{RawState: &tfprotov6.RawState{JSON: []byte(v0StateJSON)}}
	var resp resource.UpgradeStateResponse
	u.StateUpgrader(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade diagnostics: %v", resp.Diagnostics)
	}
	if resp.DynamicValue == nil {
		t.Fatal("expected an upgraded DynamicValue, got nil")
	}
}

// TestUpgradeState_RawV0StateIsUndecodableWithoutTheRewrite pins the failure the
// upgrader is preventing, so the test above cannot pass vacuously: raw v0 state
// still carrying category_name is refused by the framework's own decode against
// the v1 schema type. If this ever stops erroring the upgrader has become
// unnecessary — and if the rewrite regresses, this is the error users would see.
func TestUpgradeState_RawV0StateIsUndecodableWithoutTheRewrite(t *testing.T) {
	ctx := context.Background()
	r := NewPatchSoftwareTitleResource().(*PatchSoftwareTitleResource)

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	raw := tfprotov6.RawState{JSON: []byte(v0StateJSON)}
	if _, err := raw.Unmarshal(schemaType); err == nil {
		t.Error("expected v0 state carrying the withdrawn attributes to be undecodable against the v1 schema")
	}
}

// TestUpgradeState_NilRawStateIsNoop pins that a missing prior state produces
// neither a value nor an error. The framework treats an empty response as
// "nothing to upgrade", whereas an error would fail an operation on a resource
// that has no state to migrate in the first place.
func TestUpgradeState_NilRawStateIsNoop(t *testing.T) {
	ctx := context.Background()
	r := NewPatchSoftwareTitleResource().(*PatchSoftwareTitleResource)

	var resp resource.UpgradeStateResponse
	r.UpgradeState(ctx)[0].StateUpgrader(ctx, resource.UpgradeStateRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("a nil prior state must not raise diagnostics, got %v", resp.Diagnostics)
	}
	if resp.DynamicValue != nil {
		t.Error("a nil prior state must not produce an upgraded value")
	}
}

// TestUpgradeState_MalformedStateRaisesADiagnostic pins that an undecodable
// state body surfaces as a Terraform diagnostic naming the resource rather than
// a panic or a silently empty upgrade.
func TestUpgradeState_MalformedStateRaisesADiagnostic(t *testing.T) {
	ctx := context.Background()
	r := NewPatchSoftwareTitleResource().(*PatchSoftwareTitleResource)

	req := resource.UpgradeStateRequest{RawState: &tfprotov6.RawState{JSON: []byte(`{"id":`)}}
	var resp resource.UpgradeStateResponse
	r.UpgradeState(ctx)[0].StateUpgrader(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected an error diagnostic for malformed prior state")
	}
	if resp.DynamicValue != nil {
		t.Error("a failed upgrade must not produce a value")
	}
}
