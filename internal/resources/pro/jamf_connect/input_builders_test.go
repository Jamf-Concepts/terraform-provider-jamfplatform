// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package jamf_connect

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildJamfConnectInput_NoneOmitsVersion(t *testing.T) {
	plan := JamfConnectResourceModel{
		AutoDeploymentType: types.StringValue(autoDeploymentNone),
		Version:            types.StringNull(),
	}
	got := buildJamfConnectInput(plan)
	if got.AutoDeploymentType == nil || *got.AutoDeploymentType != autoDeploymentNone {
		t.Fatalf("autoDeploymentType = %v, want NONE", got.AutoDeploymentType)
	}
	if got.Version != nil {
		t.Errorf("version must be omitted when NONE, got %q", *got.Version)
	}
}

func TestBuildJamfConnectInput_NonNoneEmitsVersion(t *testing.T) {
	plan := JamfConnectResourceModel{
		AutoDeploymentType: types.StringValue(autoDeploymentPatch),
		Version:            types.StringValue("2.45.1"),
	}
	got := buildJamfConnectInput(plan)
	if got.AutoDeploymentType == nil || *got.AutoDeploymentType != autoDeploymentPatch {
		t.Fatalf("autoDeploymentType = %v, want PATCH_UPDATES", got.AutoDeploymentType)
	}
	if got.Version == nil || *got.Version != "2.45.1" {
		t.Errorf("version = %v, want 2.45.1", got.Version)
	}
	// Read-only fields are never sent.
	if got.ProfileName != nil || got.ScopeDescription != nil || got.SiteID != nil {
		t.Errorf("read-only fields must not be populated in the write payload")
	}
}
