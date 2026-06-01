// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_software_title

import (
	"context"
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

func TestBuildCreateInput_MintFieldsOnly(t *testing.T) {
	plan := PatchSoftwareTitleResourceModel{
		Name:     types.StringValue("8x8 Work"),
		NameID:   types.StringValue("285"),
		SourceID: types.Int64Value(1),
		// Metadata + packages must NOT be sent on the mint POST.
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
		t.Errorf("Category must not be sent on mint POST")
	}
	if got.Versions != nil {
		t.Errorf("Versions must not be sent on mint POST")
	}
	if got.ID != nil {
		t.Errorf("ID must be nil on write payload")
	}
}

func TestBuildUpdateInput_CategorySiteNotifications(t *testing.T) {
	web, email := true, false
	plan := PatchSoftwareTitleResourceModel{
		Name:              types.StringValue("Title"),
		CategoryID:        types.StringValue("5"),
		SiteID:            types.StringValue("-1"),
		WebNotification:   types.BoolValue(web),
		EmailNotification: types.BoolValue(email),
		VersionPackages:   types.MapNull(types.StringType),
	}
	got, diags := buildPatchSoftwareTitleUpdateInput(context.Background(), plan, nil)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if got.Category == nil || got.Category.ID == nil || *got.Category.ID != 5 {
		t.Errorf("Category id not mapped: %+v", got.Category)
	}
	if got.Site == nil || got.Site.ID == nil || *got.Site.ID != -1 {
		t.Errorf("Site id -1 not mapped: %+v", got.Site)
	}
	if got.Notifications == nil {
		t.Fatalf("Notifications nil")
	}
	if got.Notifications.WebNotification == nil || *got.Notifications.WebNotification != true {
		t.Errorf("web notification not true: %v", got.Notifications.WebNotification)
	}
	// false must be sent explicitly, not dropped.
	if got.Notifications.EmailNotification == nil || *got.Notifications.EmailNotification != false {
		t.Errorf("email notification false must be explicit, got %v", got.Notifications.EmailNotification)
	}
}

// TestBuildUpdateInput_CategoryClearSentinel locks the wire-probed quirk: the
// endpoint clears <category> with id 0, not -1 (id -1 is a silent no-op for
// category, though it does clear <site>). The builder maps the user-facing
// "-1" sentinel — and any non-positive id — to wire id 0. Site is unaffected
// and sends its -1 verbatim.
func TestBuildUpdateInput_CategoryClearSentinel(t *testing.T) {
	for _, in := range []string{"-1", "0"} {
		plan := PatchSoftwareTitleResourceModel{
			Name:            types.StringValue("Title"),
			CategoryID:      types.StringValue(in),
			SiteID:          types.StringValue("-1"),
			VersionPackages: types.MapNull(types.StringType),
		}
		got, diags := buildPatchSoftwareTitleUpdateInput(context.Background(), plan, nil)
		if diags.HasError() {
			t.Fatalf("diags: %v", diags)
		}
		if got.Category == nil || got.Category.ID == nil || *got.Category.ID != 0 {
			t.Errorf("category_id %q must map to wire id 0, got %+v", in, got.Category)
		}
		// Site must still send -1 verbatim (its native clear sentinel).
		if got.Site == nil || got.Site.ID == nil || *got.Site.ID != -1 {
			t.Errorf("site_id -1 must send verbatim, got %+v", got.Site)
		}
	}
}

func TestBuildUpdateInput_NoMetadataMeansNilBlocks(t *testing.T) {
	plan := PatchSoftwareTitleResourceModel{
		CategoryID:        types.StringNull(),
		SiteID:            types.StringNull(),
		WebNotification:   types.BoolNull(),
		EmailNotification: types.BoolNull(),
		VersionPackages:   types.MapNull(types.StringType),
	}
	got, diags := buildPatchSoftwareTitleUpdateInput(context.Background(), plan, nil)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if got.Category != nil {
		t.Errorf("Category must be nil when unset")
	}
	if got.Site != nil {
		t.Errorf("Site must be nil when unset")
	}
	if got.Notifications != nil {
		t.Errorf("Notifications must be nil when both unset")
	}
	if got.Versions != nil {
		t.Errorf("Versions must be nil when no plan keys and no prior keys")
	}
}

func TestBuildUpdateInput_VersionPackages_AssignOnly(t *testing.T) {
	plan := PatchSoftwareTitleResourceModel{
		VersionPackages: mustMap(t, map[string]string{"8.33.2.2": "79", "8.32.2.10": "81"}),
	}
	got, diags := buildPatchSoftwareTitleUpdateInput(context.Background(), plan, nil)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if got.Versions == nil || got.Versions.Version == nil {
		t.Fatalf("expected versions block")
	}
	items := *got.Versions.Version
	if len(items) != 2 {
		t.Fatalf("expected 2 version items, got %d", len(items))
	}
	byVer := map[string]int{}
	for _, it := range items {
		if it.SoftwareVersion == nil {
			t.Fatalf("nil software version")
		}
		if it.Package == nil || it.Package.ID == nil {
			t.Errorf("assign entry %q must carry a package id", *it.SoftwareVersion)
			continue
		}
		byVer[*it.SoftwareVersion] = *it.Package.ID
	}
	if byVer["8.33.2.2"] != 79 || byVer["8.32.2.10"] != 81 {
		t.Errorf("assignments wrong: %+v", byVer)
	}
	// Deterministic sort by software_version.
	if *items[0].SoftwareVersion != "8.32.2.10" || *items[1].SoftwareVersion != "8.33.2.2" {
		t.Errorf("expected sorted versions, got %q then %q", *items[0].SoftwareVersion, *items[1].SoftwareVersion)
	}
}

// TestBuildUpdateInput_VersionPackages_ClearDiff covers the unassign path: a key
// present in prior state but dropped from the plan emits an empty (non-nil)
// package element which the server treats as "clear". A retained key emits its
// package id. A key never declared is not touched.
func TestBuildUpdateInput_VersionPackages_ClearDiff(t *testing.T) {
	plan := PatchSoftwareTitleResourceModel{
		VersionPackages: mustMap(t, map[string]string{"8.33.2.2": "79"}),
	}
	priorKeys := []string{"8.33.2.2", "8.32.2.10"} // 8.32.2.10 was removed
	got, diags := buildPatchSoftwareTitleUpdateInput(context.Background(), plan, priorKeys)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	items := *got.Versions.Version
	if len(items) != 2 {
		t.Fatalf("expected 2 entries (1 assign + 1 clear), got %d", len(items))
	}
	var sawAssign, sawClear bool
	for _, it := range items {
		switch *it.SoftwareVersion {
		case "8.33.2.2":
			if it.Package == nil || it.Package.ID == nil || *it.Package.ID != 79 {
				t.Errorf("retained version must keep package 79, got %+v", it.Package)
			}
			sawAssign = true
		case "8.32.2.10":
			// Clear: non-nil empty package element (ID nil).
			if it.Package == nil {
				t.Errorf("clear version must carry a non-nil empty package element")
			} else if it.Package.ID != nil {
				t.Errorf("clear version package must have nil ID, got %d", *it.Package.ID)
			}
			sawClear = true
		default:
			t.Errorf("unexpected version %q", *it.SoftwareVersion)
		}
	}
	if !sawAssign || !sawClear {
		t.Errorf("expected both assign and clear entries (assign=%v clear=%v)", sawAssign, sawClear)
	}
}

func TestBuildVersionItems_EmptyWhenNothingToDo(t *testing.T) {
	items := buildVersionItems(map[string]string{}, nil)
	if len(items) != 0 {
		t.Errorf("expected no items, got %d", len(items))
	}
}

func TestVersionPackageKeys(t *testing.T) {
	keys, diags := versionPackageKeys(context.Background(), mustMap(t, map[string]string{"a": "1", "b": "2"}))
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}

	nullKeys, diags := versionPackageKeys(context.Background(), types.MapNull(types.StringType))
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if len(nullKeys) != 0 {
		t.Errorf("expected 0 keys for null map, got %d", len(nullKeys))
	}
}

func TestOptionalIntFromStringID(t *testing.T) {
	cases := []struct {
		name    string
		in      types.String
		wantNil bool
		want    int
	}{
		{name: "null", in: types.StringNull(), wantNil: true},
		{name: "unknown", in: types.StringUnknown(), wantNil: true},
		{name: "empty", in: types.StringValue(""), wantNil: true},
		{name: "non-numeric", in: types.StringValue("abc"), wantNil: true},
		{name: "minus one", in: types.StringValue("-1"), want: -1},
		{name: "positive", in: types.StringValue("42"), want: 42},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := optionalIntFromStringID(tc.in)
			if tc.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %d", *got)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected %d, got nil", tc.want)
			}
			if *got != tc.want {
				t.Errorf("expected %d, got %d", tc.want, *got)
			}
		})
	}
}
