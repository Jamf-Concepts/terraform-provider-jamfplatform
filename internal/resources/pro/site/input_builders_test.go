// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package site

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildSiteInput_NameRoundTrip(t *testing.T) {
	plan := SiteResourceModel{
		Name: types.StringValue("Primary"),
	}
	got := buildSiteInput(plan)
	if got.Name == nil {
		t.Fatalf("expected non-nil Name pointer")
	}
	if *got.Name != "Primary" {
		t.Errorf("expected Name %q, got %q", "Primary", *got.Name)
	}
	if got.ID != nil {
		t.Errorf("expected nil ID on write payload, got %d", *got.ID)
	}
}

func TestBuildSiteInput_NullNameBecomesNilPointer(t *testing.T) {
	// Schema validators prevent this in practice, but the builder must not
	// invent an empty string when the plan slot is null/unknown.
	for _, name := range []string{"null", "unknown"} {
		t.Run(name, func(t *testing.T) {
			plan := SiteResourceModel{Name: types.StringNull()}
			if name == "unknown" {
				plan.Name = types.StringUnknown()
			}
			got := buildSiteInput(plan)
			if got.Name != nil {
				t.Errorf("expected nil Name pointer for %s input, got %q", name, *got.Name)
			}
		})
	}
}
