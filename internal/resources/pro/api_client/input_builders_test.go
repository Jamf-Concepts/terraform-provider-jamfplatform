// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_client

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func apiRolesSet(t *testing.T, values ...string) types.Set {
	t.Helper()
	set, diags := types.SetValueFrom(context.Background(), types.StringType, values)
	if diags.HasError() {
		t.Fatalf("set build diags: %v", diags)
	}
	return set
}

func TestBuildApiClientInput_KnownValues(t *testing.T) {
	ctx := context.Background()
	plan := ApiClientResourceModel{
		DisplayName:                types.StringValue("My Client"),
		ApiRoles:                   apiRolesSet(t, "Role A", "Role B"),
		Enabled:                    types.BoolValue(true),
		AccessTokenLifetimeSeconds: types.Int64Value(600),
	}
	got, diags := buildApiClientInput(ctx, plan)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if got.DisplayName != "My Client" {
		t.Errorf("DisplayName = %q", got.DisplayName)
	}
	if len(got.AuthorizationScopes) != 2 {
		t.Errorf("AuthorizationScopes len = %d, want 2", len(got.AuthorizationScopes))
	}
	if got.Enabled == nil || !*got.Enabled {
		t.Errorf("Enabled should be &true")
	}
	if got.AccessTokenLifetimeSeconds == nil || *got.AccessTokenLifetimeSeconds != 600 {
		t.Errorf("AccessTokenLifetimeSeconds should be &600")
	}
}

func TestBuildApiClientInput_UnknownOptionalsSendNil(t *testing.T) {
	ctx := context.Background()
	plan := ApiClientResourceModel{
		DisplayName:                types.StringValue("My Client"),
		ApiRoles:                   apiRolesSet(t, "Role A"),
		Enabled:                    types.BoolUnknown(),
		AccessTokenLifetimeSeconds: types.Int64Unknown(),
	}
	got, diags := buildApiClientInput(ctx, plan)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if got.Enabled != nil {
		t.Errorf("Enabled should be nil when unknown so the server default applies")
	}
	if got.AccessTokenLifetimeSeconds != nil {
		t.Errorf("AccessTokenLifetimeSeconds should be nil when unknown so the server default applies")
	}
}
