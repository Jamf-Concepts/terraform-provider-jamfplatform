// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_software_title

import (
	"context"
	"strconv"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// assignPatchSoftwareTitleResourceModel populates a resource model from a
// PatchSoftwareTitle response. declaredKeys is the managed subset of
// software_version keys the caller declared (plan keys on Create/Update, prior
// state keys on Read): version_packages is rebuilt from only those keys by
// looking each up in the server's versions and reading its assigned package id.
// A declared key whose package is gone server-side is dropped from the map,
// surfacing the drift. state.ID is only overwritten when the API ID is non-nil.
func assignPatchSoftwareTitleResourceModel(ctx context.Context, state *PatchSoftwareTitleResourceModel, s *proclassic.PatchSoftwareTitle, declaredKeys []string) diag.Diagnostics {
	var diags diag.Diagnostics
	if s == nil {
		return diags
	}
	if s.ID != nil {
		state.ID = helpers.StringValueFromIntPtr(s.ID)
	}
	if s.Name != nil {
		state.Name = helpers.StringPointerValueOrNull(s.Name)
	}
	if s.NameID != nil {
		state.NameID = helpers.StringPointerValueOrNull(s.NameID)
	}
	if s.SourceID != nil {
		state.SourceID = types.Int64Value(int64(*s.SourceID))
	}

	state.CategoryID, state.CategoryName = categoryValues(s.Category)
	state.SiteID, state.SiteName = siteValues(s.Site)

	web, email := notificationValues(s.Notifications)
	state.WebNotification = web
	state.EmailNotification = email

	assigned := assignedPackagesByVersion(s.Versions)

	avail := availableVersions(s.Versions)
	availList, d := types.ListValueFrom(ctx, types.StringType, avail)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	state.AvailableVersions = availList

	vp, d := managedVersionPackages(ctx, declaredKeys, assigned)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	state.VersionPackages = vp

	return diags
}

// assignPatchSoftwareTitleDataSourceModel populates a data source model. The
// data source surfaces the full server view, so version_packages is built from
// every server version that has a package assigned (no managed-subset gating —
// there is no prior state to scope against).
func assignPatchSoftwareTitleDataSourceModel(ctx context.Context, state *PatchSoftwareTitleDataSourceModel, s *proclassic.PatchSoftwareTitle) diag.Diagnostics {
	var diags diag.Diagnostics
	if s == nil {
		return diags
	}
	if s.ID != nil {
		state.ID = helpers.StringValueFromIntPtr(s.ID)
	}
	state.Name = helpers.StringPointerValueOrNull(s.Name)
	state.NameID = helpers.StringPointerValueOrNull(s.NameID)
	if s.SourceID != nil {
		state.SourceID = types.Int64Value(int64(*s.SourceID))
	} else {
		state.SourceID = types.Int64Null()
	}

	state.CategoryID, state.CategoryName = categoryValues(s.Category)
	state.SiteID, state.SiteName = siteValues(s.Site)

	web, email := notificationValues(s.Notifications)
	state.WebNotification = web
	state.EmailNotification = email

	assigned := assignedPackagesByVersion(s.Versions)

	avail := availableVersions(s.Versions)
	availList, d := types.ListValueFrom(ctx, types.StringType, avail)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	state.AvailableVersions = availList

	vpList, d := types.MapValueFrom(ctx, types.StringType, assigned)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	state.VersionPackages = vpList

	return diags
}

// categoryValues maps an SDK category object onto (id, name) Terraform strings.
// A nil category yields null/null. The endpoint reports "no category" as id -1
// (never assigned) or id 0 (explicitly cleared via the buildPatchSoftwareTitleUpdateInput
// wire translation); both collapse to the "-1" user-facing sentinel so state
// matches config regardless of which the server echoes.
func categoryValues(c *proclassic.CategoryObject) (types.String, types.String) {
	if c == nil {
		return types.StringNull(), types.StringNull()
	}
	if c.ID == nil || *c.ID <= 0 {
		return types.StringValue("-1"), helpers.StringPointerValueOrNull(c.Name)
	}
	return helpers.StringValueFromIntPtr(c.ID), helpers.StringPointerValueOrNull(c.Name)
}

// siteValues maps an SDK site object onto (id, name) Terraform strings.
func siteValues(s *proclassic.SiteObject) (types.String, types.String) {
	if s == nil {
		return types.StringNull(), types.StringNull()
	}
	return helpers.StringValueFromIntPtr(s.ID), helpers.StringPointerValueOrNull(s.Name)
}

// notificationValues maps the notifications block onto (web, email) bools. A nil
// block yields null/null.
func notificationValues(n *proclassic.PatchSoftwareTitleNotifications) (types.Bool, types.Bool) {
	if n == nil {
		return types.BoolNull(), types.BoolNull()
	}
	return helpers.BoolPointerValueOrNull(n.WebNotification), helpers.BoolPointerValueOrNull(n.EmailNotification)
}

// availableVersions returns every software_version string the server publishes
// for the title, preserving server order (newest first).
func availableVersions(v *proclassic.PatchSoftwareTitleVersions) []string {
	if v == nil || v.Version == nil {
		return []string{}
	}
	out := make([]string, 0, len(*v.Version))
	for _, item := range *v.Version {
		if item.SoftwareVersion != nil {
			out = append(out, *item.SoftwareVersion)
		}
	}
	return out
}

// assignedPackagesByVersion returns a software_version → package_id map for every
// server version that has a non-empty package id assigned.
func assignedPackagesByVersion(v *proclassic.PatchSoftwareTitleVersions) map[string]string {
	out := map[string]string{}
	if v == nil || v.Version == nil {
		return out
	}
	for _, item := range *v.Version {
		if item.SoftwareVersion == nil || item.Package == nil || item.Package.ID == nil {
			continue
		}
		out[*item.SoftwareVersion] = strconv.Itoa(*item.Package.ID)
	}
	return out
}

// managedVersionPackages builds the version_packages map from only the declared
// keys: each is looked up in the server's assigned set and included if still
// present. A declared key with no server-side package is dropped (surfaces
// drift). When no keys are declared the map is null (matches an unset config).
func managedVersionPackages(ctx context.Context, declaredKeys []string, assigned map[string]string) (types.Map, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(declaredKeys) == 0 {
		return types.MapNull(types.StringType), diags
	}
	managed := map[string]string{}
	for _, k := range declaredKeys {
		if pkg, ok := assigned[k]; ok {
			managed[k] = pkg
		}
	}
	if len(managed) == 0 {
		return types.MapNull(types.StringType), diags
	}
	m, d := types.MapValueFrom(ctx, types.StringType, managed)
	diags.Append(d...)
	return m, diags
}
