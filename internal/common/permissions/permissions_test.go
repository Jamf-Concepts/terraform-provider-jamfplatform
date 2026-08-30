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
			// The older {action}:{scope}:{resource} spelling puts the action
			// first and the capability last, so both hasField and capabilityOf
			// have to read it positionally rather than assuming the GA form.
			name:   "old three-field spelling, aligned",
			scoped: []string{"create:pro:packages", "read:pro:packages"},
			legacy: []string{"Create Packages", "Read Packages"},
			want:   true,
		},
		{
			name:   "old three-field spelling, swapped",
			scoped: []string{"create:pro:packages", "read:pro:packages"},
			legacy: []string{"Read Packages", "Create Packages"},
			want:   false,
		},
		{
			name:   "same action, capability matches its description",
			scoped: []string{"gsx-connection:read", "push-certificates:read"},
			legacy: []string{"Read GSX Connection", "Read Push Certificates"},
			want:   true,
		},
		{
			// GetDeviceGroupsForDeviceV1 as published. Every action is "read", so
			// the verb test passes whatever the order; only the capability check
			// catches that `device-groups:read` is not "Read Computers".
			name:   "same action, capability contradicts its description",
			scoped: []string{"device-groups:read", "devices:read"},
			legacy: []string{"Read Computers", "Read Mobile Devices"},
			want:   false,
		},
		{
			// Jamf spells the same noun two ways either side of the boundary;
			// de-pluralising and splitting on punctuation has to absorb both.
			name:   "singular/plural and hyphen spellings still match",
			scoped: []string{"categories:read", "computer-check-in:read"},
			legacy: []string{"Read Categories", "Read Computer Check-In"},
			want:   true,
		},
		{
			// UploadInventoryPreloadCsvV2 as published: a plural capability
			// against a singular label. Only the de-pluralisation in splitWords
			// makes these rows confirmable, and a same-capability sibling must
			// not count as a rival for the discriminating check.
			name:   "plural capability against singular label",
			scoped: []string{"inventory-preload-records:create", "users:create"},
			legacy: []string{"Create Inventory Preload Records", "Create User"},
			want:   true,
		},
		{
			// ListSmartMobileDeviceGroupMembershipV1 as published. Both labels
			// name a device, so a shared word confirms neither row: the pair
			// must be refused in BOTH orders, not waved through in both.
			name:   "shared noun confirms nothing, published order",
			scoped: []string{"device-groups:read", "devices:read"},
			legacy: []string{"Read Smart Mobile Device Groups", "Read Mobile Devices"},
			want:   false,
		},
		{
			name:   "shared noun confirms nothing, swapped order",
			scoped: []string{"device-groups:read", "devices:read"},
			legacy: []string{"Read Mobile Devices", "Read Smart Mobile Device Groups"},
			want:   false,
		},
		{
			// The function is total on its own rather than relying on the
			// caller's length gate: Section is called from package-level var
			// initialisers, so an index panic here aborts provider startup.
			name:   "more legacy names than scoped identifiers",
			scoped: []string{"packages:read", "packages:update"},
			legacy: []string{"Read Packages", "Update Packages", "Delete Packages"},
			want:   false,
		},
		{
			name:   "no legacy names at all",
			scoped: []string{"devices:create", "users:create"},
			legacy: nil,
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
		// RedeployJamfManagementFrameworkV1 as published. Two capabilities, and
		// one label leads with a verb outside the four CRUD ones, so neither
		// index pairing nor a verb key can place the rows. Index pairing here
		// is what used to label `computer-check-in:read` "Send Computer Remote
		// Command to Install Package".
		"Unrecoverable": jamfplatform.MethodPrivileges{
			Scoped: []string{"computer-check-in:read", "device-actions:execute"},
			Legacy: []string{"Send Computer Remote Command to Install Package", "Read Computer Check-In"},
		},
	}

	got := Section(reg, "Unrecoverable")
	for _, scoped := range []string{"computer-check-in:read", "device-actions:execute"} {
		if !strings.Contains(got, "`"+scoped+"`") {
			t.Errorf("section dropped the scoped name %s:\n%s", scoped, got)
		}
	}
	if strings.Contains(got, "Send Computer Remote Command") || strings.Contains(got, "Read Computer Check-In") {
		t.Errorf("section rendered an unverified admin-UI label:\n%s", got)
	}
	if !strings.Contains(got, "| Required privilege |") {
		t.Errorf("section should fall back to the scoped-only table:\n%s", got)
	}
}

// TestSection_SwappedSingleCapabilityIsRepaired pins the verb-keyed stage: where
// every privilege on a method shares one capability, the label's leading verb
// names its action outright, so a set the SDK emitted in a different order is
// reconstructed rather than discarded. UpdateManagedSoftwareUpdateFeatureToggleV1
// is this shape, and dropping its labels is what made the docs unusable for an
// operator granting privileges by name in the Jamf Pro admin UI.
func TestSection_SwappedSingleCapabilityIsRepaired(t *testing.T) {
	reg := Registry{
		"Swapped": jamfplatform.MethodPrivileges{
			Scoped: []string{"managed-software-updates:create", "managed-software-updates:read", "managed-software-updates:update"},
			Legacy: []string{"Read Managed Software Updates", "Create Managed Software Updates", "Update Managed Software Updates"},
		},
	}

	got := Section(reg, "Swapped")
	for _, want := range []string{
		"| Create Managed Software Updates | `managed-software-updates:create` |",
		"| Read Managed Software Updates | `managed-software-updates:read` |",
		"| Update Managed Software Updates | `managed-software-updates:update` |",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("verb-keyed pairing did not render %q:\n%s", want, got)
		}
	}
}

// TestSection_CollapsedLegacyNamesAreAllListed pins the single-scoped stage. Jamf's
// GA privilege collapse maps several pre-GA privileges onto one identifier, so the
// lists differ in length without the pairing being ambiguous — every published name
// belongs to the one identifier, and an operator on a pre-GA tenant needs all of
// them. ListMacOSBrandingConfigurationsV1 is this shape.
func TestSection_CollapsedLegacyNamesAreAllListed(t *testing.T) {
	reg := Registry{
		"Collapsed": jamfplatform.MethodPrivileges{
			Scoped: []string{"self-service:read"},
			Legacy: []string{"Read Self Service Branding Configuration", "Read Self Service"},
		},
	}

	got := Section(reg, "Collapsed")
	if !strings.Contains(got, "Read Self Service Branding Configuration") || !strings.Contains(got, "Read Self Service |") {
		t.Errorf("section dropped a collapsed legacy name:\n%s", got)
	}
	if !strings.Contains(got, "grant every name listed") {
		t.Errorf("a multi-name row needs the note explaining that all of them are required:\n%s", got)
	}
}

// TestSection_UnpairableMismatchStaysUnlabelled guards the other side of that
// stage: more than one scoped identifier against a different number of legacy
// names really is unpairable, and must not be labelled.
func TestSection_UnpairableMismatchStaysUnlabelled(t *testing.T) {
	reg := Registry{
		"Mismatched": jamfplatform.MethodPrivileges{
			Scoped: []string{"devices:create", "users:create"},
			Legacy: []string{"Create Computers"},
		},
	}

	got := Section(reg, "Mismatched")
	if strings.Contains(got, "Create Computers") {
		t.Errorf("section labelled an unpairable mismatch:\n%s", got)
	}
}

// TestSection_WellPairedSiblingStillLabelsTheRow guards the dedup interaction: a
// method whose pairing cannot be verified must not strip a label another method
// legitimately supplied for the same scoped name. The unlabelled method is named
// FIRST deliberately — that is the order which makes collect reach the branch
// that back-fills the label, and the order jamfplatform_pro_self_service_branding_macos
// actually declares (its List method carries the collapsed pair, its Get method
// the confirmable single).
func TestSection_WellPairedSiblingStillLabelsTheRow(t *testing.T) {
	reg := Registry{
		// Genuinely unpairable, so it contributes computer-check-in:read with no
		// label at all: two capabilities and a non-CRUD leading verb.
		"Unrecoverable": jamfplatform.MethodPrivileges{
			Scoped: []string{"computer-check-in:read", "device-actions:execute"},
			Legacy: []string{"Send Computer Remote Command to Install Package", "Read Computer Check-In"},
		},
		"GoodRead": jamfplatform.MethodPrivileges{
			Scoped: []string{"computer-check-in:read"},
			Legacy: []string{"Read Computer Check-In"},
		},
	}

	got := Section(reg, "Unrecoverable", "GoodRead")
	if !strings.Contains(got, "| Read Computer Check-In | `computer-check-in:read` |") {
		t.Errorf("the well-paired label was not back-filled onto the blank row:\n%s", got)
	}
	if strings.Contains(got, "| Send Computer Remote Command to Install Package |") {
		t.Errorf("an unverified pairing mislabelled a row:\n%s", got)
	}
	if !strings.Contains(got, "`—` means Jamf publishes no Jamf Pro privilege name") {
		t.Errorf("a table with a blank label needs the note explaining the em dash:\n%s", got)
	}
}

func TestSharesWord(t *testing.T) {
	cases := []struct {
		capability, description string
		want                    bool
	}{
		{"device-groups", "Smart Mobile Device Groups", true},
		{"device-groups", "Static Computer Groups", true},
		{"device-groups", "Computers", false},
		{"devices", "Mobile Devices", true},
		{"categories", "Categories", true},
		{"computer-check-in", "Computer Check-In", true},
		{"gsx-connection", "GSX Connection", true},
		{"push-certificates", "GSX Connection", false},
		{"self-service", "Self Service", true},
		{"packages", "Packages", true},
		{"", "Packages", false},
		{"packages", "", false},
	}
	for _, c := range cases {
		if got := sharesWord(c.capability, c.description); got != c.want {
			t.Errorf("sharesWord(%q, %q) = %v, want %v", c.capability, c.description, got, c.want)
		}
	}
}

// TestVerifiedPairing_SingleIsAlwaysTrusted pins the deliberate asymmetry: the
// capability check is NOT applied to a lone privilege, because there is no
// ordering to get wrong and many legitimate one-privilege methods carry a label
// whose wording does not overlap its capability slug. Tightening this would
// strip labels from ordinary constructs for no correctness gain.
func TestVerifiedPairing_SingleIsAlwaysTrusted(t *testing.T) {
	if !verifiedPairing([]string{"jamf-protect-deployments:read"}, []string{"Read Jamf Protect Settings"}) {
		t.Error("a single privilege must stay trusted even when its label does not match its capability")
	}
}

// TestVerbKeyedPairing covers the reconstruction stage directly, including every
// way it refuses. It only ever fires on a set verifiedPairing already rejected,
// so refusing is the common case and each refusal has to be deliberate rather
// than a fall-through.
func TestVerbKeyedPairing(t *testing.T) {
	cases := []struct {
		name           string
		scoped, legacy []string
		want           []string
	}{
		{
			name:   "swapped single-capability CRUD set is re-paired",
			scoped: []string{"packages:read", "packages:update"},
			legacy: []string{"Update Packages", "Read Packages"},
			want:   []string{"Read Packages", "Update Packages"},
		},
		{
			name:   "old three-field spelling re-pairs too",
			scoped: []string{"create:pro:packages", "read:pro:packages"},
			legacy: []string{"Read Packages", "Create Packages"},
			want:   []string{"Create Packages", "Read Packages"},
		},
		{
			// Two capabilities: the verb no longer identifies a row, because
			// two different capabilities can share an action.
			name:   "more than one capability is refused",
			scoped: []string{"device-groups:read", "devices:read"},
			legacy: []string{"Read Mobile Devices", "Read Smart Mobile Device Groups"},
			want:   nil,
		},
		{
			name:   "non-CRUD verb is refused",
			scoped: []string{"packages:read", "packages:update"},
			legacy: []string{"Send Packages", "Read Packages"},
			want:   nil,
		},
		{
			name:   "single-word legacy name is refused",
			scoped: []string{"packages:read", "packages:update"},
			legacy: []string{"Read", "Update"},
			want:   nil,
		},
		{
			// "Delete Packages" names an action no scoped identifier carries,
			// so the assignment cannot be total.
			name:   "verb naming an absent action is refused",
			scoped: []string{"packages:read", "packages:update"},
			legacy: []string{"Read Packages", "Delete Packages"},
			want:   nil,
		},
		{
			// Two labels claiming the same row: half-pairing it would leave the
			// other identifier silently unlabelled inside a "trusted" set.
			name:   "two labels claiming one action is refused",
			scoped: []string{"packages:read", "packages:update"},
			legacy: []string{"Read Packages", "Read Packages"},
			want:   nil,
		},
		{
			name:   "fewer labels than identifiers is refused",
			scoped: []string{"packages:read", "packages:update"},
			legacy: []string{"Read Packages"},
			want:   nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := verbKeyedPairing(c.scoped, c.legacy)
			if len(got) != len(c.want) {
				t.Fatalf("verbKeyedPairing(%v, %v) = %v, want %v", c.scoped, c.legacy, got, c.want)
			}
			for i := range c.want {
				if got[i] != c.want[i] {
					t.Errorf("row %d = %q, want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}
