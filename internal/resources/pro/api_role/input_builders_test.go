// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_role

import (
	"context"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildApiRoleInput(t *testing.T) {
	ctx := context.Background()
	privs, diags := types.SetValueFrom(ctx, types.StringType, []string{"Read Computers", "Create API Roles"})
	if diags.HasError() {
		t.Fatalf("set build diags: %v", diags)
	}
	plan := ApiRoleResourceModel{
		DisplayName: types.StringValue("Example Role"),
		Privileges:  privs,
	}

	got, diags := buildApiRoleInput(ctx, plan)
	if diags.HasError() {
		t.Fatalf("buildApiRoleInput diags: %v", diags)
	}
	if got.DisplayName != "Example Role" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Example Role")
	}
	want := []string{"Create API Roles", "Read Computers"}
	gotPrivs := append([]string(nil), got.Privileges...)
	sort.Strings(gotPrivs)
	if len(gotPrivs) != len(want) {
		t.Fatalf("privileges len = %d, want %d", len(gotPrivs), len(want))
	}
	for i := range want {
		if gotPrivs[i] != want[i] {
			t.Errorf("privilege[%d] = %q, want %q", i, gotPrivs[i], want[i])
		}
	}
}
