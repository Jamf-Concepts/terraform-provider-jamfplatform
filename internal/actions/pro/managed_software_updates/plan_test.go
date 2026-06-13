// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package managed_software_updates

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestBuildGroupPlanRequest_RequiredOnly verifies required fields map through and optional
// fields are omitted (nil) when unset.
func TestBuildGroupPlanRequest_RequiredOnly(t *testing.T) {
	req := buildGroupPlanRequest(PlanActionModel{
		GroupID:      types.StringValue("2"),
		ObjectType:   types.StringValue("COMPUTER_GROUP"),
		UpdateAction: types.StringValue("DOWNLOAD_INSTALL"),
		VersionType:  types.StringValue("LATEST_ANY"),
	})

	if req.Group.GroupID != "2" || req.Group.ObjectType != "COMPUTER_GROUP" {
		t.Errorf("group mapped wrong: %+v", req.Group)
	}
	if req.Config.UpdateAction != "DOWNLOAD_INSTALL" || req.Config.VersionType != "LATEST_ANY" {
		t.Errorf("config required fields mapped wrong: %+v", req.Config)
	}
	if req.Config.SpecificVersion != nil || req.Config.BuildVersion != nil ||
		req.Config.ForceInstallLocalDateTime != nil || req.Config.MaxDeferrals != nil {
		t.Errorf("expected unset optionals to be nil, got %+v", req.Config)
	}
}

// TestBuildGroupPlanRequest_AllFields verifies every optional field is forwarded when set.
func TestBuildGroupPlanRequest_AllFields(t *testing.T) {
	req := buildGroupPlanRequest(PlanActionModel{
		GroupID:                   types.StringValue("5"),
		ObjectType:                types.StringValue("MOBILE_DEVICE_GROUP"),
		UpdateAction:              types.StringValue("DOWNLOAD_INSTALL_SCHEDULE"),
		VersionType:               types.StringValue("SPECIFIC_VERSION"),
		SpecificVersion:           types.StringValue("17.5.1"),
		BuildVersion:              types.StringValue("21F90"),
		ForceInstallLocalDateTime: types.StringValue("2026-07-01T09:00:00"),
		MaxDeferrals:              types.Int64Value(3),
	})

	if got := req.Config.SpecificVersion; got == nil || *got != "17.5.1" {
		t.Errorf("specific_version: got %v", got)
	}
	if got := req.Config.BuildVersion; got == nil || *got != "21F90" {
		t.Errorf("build_version: got %v", got)
	}
	if got := req.Config.ForceInstallLocalDateTime; got == nil || *got != "2026-07-01T09:00:00" {
		t.Errorf("force_install_local_date_time: got %v", got)
	}
	if got := req.Config.MaxDeferrals; got == nil || *got != 3 {
		t.Errorf("max_deferrals: got %v", got)
	}
}
