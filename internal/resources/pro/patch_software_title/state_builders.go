// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_software_title

import (
	"context"
	"strconv"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// assignPatchSoftwareTitleResourceModel populates a resource model from a v3
// patch software title configuration plus the version catalog, which lives on
// the separate /definitions sub-resource rather than the configuration body.
// declaredKeys is the managed subset of software_version keys the caller
// declared (plan keys on Create/Update, prior state keys on Read):
// version_packages is rebuilt from only those keys by looking each up in the
// configuration's package assignments. A declared key whose package is gone
// server-side is dropped from the map, surfacing the drift.
//
// source_id is deliberately left untouched. The v3 configuration names its
// patch source (patchSourceName) but never numbers it, and a title's source
// cannot change once minted, so whatever is already in state stays correct.
// Import is the one case with nothing in state to keep, and Read resolves the
// number from the name there — see resolveSourceID.
func assignPatchSoftwareTitleResourceModel(ctx context.Context, state *PatchSoftwareTitleResourceModel, s *pro.PatchSoftwareTitleConfiguration, availableVersions, declaredKeys []string) diag.Diagnostics {
	var diags diag.Diagnostics
	if s == nil {
		return diags
	}
	if s.ID != "" {
		state.ID = types.StringValue(s.ID)
	}
	state.Name = types.StringValue(s.DisplayName)
	if s.SoftwareTitleNameID != "" {
		state.NameID = types.StringValue(s.SoftwareTitleNameID)
	}

	state.CategoryID = refIDValue(s.CategoryID)
	state.SiteID = refIDValue(s.SiteID)
	state.WebNotification = types.BoolValue(s.UiNotifications)
	state.EmailNotification = types.BoolValue(s.EmailNotifications)

	availList, d := types.ListValueFrom(ctx, types.StringType, orEmpty(availableVersions))
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	state.AvailableVersions = availList

	vp, d := managedVersionPackages(ctx, declaredKeys, assignedPackagesByVersion(s.Packages))
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	state.VersionPackages = vp

	return diags
}

// assignPatchSoftwareTitleDataSourceModel populates a data source model. The
// data source surfaces the full server view, so version_packages is built from
// every assignment the configuration reports (no managed-subset gating — there
// is no prior state to scope against). sourceID is resolved by the caller from
// the configuration's patch source name.
func assignPatchSoftwareTitleDataSourceModel(ctx context.Context, state *PatchSoftwareTitleDataSourceModel, s *pro.PatchSoftwareTitleConfiguration, availableVersions []string, sourceID types.Int64) diag.Diagnostics {
	var diags diag.Diagnostics
	if s == nil {
		return diags
	}
	state.ID = types.StringValue(s.ID)
	state.Name = types.StringValue(s.DisplayName)
	state.NameID = types.StringValue(s.SoftwareTitleNameID)
	state.SourceID = sourceID

	state.CategoryID = refIDValue(s.CategoryID)
	state.SiteID = refIDValue(s.SiteID)
	state.WebNotification = types.BoolValue(s.UiNotifications)
	state.EmailNotification = types.BoolValue(s.EmailNotifications)

	availList, d := types.ListValueFrom(ctx, types.StringType, orEmpty(availableVersions))
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	state.AvailableVersions = availList

	vpMap, d := types.MapValueFrom(ctx, types.StringType, assignedPackagesByVersion(s.Packages))
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	state.VersionPackages = vpMap

	return diags
}

// refIDValue maps a v3 category or site id onto its Terraform value. The
// configurations endpoint reports "not assigned" as the string "-1" and rejects
// any other non-positive id on a write, so that sentinel round-trips as-is.
// Two other shapes are normalised to it defensively: an empty string (a field
// the server omitted), and the literal "0" — which no v3 write can produce but
// a title last written through the classic /patchsoftwaretitles endpoint can
// still carry, because that endpoint cleared a category by storing wire id 0.
func refIDValue(id string) types.String {
	if id == "" {
		return types.StringValue("-1")
	}
	if n, err := strconv.Atoi(id); err == nil && n <= 0 {
		return types.StringValue("-1")
	}
	return types.StringValue(id)
}

// definitionVersions returns every version string the patch source publishes for
// the title, preserving server order. The /definitions default sort is
// absoluteOrderId:asc, which Jamf orders newest-first — the same order the
// classic <versions> block used, so available_versions needs no re-sorting.
func definitionVersions(defs []pro.PatchSoftwareTitleDefinition) []string {
	out := make([]string, 0, len(defs))
	for i := range defs {
		if defs[i].Version != "" {
			out = append(out, defs[i].Version)
		}
	}
	return out
}

// assignedPackagesByVersion returns a software_version → package_id map for
// every package assignment the configuration reports.
func assignedPackagesByVersion(pkgs []pro.PatchSoftwareTitlePackages) map[string]string {
	out := map[string]string{}
	for i := range pkgs {
		if pkgs[i].Version == nil || pkgs[i].PackageID == nil {
			continue
		}
		if *pkgs[i].Version == "" || *pkgs[i].PackageID == "" {
			continue
		}
		out[*pkgs[i].Version] = *pkgs[i].PackageID
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

// orEmpty substitutes an empty slice for nil so a Computed list attribute lands
// as an empty list rather than null.
func orEmpty(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}
