// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package jamf_protect

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestBuildJamfProtectRegistrationInput pins the WriteOnly plumbing: the
// plaintext password is sourced from the Config model (the framework
// nullifies WriteOnly attributes in the plan), while api_url and client_id
// come from the plan, verbatim.
func TestBuildJamfProtectRegistrationInput(t *testing.T) {
	plan := JamfProtectResourceModel{
		APIURL:   types.StringValue("https://example.protect.jamfcloud.com/graphql"),
		ClientID: types.StringValue("protect-client-id"),
		Password: types.StringNull(), // WriteOnly: null in plan
	}
	cfg := JamfProtectResourceModel{
		Password: types.StringValue("hunter2"),
	}

	got := buildJamfProtectRegistrationInput(plan, cfg)

	if got.ProtectURL != "https://example.protect.jamfcloud.com/graphql" {
		t.Errorf("ProtectURL = %q", got.ProtectURL)
	}
	if got.ClientID != "protect-client-id" {
		t.Errorf("ClientID = %q", got.ClientID)
	}
	if got.Password != "hunter2" {
		t.Errorf("Password = %q, want the Config-sourced plaintext", got.Password)
	}
}

func TestBuildJamfProtectSettingsInput(t *testing.T) {
	if got := buildJamfProtectSettingsInput(nil); got != nil {
		t.Errorf("nil auto_install must produce a nil request, got %+v", got)
	}

	tr := true
	if got := buildJamfProtectSettingsInput(&tr); got == nil || got.AutoInstall == nil || !*got.AutoInstall {
		t.Errorf("true pointer mismapped: %+v", got)
	}

	fa := false
	if got := buildJamfProtectSettingsInput(&fa); got == nil || got.AutoInstall == nil || *got.AutoInstall {
		t.Errorf("false pointer mismapped: %+v", got)
	}
}
