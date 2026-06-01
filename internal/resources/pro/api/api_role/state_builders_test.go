// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_role

import (
	"context"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
)

func TestAssignApiRoleResourceModel(t *testing.T) {
	ctx := context.Background()
	role := &pro.ApiRole{
		ID:          "42",
		DisplayName: "Example Role",
		Privileges:  []string{"Create API Roles", "Read Computers"},
	}

	var state ApiRoleResourceModel
	if diags := assignApiRoleResourceModel(ctx, &state, role); diags.HasError() {
		t.Fatalf("assign diags: %v", diags)
	}

	if state.ID.ValueString() != "42" {
		t.Errorf("ID = %q, want %q", state.ID.ValueString(), "42")
	}
	if state.DisplayName.ValueString() != "Example Role" {
		t.Errorf("DisplayName = %q", state.DisplayName.ValueString())
	}
	if state.Privileges.IsNull() || len(state.Privileges.Elements()) != 2 {
		t.Errorf("expected 2 privileges, got %v", state.Privileges)
	}
}

func TestAssignApiRoleResourceModel_EmptyPrivileges(t *testing.T) {
	ctx := context.Background()
	role := &pro.ApiRole{ID: "1", DisplayName: "Empty", Privileges: []string{}}

	var state ApiRoleResourceModel
	if diags := assignApiRoleResourceModel(ctx, &state, role); diags.HasError() {
		t.Fatalf("assign diags: %v", diags)
	}
	if state.Privileges.IsNull() {
		t.Errorf("privileges should be an empty set, not null")
	}
	if len(state.Privileges.Elements()) != 0 {
		t.Errorf("expected 0 privileges, got %d", len(state.Privileges.Elements()))
	}
}
