// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package service_discovery_enrollment

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildServiceDiscoveryEnrollmentInput(t *testing.T) {
	models := []wellKnownSettingModel{
		{ServerUUID: types.StringValue("UUID-A"), EnrollmentType: types.StringValue(enrollmentTypeMDMADDE), OrgName: types.StringValue("Acme")},
		{ServerUUID: types.StringValue("UUID-B"), EnrollmentType: types.StringValue(enrollmentTypeNone), OrgName: types.StringNull()},
	}

	got := buildServiceDiscoveryEnrollmentInput(models)
	if len(got.WellKnownSettings) != 2 {
		t.Fatalf("want 2 rows, got %d", len(got.WellKnownSettings))
	}
	if got.WellKnownSettings[0].ServerUUID != "UUID-A" || got.WellKnownSettings[0].EnrollmentType != enrollmentTypeMDMADDE {
		t.Errorf("row 0 = %+v", got.WellKnownSettings[0])
	}
	// org_name is a read-only echo and must never be sent.
	if got.WellKnownSettings[0].OrgName != nil {
		t.Errorf("org_name must not be sent, got %v", *got.WellKnownSettings[0].OrgName)
	}
}

func TestBuildServiceDiscoveryEnrollmentInput_Empty(t *testing.T) {
	got := buildServiceDiscoveryEnrollmentInput(nil)
	if got.WellKnownSettings == nil {
		t.Fatalf("want non-nil empty slice (the wire-accepted no-op), got nil")
	}
	if len(got.WellKnownSettings) != 0 {
		t.Errorf("want 0 rows, got %d", len(got.WellKnownSettings))
	}
}

func TestWellKnownSettingsFromList_NullOrUnknown(t *testing.T) {
	ctx := context.Background()
	for _, l := range []types.List{
		types.ListNull(wellKnownSettingObjectType()),
		types.ListUnknown(wellKnownSettingObjectType()),
	} {
		got, diags := wellKnownSettingsFromList(ctx, l)
		if diags.HasError() {
			t.Fatalf("diags: %v", diags)
		}
		if got == nil || len(got) != 0 {
			t.Errorf("want non-nil empty slice, got %#v", got)
		}
	}
}
