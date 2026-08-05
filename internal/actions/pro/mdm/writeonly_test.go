// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mdmactions

import (
	"context"
	"maps"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
)

// allMdmActions is every action this package registers.
func allMdmActions() map[string]action.Action {
	all := map[string]action.Action{
		"clear_passcode":          NewClearPasscodeAction(),
		"set_auto_admin_password": NewSetAutoAdminPasswordAction(),
		"flush_mdm_commands":      NewFlushMdmCommandsAction(),
		"renew_mdm_profile":       NewRenewMdmProfileAction(),
	}
	maps.Copy(all, batchActions())
	return all
}

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
	for name, a := range allMdmActions() {
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

// TestSecretAttributes_DiscloseVisibility pairs with the above: since neither
// WriteOnly nor Sensitive is available, the only remaining protection is telling
// the user the value is visible. Assert the attributes carrying user-supplied
// secrets actually say so, because that disclosure is now the whole mitigation.
func TestSecretAttributes_DiscloseVisibility(t *testing.T) {
	secretAttrs := map[string]string{
		"device_lock":             "pin",
		"clear_passcode":          "unlock_token",
		"set_auto_admin_password": "password",
	}

	all := allMdmActions()
	for actionName, attrName := range secretAttrs {
		a, ok := all[actionName]
		if !ok {
			t.Fatalf("%s not registered in allMdmActions", actionName)
		}
		var resp action.SchemaResponse
		a.Schema(context.Background(), action.SchemaRequest{}, &resp)

		attr, ok := resp.Schema.Attributes[attrName]
		if !ok {
			t.Errorf("%s: missing attribute %q", actionName, attrName)
			continue
		}
		if !strings.Contains(attr.GetMarkdownDescription(), "appears in Terraform plan output") {
			t.Errorf("%s: %s carries a user-supplied secret but does not disclose that its value is visible in plan output", actionName, attrName)
		}
	}
}
