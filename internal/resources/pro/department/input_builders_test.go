// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package department

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildDepartmentInput_PopulatesName(t *testing.T) {
	plan := DepartmentResourceModel{
		Name: types.StringValue("Engineering"),
	}

	got := buildDepartmentInput(plan)

	if got.Name != "Engineering" {
		t.Errorf("expected Name %q, got %q", "Engineering", got.Name)
	}
}
