// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildGroupCreateInput(t *testing.T) {
	got := buildGroupCreateInput(DeviceGroupResourceModel{
		ID:   types.StringValue("ignored-on-create"),
		Name: types.StringValue("Executives"),
	})

	if got.Name != "Executives" {
		t.Errorf("Name = %q, want %q", got.Name, "Executives")
	}
}

func TestBuildGroupUpdateInput(t *testing.T) {
	got := buildGroupUpdateInput(DeviceGroupResourceModel{
		ID:   types.StringValue("abc"),
		Name: types.StringValue("Executives EMEA"),
	})

	if got.Name != "Executives EMEA" {
		t.Errorf("Name = %q, want %q", got.Name, "Executives EMEA")
	}
}

// TestBuildGroupInput_SendsNameVerbatim pins that neither builder normalises the
// name. Trimming here would be the wrong fix for the server's silent trim: state
// would still disagree with config and Terraform would still fail the apply. The
// plan-time validator in validators.go is what stops it reaching this point.
func TestBuildGroupInput_SendsNameVerbatim(t *testing.T) {
	plan := DeviceGroupResourceModel{Name: types.StringValue("  padded  ")}

	if got := buildGroupCreateInput(plan).Name; got != "  padded  " {
		t.Errorf("create Name = %q, want the authored value untouched", got)
	}
	if got := buildGroupUpdateInput(plan).Name; got != "  padded  " {
		t.Errorf("update Name = %q, want the authored value untouched", got)
	}
}
