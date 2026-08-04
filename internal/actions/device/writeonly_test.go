// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package deviceactions

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
)

// TestActionAttributes_AreNotWriteOnly is a regression guard with teeth.
//
// WriteOnly exists on action attributes and compiles, but the framework hardcodes
// WriteOnlyAttributesAllowed: false for action config validation while still
// applying the resource write-only gate, so ANY non-null value fails with
// "WriteOnly Attribute Not Allowed" — unconditionally, on every Terraform
// version. That breaks silently: nothing fails until someone actually assigns the
// attribute, and the acceptance tests that would notice are device-gated and skip
// in CI. So the invariant is asserted here, in a unit test that always runs.
//
// If a future attribute genuinely needs write-only semantics, verify against a
// real Terraform binary first — see secretAttrNote.
func TestActionAttributes_AreNotWriteOnly(t *testing.T) {
	actions := map[string]action.Action{
		"erase":    NewEraseAction(),
		"restart":  NewRestartAction(),
		"shutdown": NewShutdownAction(),
		"unmanage": NewUnmanageAction(),
	}

	for name, a := range actions {
		var resp action.SchemaResponse
		a.Schema(context.Background(), action.SchemaRequest{}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("%s: schema diagnostics: %v", name, resp.Diagnostics)
		}
		for attrName, attr := range resp.Schema.Attributes {
			if attr.IsWriteOnly() {
				t.Errorf("%s: attribute %q is WriteOnly; Terraform rejects any value set for a write-only action attribute, so the attribute cannot be used at all", name, attrName)
			}
		}
	}
}

// TestErasePin_DisclosesVisibility pairs with the above: with neither WriteOnly
// nor Sensitive available, disclosing that the value is visible is the whole
// remaining mitigation, so assert it is actually disclosed.
func TestErasePin_DisclosesVisibility(t *testing.T) {
	var resp action.SchemaResponse
	NewEraseAction().Schema(context.Background(), action.SchemaRequest{}, &resp)

	attr, ok := resp.Schema.Attributes["pin"]
	if !ok {
		t.Fatal("erase: missing attribute \"pin\"")
	}
	if !strings.Contains(attr.GetMarkdownDescription(), "appears in Terraform plan output") {
		t.Error("erase: pin carries a user-supplied secret but does not disclose that its value is visible in plan output")
	}
}
