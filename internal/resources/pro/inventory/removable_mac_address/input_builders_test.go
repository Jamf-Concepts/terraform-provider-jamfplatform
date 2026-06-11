// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package removable_mac_address

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildRemovableMacAddressInput_MacAddressRoundTrip(t *testing.T) {
	plan := RemovableMacAddressResourceModel{
		MacAddress: types.StringValue("00:A0:C9:14:C8:20"),
	}
	got := buildRemovableMacAddressInput(plan)
	if got.Name == nil {
		t.Fatalf("expected non-nil Name pointer")
	}
	if *got.Name != "00:A0:C9:14:C8:20" {
		t.Errorf("expected Name %q, got %q", "00:A0:C9:14:C8:20", *got.Name)
	}
	if got.ID != nil {
		t.Errorf("expected nil ID on write payload, got %d", *got.ID)
	}
}

func TestBuildRemovableMacAddressInput_NullBecomesNilPointer(t *testing.T) {
	// Schema validators prevent this in practice, but the builder must not invent an
	// empty string when the plan slot is null/unknown.
	for _, name := range []string{"null", "unknown"} {
		t.Run(name, func(t *testing.T) {
			plan := RemovableMacAddressResourceModel{MacAddress: types.StringNull()}
			if name == "unknown" {
				plan.MacAddress = types.StringUnknown()
			}
			got := buildRemovableMacAddressInput(plan)
			if got.Name != nil {
				t.Errorf("expected nil Name pointer for %s input, got %q", name, *got.Name)
			}
		})
	}
}
