// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_client

import (
	"context"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAssignApiClientServerFields(t *testing.T) {
	ctx := context.Background()
	resp := &pro.ApiIntegrationResponse{
		ID:                         88,
		DisplayName:                "My Client",
		AuthorizationScopes:        []string{"Role A", "Role B"},
		Enabled:                    true,
		AccessTokenLifetimeSeconds: 300,
		ClientID:                   "abc-123",
		AppType:                    "CLIENT_CREDENTIALS",
	}
	var state ApiClientResourceModel
	if diags := assignApiClientServerFields(ctx, &state, resp); diags.HasError() {
		t.Fatalf("assign diags: %v", diags)
	}
	if state.ID.ValueString() != "88" {
		t.Errorf("ID = %q, want 88 (int rendered as string)", state.ID.ValueString())
	}
	if state.ClientID.ValueString() != "abc-123" {
		t.Errorf("ClientID = %q", state.ClientID.ValueString())
	}
	if state.AppType.ValueString() != "CLIENT_CREDENTIALS" {
		t.Errorf("AppType = %q", state.AppType.ValueString())
	}
	if state.AccessTokenLifetimeSeconds.ValueInt64() != 300 {
		t.Errorf("lifetime = %d", state.AccessTokenLifetimeSeconds.ValueInt64())
	}
	if len(state.ApiRoles.Elements()) != 2 {
		t.Errorf("api_roles len = %d, want 2", len(state.ApiRoles.Elements()))
	}
	// assign must not touch the secret or the rotation trigger.
	if !state.ClientSecret.IsNull() {
		t.Errorf("assign must not set client_secret")
	}
}

func TestResolveStoredSecret(t *testing.T) {
	prior := types.StringValue("s3cr3t")

	if got := resolveStoredSecret("NONE", prior); !got.IsNull() {
		t.Errorf("app_type NONE must clear the secret, got %v", got)
	}
	if got := resolveStoredSecret("CLIENT_CREDENTIALS", prior); got.ValueString() != "s3cr3t" {
		t.Errorf("non-NONE must carry the prior secret, got %v", got)
	}
	if got := resolveStoredSecret("CLIENT_CREDENTIALS", types.StringNull()); !got.IsNull() {
		t.Errorf("null prior stays null, got %v", got)
	}
}

func TestIDString(t *testing.T) {
	if idString(88) != "88" {
		t.Errorf("idString(88) = %q, want 88", idString(88))
	}
}
