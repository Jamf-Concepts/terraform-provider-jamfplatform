// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_provisioning_profile

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildInput_FullPayload(t *testing.T) {
	plan := ProvisioningProfileResourceModel{
		Name:        types.StringValue("in-house"),
		DisplayName: types.StringValue("In-House Apps"),
		ProfileData: types.StringValue("QkxPQg=="),
	}
	got := buildProvisioningProfileInput(plan)
	if got.General == nil {
		t.Fatal("expected non-nil General")
	}
	if got.General.Name == nil || *got.General.Name != "in-house" {
		t.Errorf("Name not carried, got %v", got.General.Name)
	}
	// display_name is server-derived (forced == name) and order-sensitive on the
	// wire — it must NOT be sent.
	if got.General.DisplayName != nil {
		t.Errorf("DisplayName must not be sent, got %q", *got.General.DisplayName)
	}
	if got.General.Profile == nil || got.General.Profile.Data == nil || *got.General.Profile.Data != "QkxPQg==" {
		t.Errorf("profile.data not carried, got %v", got.General.Profile)
	}
	if got.ID != nil {
		t.Errorf("expected nil top-level ID on write payload, got %d", *got.ID)
	}
}

func TestBuildInput_OmitsProfileWhenNoData(t *testing.T) {
	// Empty-shell create: no blob. The profile sub-object must be omitted, not
	// emitted with an empty <data/>.
	for _, tc := range []struct {
		name string
		data types.String
	}{
		{"null", types.StringNull()},
		{"unknown", types.StringUnknown()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan := ProvisioningProfileResourceModel{
				Name:        types.StringValue("shell"),
				ProfileData: tc.data,
			}
			got := buildProvisioningProfileInput(plan)
			if got.General.Profile != nil {
				t.Errorf("expected nil Profile for %s data, got %+v", tc.name, got.General.Profile)
			}
		})
	}
}
