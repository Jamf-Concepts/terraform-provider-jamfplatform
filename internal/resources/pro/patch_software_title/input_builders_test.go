// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_software_title

import (
	"context"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func mustMap(t *testing.T, m map[string]string) types.Map {
	t.Helper()
	v, diags := types.MapValueFrom(context.Background(), types.StringType, m)
	if diags.HasError() {
		t.Fatalf("build map: %v", diags)
	}
	return v
}

// TestBuildCreateInput_MintFieldsOnly pins the split between the two calls
// Create makes. The classic POST exists only to mint an id, and sending
// metadata or version assignments on it would write them through a deprecated
// endpoint whose category/site clear conventions are the inverse of v3's — so
// the mint payload must carry name + name_id + source_id and nothing else, even
// when the plan has category and package assignments configured.
func TestBuildCreateInput_MintFieldsOnly(t *testing.T) {
	plan := PatchSoftwareTitleResourceModel{
		Name:            types.StringValue("8x8 Work"),
		NameID:          types.StringValue("285"),
		SourceID:        types.Int64Value(1),
		CategoryID:      types.StringValue("5"),
		VersionPackages: mustMap(t, map[string]string{"8.33.2.2": "79"}),
	}
	got := buildPatchSoftwareTitleCreateInput(plan)

	if got.Name == nil || *got.Name != "8x8 Work" {
		t.Errorf("Name not set: %v", got.Name)
	}
	if got.NameID == nil || *got.NameID != "285" {
		t.Errorf("NameID not set: %v", got.NameID)
	}
	if got.SourceID == nil || *got.SourceID != 1 {
		t.Errorf("SourceID not set: %v", got.SourceID)
	}
	if got.Category != nil {
		t.Errorf("Category must not be sent on the mint POST")
	}
	if got.Site != nil {
		t.Errorf("Site must not be sent on the mint POST")
	}
	if got.Notifications != nil {
		t.Errorf("Notifications must not be sent on the mint POST")
	}
	if got.Versions != nil {
		t.Errorf("Versions must not be sent on the mint POST")
	}
	if got.ID != nil {
		t.Errorf("ID must be nil on a write payload")
	}
}

// TestBuildConfigurationPatch_MetadataAndNotifications pins that a configured
// value reaches the merge-patch, and specifically that a false notification is
// emitted rather than dropped. A merge-patch omitting the key leaves the
// server's value in place, so treating false as "nothing to say" would make
// disabling a notification impossible.
func TestBuildConfigurationPatch_MetadataAndNotifications(t *testing.T) {
	plan := PatchSoftwareTitleResourceModel{
		Name:              types.StringValue("8x8 Work"),
		CategoryID:        types.StringValue("58"),
		SiteID:            types.StringValue("-1"),
		WebNotification:   types.BoolValue(true),
		EmailNotification: types.BoolValue(false),
		VersionPackages:   types.MapNull(types.StringType),
	}
	got := buildPatchSoftwareTitleConfigurationPatch(plan, nil)

	if got.DisplayName == nil || *got.DisplayName != "8x8 Work" {
		t.Errorf("displayName not mapped: %v", got.DisplayName)
	}
	if got.CategoryID == nil || *got.CategoryID != "58" {
		t.Errorf("categoryId not mapped: %v", got.CategoryID)
	}
	if got.SiteID == nil || *got.SiteID != "-1" {
		t.Errorf("siteId not mapped: %v", got.SiteID)
	}
	if got.UiNotifications == nil || !*got.UiNotifications {
		t.Errorf("uiNotifications must be true, got %v", got.UiNotifications)
	}
	if got.EmailNotifications == nil {
		t.Fatalf("emailNotifications false must be emitted explicitly, not omitted")
	}
	if *got.EmailNotifications {
		t.Errorf("emailNotifications must be false, got true")
	}
}

// TestBuildConfigurationPatch_UnsetMetadataOmitsFields pins the other half of
// the merge-patch contract: an attribute the plan does not set must leave its
// key out of the body entirely, so the server's stored value survives rather
// than being overwritten with a zero value.
func TestBuildConfigurationPatch_UnsetMetadataOmitsFields(t *testing.T) {
	plan := PatchSoftwareTitleResourceModel{
		Name:              types.StringNull(),
		CategoryID:        types.StringNull(),
		SiteID:            types.StringUnknown(),
		WebNotification:   types.BoolNull(),
		EmailNotification: types.BoolUnknown(),
		VersionPackages:   types.MapNull(types.StringType),
	}
	got := buildPatchSoftwareTitleConfigurationPatch(plan, nil)

	if got.DisplayName != nil {
		t.Errorf("displayName must be omitted when unset, got %q", *got.DisplayName)
	}
	if got.CategoryID != nil {
		t.Errorf("categoryId must be omitted when null, got %q", *got.CategoryID)
	}
	if got.SiteID != nil {
		t.Errorf("siteId must be omitted when unknown, got %q", *got.SiteID)
	}
	if got.UiNotifications != nil {
		t.Errorf("uiNotifications must be omitted when null, got %v", *got.UiNotifications)
	}
	if got.EmailNotifications != nil {
		t.Errorf("emailNotifications must be omitted when unknown, got %v", *got.EmailNotifications)
	}
	if got.SoftwareTitleID != nil {
		t.Errorf("softwareTitleId must never be sent on the patch, got %q", *got.SoftwareTitleID)
	}
	if got.ExtensionAttributes != nil {
		t.Errorf("extensionAttributes must never be sent on the patch")
	}
}

// TestBuildConfigurationPatch_PackagesIsFullReplacement pins the wire law that
// makes the packages argument three-valued rather than a plain map. v3 treats a
// supplied `packages` array as the complete assignment set — anything absent
// from it is cleared — while omitting the key leaves the server's set alone. So
// nil must omit the key, an empty (non-nil) map must serialise as `[]` to clear
// every assignment, and a populated map must carry the whole array. Collapsing
// the empty map onto nil would make "unassign everything" unexpressible; the
// reverse would silently wipe out-of-band assignments on every apply.
func TestBuildConfigurationPatch_PackagesIsFullReplacement(t *testing.T) {
	plan := PatchSoftwareTitleResourceModel{
		Name:            types.StringValue("Title"),
		VersionPackages: types.MapNull(types.StringType),
	}

	tests := []struct {
		name     string
		packages map[string]string
		wantNil  bool
		want     map[string]string
	}{
		{name: "nil omits the key", packages: nil, wantNil: true},
		{name: "empty map clears every assignment", packages: map[string]string{}, want: map[string]string{}},
		{
			name:     "populated map replaces the whole set",
			packages: map[string]string{"8.33.2.2": "1", "8.32.2.10": "2"},
			want:     map[string]string{"8.33.2.2": "1", "8.32.2.10": "2"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildPatchSoftwareTitleConfigurationPatch(plan, tc.packages)
			if tc.wantNil {
				if got.Packages != nil {
					t.Fatalf("packages must be omitted for a nil map, got %+v", *got.Packages)
				}
				return
			}
			if got.Packages == nil {
				t.Fatalf("packages must be emitted for a non-nil map")
			}
			items := *got.Packages
			if len(items) != len(tc.want) {
				t.Fatalf("expected %d package items, got %d", len(tc.want), len(items))
			}
			for _, it := range items {
				if it.Version == nil || it.PackageID == nil {
					t.Fatalf("package item missing version or packageId: %+v", it)
				}
				if want, ok := tc.want[*it.Version]; !ok || want != *it.PackageID {
					t.Errorf("unexpected assignment %q → %q", *it.Version, *it.PackageID)
				}
			}
		})
	}
}

// TestRefIDPtr_NormalisesNonPositiveToMinusOne pins the v3 category/site id
// vocabulary: only a positive id or the literal "-1" is accepted, and anything
// else is refused outright. A title last written through the classic endpoint
// can still carry id "0" (that endpoint's own clear sentinel), so passing state
// straight through would fail the very apply meant to migrate it.
func TestRefIDPtr_NormalisesNonPositiveToMinusOne(t *testing.T) {
	tests := []struct {
		name    string
		in      types.String
		wantNil bool
		want    string
	}{
		{name: "null omits the key", in: types.StringNull(), wantNil: true},
		{name: "unknown omits the key", in: types.StringUnknown(), wantNil: true},
		{name: "empty omits the key", in: types.StringValue(""), wantNil: true},
		{name: "minus one passes through", in: types.StringValue("-1"), want: "-1"},
		{name: "classic zero normalises", in: types.StringValue("0"), want: "-1"},
		{name: "other negative normalises", in: types.StringValue("-7"), want: "-1"},
		{name: "positive passes through", in: types.StringValue("58"), want: "58"},
		{name: "non-numeric passes through", in: types.StringValue("abc"), want: "abc"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := refIDPtr(tc.in)
			if tc.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %q", *got)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected %q, got nil", tc.want)
			}
			if *got != tc.want {
				t.Errorf("expected %q, got %q", tc.want, *got)
			}
		})
	}
}

// TestUnionVersionPackages_PreservesOutOfBandAssignments pins the promise
// version_packages makes: only declared keys are managed. Because v3 `packages`
// is a full replacement, sending the plan alone would silently unassign every
// version a Jamf admin wired up outside Terraform, so the plan has to be folded
// over the live set instead.
func TestUnionVersionPackages_PreservesOutOfBandAssignments(t *testing.T) {
	live := map[string]string{"1.0": "5", "9.9": "7"}
	got := unionVersionPackages(live, map[string]string{"1.0": "6"}, nil)

	if len(got) != 2 {
		t.Fatalf("expected 2 assignments, got %d: %+v", len(got), got)
	}
	if got["9.9"] != "7" {
		t.Errorf("out-of-band assignment 9.9 must survive as 7, got %q", got["9.9"])
	}
	if got["1.0"] != "6" {
		t.Errorf("declared assignment 1.0 must take the plan value 6, got %q", got["1.0"])
	}
}

// TestUnionVersionPackages_PriorKeyDroppedFromPlanIsUnassigned pins the one way
// an assignment is allowed to disappear: a key that prior state managed and the
// plan no longer declares is a deliberate unassign, so it must leave the union.
// Without the prior-state diff there would be no difference between "the user
// removed this key" and "the user never managed it", and a removed key would
// stick forever.
func TestUnionVersionPackages_PriorKeyDroppedFromPlanIsUnassigned(t *testing.T) {
	live := map[string]string{"1.0": "5", "9.9": "7"}
	got := unionVersionPackages(live, map[string]string{"1.0": "6"}, []string{"9.9"})

	if _, ok := got["9.9"]; ok {
		t.Errorf("9.9 was managed and has been dropped from the plan, so it must be unassigned, got %q", got["9.9"])
	}
	if got["1.0"] != "6" {
		t.Errorf("declared assignment 1.0 must take the plan value 6, got %q", got["1.0"])
	}
}

// TestUnionVersionPackages_RetainedPriorKeyIsNotDropped guards the ordering
// inside the fold: the prior-key deletion pass runs before the plan is applied,
// so a key present in both prior state and the plan must survive with the
// plan's value rather than being deleted by its own prior-state entry.
func TestUnionVersionPackages_RetainedPriorKeyIsNotDropped(t *testing.T) {
	got := unionVersionPackages(map[string]string{"1.0": "5"}, map[string]string{"1.0": "6"}, []string{"1.0"})
	if got["1.0"] != "6" {
		t.Errorf("a retained key must keep the plan value 6, got %q", got["1.0"])
	}
}

// TestPackageItems_SortedByVersion pins the deterministic payload order. Go map
// iteration is randomised, so an unsorted array would make the merge-patch body
// differ between otherwise identical applies — noise in request logs and in any
// diff taken against a captured body.
func TestPackageItems_SortedByVersion(t *testing.T) {
	items := packageItems(map[string]string{"8.33.2.2": "1", "8.32.2.10": "2", "8.31.3.1": "3"})

	wantVersions := []string{"8.31.3.1", "8.32.2.10", "8.33.2.2"}
	wantPackages := []string{"3", "2", "1"}
	if len(items) != len(wantVersions) {
		t.Fatalf("expected %d items, got %d", len(wantVersions), len(items))
	}
	for i, it := range items {
		if it.Version == nil || it.PackageID == nil {
			t.Fatalf("item %d missing version or packageId: %+v", i, it)
		}
		if *it.Version != wantVersions[i] {
			t.Errorf("item %d: expected version %q, got %q", i, wantVersions[i], *it.Version)
		}
		if *it.PackageID != wantPackages[i] {
			t.Errorf("item %d (%q): expected package %q, got %q", i, *it.Version, wantPackages[i], *it.PackageID)
		}
	}
}

// TestPackageItems_EmptyMapYieldsNonNilEmptySlice pins the shape the clear path
// depends on: an empty map must produce a non-nil empty slice, because the
// caller takes its address and a nil slice would marshal as JSON null rather
// than the `[]` that clears every assignment.
func TestPackageItems_EmptyMapYieldsNonNilEmptySlice(t *testing.T) {
	items := packageItems(map[string]string{})
	if items == nil {
		t.Fatalf("expected a non-nil empty slice so the body carries [] rather than null")
	}
	if len(items) != 0 {
		t.Errorf("expected no items, got %d", len(items))
	}
}

// TestVersionPackageMap pins that an unset attribute decodes to an empty map
// rather than nil. Callers distinguish "nothing configured" by length, and the
// nil-versus-empty distinction is reserved for the packages argument of the
// merge-patch builder, where it means something else entirely.
func TestVersionPackageMap(t *testing.T) {
	tests := []struct {
		name string
		in   types.Map
		want map[string]string
	}{
		{name: "null", in: types.MapNull(types.StringType), want: map[string]string{}},
		{name: "unknown", in: types.MapUnknown(types.StringType), want: map[string]string{}},
		{name: "populated", in: mustMap(t, map[string]string{"1.0": "5"}), want: map[string]string{"1.0": "5"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, diags := versionPackageMap(context.Background(), tc.in)
			if diags.HasError() {
				t.Fatalf("diags: %v", diags)
			}
			if got == nil {
				t.Fatalf("expected a non-nil map")
			}
			if len(got) != len(tc.want) {
				t.Fatalf("expected %+v, got %+v", tc.want, got)
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Errorf("expected %q → %q, got %q", k, v, got[k])
				}
			}
		})
	}
}

// TestVersionPackageKeys pins the declared-key set used for Read reconciliation
// and Update unassign-diffing. An unset map must yield no keys at all rather
// than an empty non-nil slice, because "no keys declared" is what makes Read
// leave version_packages null instead of writing an empty map into state.
func TestVersionPackageKeys(t *testing.T) {
	keys, diags := versionPackageKeys(context.Background(), mustMap(t, map[string]string{"a": "1", "b": "2"}))
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	sort.Strings(keys)
	if len(keys) != 2 || keys[0] != "a" || keys[1] != "b" {
		t.Errorf("expected [a b], got %v", keys)
	}

	nullKeys, diags := versionPackageKeys(context.Background(), types.MapNull(types.StringType))
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if nullKeys != nil {
		t.Errorf("expected nil keys for a null map, got %v", nullKeys)
	}
}
