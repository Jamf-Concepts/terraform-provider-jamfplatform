// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package allowed_file_extension

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildAllowedFileExtensionInput_ExtensionRoundTrip(t *testing.T) {
	plan := AllowedFileExtensionResourceModel{
		Extension: types.StringValue("jpg"),
	}
	got := buildAllowedFileExtensionInput(plan)
	if got.Extension == nil {
		t.Fatalf("expected non-nil Extension pointer")
	}
	if *got.Extension != "jpg" {
		t.Errorf("expected Extension %q, got %q", "jpg", *got.Extension)
	}
	if got.ID != nil {
		t.Errorf("expected nil ID on write payload, got %d", *got.ID)
	}
}

func TestBuildAllowedFileExtensionInput_NullBecomesNilPointer(t *testing.T) {
	// Schema validators prevent this in practice, but the builder must not invent an
	// empty string when the plan slot is null/unknown.
	for _, name := range []string{"null", "unknown"} {
		t.Run(name, func(t *testing.T) {
			plan := AllowedFileExtensionResourceModel{Extension: types.StringNull()}
			if name == "unknown" {
				plan.Extension = types.StringUnknown()
			}
			got := buildAllowedFileExtensionInput(plan)
			if got.Extension != nil {
				t.Errorf("expected nil Extension pointer for %s input, got %q", name, *got.Extension)
			}
		})
	}
}
