// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package permissions

import (
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
)

func TestSection_BuildingCRUD(t *testing.T) {
	got := Section(pro.Privileges,
		"CreateBuildingV1", "GetBuildingV1", "UpdateBuildingV1", "DeleteBuildingV1")

	for _, want := range []string{
		"**Required Jamf privileges**",
		"| Jamf Pro privilege | Scoped name |",
		"| Create Buildings | `buildings:create` |",
		"| Read Buildings | `buildings:read` |",
		"| Update Buildings | `buildings:update` |",
		"| Delete Buildings | `buildings:delete` |",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Section() missing %q\n--- got ---\n%s", want, got)
		}
	}
}

func TestSection_DeduplicatesAcrossMethods(t *testing.T) {
	// Read + Get for the same resource both require buildings:read; the
	// table must list it once.
	got := Section(pro.Privileges, "GetBuildingV1", "ListBuildingsV1")
	if n := strings.Count(got, "`buildings:read`"); n != 1 {
		t.Fatalf("buildings:read should appear once, got %d\n%s", n, got)
	}
}

func TestSection_NoPrivilegeEndpoint(t *testing.T) {
	// Find a registry entry that requires no special privilege and assert the
	// "None" wording renders.
	var none string
	for name, mp := range pro.Privileges {
		if len(mp.Scoped) == 0 {
			none = name
			break
		}
	}
	if none == "" {
		t.Skip("no zero-privilege method in registry")
	}
	got := Section(pro.Privileges, none)
	if !strings.Contains(got, "None — any authenticated Jamf Platform API integration") {
		t.Errorf("expected no-privilege wording for %s, got:\n%s", none, got)
	}
}

func TestSection_EmptyWhenAllMissing(t *testing.T) {
	if got := Section(pro.Privileges, "NoSuchMethodXYZ"); got != "" {
		t.Errorf("expected empty section for unknown method, got:\n%s", got)
	}
}

func TestMissing_DetectsUnknown(t *testing.T) {
	missing := Missing(pro.Privileges, "GetBuildingV1", "NoSuchMethodXYZ")
	if len(missing) != 1 || missing[0] != "NoSuchMethodXYZ" {
		t.Fatalf("want [NoSuchMethodXYZ], got %v", missing)
	}
}

func TestMerge(t *testing.T) {
	a := Registry{"X": jamfplatform.MethodPrivileges{Method: "X"}}
	b := Registry{"Y": jamfplatform.MethodPrivileges{Method: "Y"}}
	merged := Merge(a, b)
	if _, ok := merged["X"]; !ok {
		t.Error("merged registry missing X")
	}
	if _, ok := merged["Y"]; !ok {
		t.Error("merged registry missing Y")
	}
}

// --- pairing verification -----------------------------------------------------

func TestVerifiedPairing(t *testing.T) {
	cases := []struct {
		name   string
		scoped []string
		legacy []string
		want   bool
	}{
		{
			name:   "single privilege is always trusted",
			scoped: []string{"packages:read"},
			legacy: []string{"Anything At All"},
			want:   true,
		},
		{
			name:   "GA form, aligned CRUD pair",
			scoped: []string{"packages:read", "packages:update"},
			legacy: []string{"Read Packages", "Update Packages"},
			want:   true,
		},
		{
			name:   "GA form, swapped CRUD pair",
			scoped: []string{"packages:read", "packages:update"},
			legacy: []string{"Update Packages", "Read Packages"},
			want:   false,
		},
		{
			name:   "legacy privilege spelling, aligned",
			scoped: []string{"packages:create", "packages:read"},
			legacy: []string{"Create Packages", "Read Packages"},
			want:   true,
		},
		{
			name:   "legacy privilege spelling, swapped",
			scoped: []string{"packages:create", "packages:read"},
			legacy: []string{"Read Packages", "Create Packages"},
			want:   false,
		},
		{
			name:   "non-CRUD verb cannot be checked",
			scoped: []string{"computer-check-in:read", "device-actions:execute"},
			legacy: []string{"Send Computer Remote Command to Install Package", "Read Computer Check-In"},
			want:   false,
		},
		{
			name:   "single-word legacy name cannot be checked",
			scoped: []string{"packages:read", "packages:update"},
			legacy: []string{"Read", "Update"},
			want:   false,
		},
	}

	for _, c := range cases {
		if got := verifiedPairing(c.scoped, c.legacy); got != c.want {
			t.Errorf("%s: verifiedPairing = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestSection_UnverifiedPairingDropsLegacyNames is the user-facing consequence:
// an unverifiable pairing must render the scoped names WITHOUT admin-UI labels,
// never with mislabelled ones. Asserted through Section rather than the helper so
// the wiring into collect is covered too.
func TestSection_UnverifiedPairingDropsLegacyNames(t *testing.T) {
	reg := Registry{
		"Swapped": jamfplatform.MethodPrivileges{
			Scoped: []string{"managed-software-updates:create", "managed-software-updates:read"},
			Legacy: []string{"Read Managed Software Updates", "Create Managed Software Updates"},
		},
	}

	got := Section(reg, "Swapped")
	for _, scoped := range []string{"managed-software-updates:create", "managed-software-updates:read"} {
		if !strings.Contains(got, "`"+scoped+"`") {
			t.Errorf("section dropped the scoped name %s:\n%s", scoped, got)
		}
	}
	if strings.Contains(got, "Managed Software Updates |") {
		t.Errorf("section rendered an unverified admin-UI label:\n%s", got)
	}
	if !strings.Contains(got, "| Required privilege |") {
		t.Errorf("section should fall back to the scoped-only table:\n%s", got)
	}
}

// TestSection_WellPairedSiblingStillLabelsTheRow guards the dedup interaction: a
// method whose pairing cannot be verified must not strip a label another method
// legitimately supplied for the same scoped name. This is why jamfplatform_pro_package
// keeps its labels despite three of its methods being unverifiable.
func TestSection_WellPairedSiblingStillLabelsTheRow(t *testing.T) {
	reg := Registry{
		"GoodRead": jamfplatform.MethodPrivileges{
			Scoped: []string{"packages:read"},
			Legacy: []string{"Read Packages"},
		},
		"Unverifiable": jamfplatform.MethodPrivileges{
			Scoped: []string{"packages:read", "packages:update"},
			Legacy: []string{"Update Packages", "Read Packages"},
		},
	}

	got := Section(reg, "GoodRead", "Unverifiable")
	if !strings.Contains(got, "| Read Packages | `packages:read` |") {
		t.Errorf("the well-paired label was lost:\n%s", got)
	}
	if strings.Contains(got, "| Update Packages | `packages:read` |") {
		t.Errorf("an unverified pairing mislabelled a row:\n%s", got)
	}
}
