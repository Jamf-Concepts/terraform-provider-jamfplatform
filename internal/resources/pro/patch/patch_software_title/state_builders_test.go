// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_software_title

import (
	"context"
	"encoding/xml"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// liveWire is a trimmed real /patchsoftwaretitles GET (8x8 Work, id 6) with one
// version assigned (8.33.2.2 → package 79), default category/site (id -1) and
// both notifications true. Pins the decode + mapping contract.
const liveWire = `<?xml version="1.0" encoding="UTF-8"?><patch_software_title>
  <id>6</id>
  <name>8x8 Work</name>
  <name_id>285</name_id>
  <source_id>1</source_id>
  <notifications>
    <web_notification>true</web_notification>
    <email_notification>true</email_notification>
  </notifications>
  <category><id>-1</id><name>No category assigned</name></category>
  <site><id>-1</id><name>NONE</name></site>
  <versions>
    <version><software_version>8.33.2.2</software_version><package><id>79</id><name>acs-pkg-sophos.pkg</name></package></version>
    <version><software_version>8.32.2.10</software_version><package></package></version>
    <version><software_version>8.31.3.1</software_version><package></package></version>
  </versions>
</patch_software_title>`

func decodeLiveWire(t *testing.T) *proclassic.PatchSoftwareTitle {
	t.Helper()
	var p proclassic.PatchSoftwareTitle
	if err := xml.Unmarshal([]byte(liveWire), &p); err != nil {
		t.Fatalf("unmarshal live wire: %v", err)
	}
	return &p
}

func TestLiveWireUnmarshal(t *testing.T) {
	p := decodeLiveWire(t)
	if p.ID == nil || *p.ID != 6 {
		t.Errorf("id: %v", p.ID)
	}
	if p.NameID == nil || *p.NameID != "285" {
		t.Errorf("name_id: %v", p.NameID)
	}
	if p.SourceID == nil || *p.SourceID != 1 {
		t.Errorf("source_id: %v", p.SourceID)
	}
	if p.Versions == nil || p.Versions.Version == nil || len(*p.Versions.Version) != 3 {
		t.Fatalf("expected 3 versions")
	}
	// First version has a package; the empty <package></package> decodes to a
	// non-nil package with nil ID (treated as unassigned).
	first := (*p.Versions.Version)[0]
	if first.Package == nil || first.Package.ID == nil || *first.Package.ID != 79 {
		t.Errorf("first version package id: %+v", first.Package)
	}
	second := (*p.Versions.Version)[1]
	if second.Package != nil && second.Package.ID != nil {
		t.Errorf("empty <package></package> must decode to nil ID, got %d", *second.Package.ID)
	}
}

func TestAssignResourceModel_ManagedSubsetReconcile(t *testing.T) {
	p := decodeLiveWire(t)
	// User declared only 8.33.2.2 (which is assigned) and 8.30.0.0 (which the
	// server does not have / has no package). Read must include only 8.33.2.2.
	declared := []string{"8.33.2.2", "8.30.0.0"}
	state := PatchSoftwareTitleResourceModel{}
	diags := assignPatchSoftwareTitleResourceModel(context.Background(), &state, p, declared)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}

	if state.ID.ValueString() != "6" {
		t.Errorf("id: %q", state.ID.ValueString())
	}
	if state.Name.ValueString() != "8x8 Work" {
		t.Errorf("name: %q", state.Name.ValueString())
	}
	if state.SourceID.ValueInt64() != 1 {
		t.Errorf("source_id: %d", state.SourceID.ValueInt64())
	}
	if state.CategoryID.ValueString() != "-1" {
		t.Errorf("category_id: %q", state.CategoryID.ValueString())
	}
	if state.CategoryName.ValueString() != "No category assigned" {
		t.Errorf("category_name: %q", state.CategoryName.ValueString())
	}
	if state.SiteID.ValueString() != "-1" || state.SiteName.ValueString() != "NONE" {
		t.Errorf("site: id=%q name=%q", state.SiteID.ValueString(), state.SiteName.ValueString())
	}
	if !state.WebNotification.ValueBool() || !state.EmailNotification.ValueBool() {
		t.Errorf("notifications: web=%v email=%v", state.WebNotification.ValueBool(), state.EmailNotification.ValueBool())
	}

	// version_packages must be exactly {8.33.2.2: 79} — declared-but-unassigned
	// 8.30.0.0 is dropped (surfaces drift).
	if state.VersionPackages.IsNull() {
		t.Fatalf("version_packages must be non-null when a declared key is assigned")
	}
	vp := map[string]string{}
	state.VersionPackages.ElementsAs(context.Background(), &vp, false)
	if len(vp) != 1 || vp["8.33.2.2"] != "79" {
		t.Errorf("expected {8.33.2.2:79}, got %+v", vp)
	}

	// available_versions surfaces every server version in order.
	av := []string{}
	state.AvailableVersions.ElementsAs(context.Background(), &av, false)
	if len(av) != 3 || av[0] != "8.33.2.2" || av[2] != "8.31.3.1" {
		t.Errorf("available_versions wrong: %+v", av)
	}
}

func TestAssignResourceModel_NoDeclaredKeysYieldsNullMap(t *testing.T) {
	p := decodeLiveWire(t)
	state := PatchSoftwareTitleResourceModel{}
	diags := assignPatchSoftwareTitleResourceModel(context.Background(), &state, p, nil)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if !state.VersionPackages.IsNull() {
		t.Errorf("expected null version_packages when no keys declared, got %v", state.VersionPackages)
	}
}

func TestAssignResourceModel_DeclaredKeyDroppedWhenPackageGone(t *testing.T) {
	p := decodeLiveWire(t)
	// Declared only a version whose package is empty server-side → map drops to null.
	state := PatchSoftwareTitleResourceModel{}
	diags := assignPatchSoftwareTitleResourceModel(context.Background(), &state, p, []string{"8.32.2.10"})
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if !state.VersionPackages.IsNull() {
		t.Errorf("declared key with no server package must drop → null map, got %v", state.VersionPackages)
	}
}

func TestAssignResourceModel_PreservesIDWhenAPINil(t *testing.T) {
	name := "refreshed"
	state := PatchSoftwareTitleResourceModel{ID: types.StringValue("42")}
	api := &proclassic.PatchSoftwareTitle{ID: nil, Name: &name}
	diags := assignPatchSoftwareTitleResourceModel(context.Background(), &state, api, nil)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if state.ID.ValueString() != "42" {
		t.Errorf("expected ID preserved as 42, got %q", state.ID.ValueString())
	}
}

func TestAssignResourceModel_NilAPIIsNoop(t *testing.T) {
	state := PatchSoftwareTitleResourceModel{ID: types.StringValue("7"), Name: types.StringValue("Keep")}
	diags := assignPatchSoftwareTitleResourceModel(context.Background(), &state, nil, nil)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if state.ID.ValueString() != "7" || state.Name.ValueString() != "Keep" {
		t.Errorf("expected state unchanged, got id=%q name=%q", state.ID.ValueString(), state.Name.ValueString())
	}
}

func TestAssignDataSourceModel_FullServerView(t *testing.T) {
	p := decodeLiveWire(t)
	state := PatchSoftwareTitleDataSourceModel{}
	diags := assignPatchSoftwareTitleDataSourceModel(context.Background(), &state, p)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	// Data source surfaces every assigned version, no managed-subset gating.
	vp := map[string]string{}
	state.VersionPackages.ElementsAs(context.Background(), &vp, false)
	if len(vp) != 1 || vp["8.33.2.2"] != "79" {
		t.Errorf("expected {8.33.2.2:79}, got %+v", vp)
	}
	if state.NameID.ValueString() != "285" {
		t.Errorf("name_id: %q", state.NameID.ValueString())
	}
}

func TestCategorySiteNotificationValues_NilBlocks(t *testing.T) {
	cid, cname := categoryValues(nil)
	if !cid.IsNull() || !cname.IsNull() {
		t.Errorf("nil category must yield null/null")
	}
	sid, sname := siteValues(nil)
	if !sid.IsNull() || !sname.IsNull() {
		t.Errorf("nil site must yield null/null")
	}
	web, email := notificationValues(nil)
	if !web.IsNull() || !email.IsNull() {
		t.Errorf("nil notifications must yield null/null")
	}
}

// TestCategoryValues_NoCategorySentinel locks the read side of the wire-probed
// quirk: the endpoint reports "no category" as id -1 (never assigned) or id 0
// (explicitly cleared); both must collapse to the "-1" user-facing sentinel so
// state round-trips against a config of "-1".
func TestCategoryValues_NoCategorySentinel(t *testing.T) {
	for _, id := range []int{-1, 0} {
		v := id
		cid, _ := categoryValues(&proclassic.CategoryObject{ID: &v})
		if cid.ValueString() != "-1" {
			t.Errorf("server category id %d must normalise to \"-1\", got %q", id, cid.ValueString())
		}
	}
	real := 58
	cid, _ := categoryValues(&proclassic.CategoryObject{ID: &real})
	if cid.ValueString() != "58" {
		t.Errorf("real category id must pass through, got %q", cid.ValueString())
	}
}
