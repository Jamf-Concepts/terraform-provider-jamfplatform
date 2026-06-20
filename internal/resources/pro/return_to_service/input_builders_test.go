// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package return_to_service

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildReturnToServiceInput_EmitsBothFields(t *testing.T) {
	plan := ReturnToServiceResourceModel{
		DisplayName:   types.StringValue("Front Desk iPads"),
		WifiProfileID: types.StringValue("70"),
	}

	got := buildReturnToServiceInput(plan)
	if got == nil {
		t.Fatal("expected non-nil request")
	}
	if got.DisplayName == nil || *got.DisplayName != "Front Desk iPads" {
		t.Errorf("displayName = %v, want %q", got.DisplayName, "Front Desk iPads")
	}
	if got.WifiProfileID == nil || *got.WifiProfileID != "70" {
		t.Errorf("wifiProfileId = %v, want %q", got.WifiProfileID, "70")
	}
}
