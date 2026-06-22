// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_software_title

import (
	"context"
	"sort"
	"strconv"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// buildPatchSoftwareTitleCreateInput builds the minimal POST payload that mints a
// title. Per the verified wire semantics, name + name_id + source_id define the
// title; the server then populates the full versions catalog from the patch
// definition. category/site/notifications/version_packages are applied via a
// follow-up PUT (see crud.go Create).
func buildPatchSoftwareTitleCreateInput(plan PatchSoftwareTitleResourceModel) *proclassic.PatchSoftwareTitle {
	out := &proclassic.PatchSoftwareTitle{}
	if v := plan.Name.ValueString(); !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		name := v
		out.Name = &name
	}
	if v := plan.NameID.ValueString(); !plan.NameID.IsNull() && !plan.NameID.IsUnknown() {
		nameID := v
		out.NameID = &nameID
	}
	if !plan.SourceID.IsNull() && !plan.SourceID.IsUnknown() {
		sourceID := int(plan.SourceID.ValueInt64())
		out.SourceID = &sourceID
	}
	return out
}

// buildPatchSoftwareTitleUpdateInput builds the full PUT payload carrying the
// desired metadata (name, category, site, notifications) plus the per-version
// package operations. priorKeys are the version_packages keys recorded in the
// prior Terraform state (empty on Create); plan is the desired config.
//
// Per the verified wire semantics the server merges versions by software_version:
//   - a version carrying <package><id>N</id></package> assigns package N
//   - a version carrying an empty <package></package> CLEARS the assignment
//   - a version omitted from the payload is left untouched
//
// We therefore emit one entry per plan key (assign) and one empty-package entry
// per key that was in prior state but dropped from the plan (clear/unassign).
func buildPatchSoftwareTitleUpdateInput(ctx context.Context, plan PatchSoftwareTitleResourceModel, priorKeys []string) (*proclassic.PatchSoftwareTitle, diag.Diagnostics) {
	var diags diag.Diagnostics

	out := &proclassic.PatchSoftwareTitle{}
	if v := plan.Name.ValueString(); !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		name := v
		out.Name = &name
	}
	if id := helpers.StringIDPtr(plan.CategoryID); id != nil {
		// This endpoint clears a category with wire id 0 — id -1 is a silent
		// no-op here (it does clear <site>, but not <category>). Map the
		// user-facing "-1 = No category assigned" sentinel (and any non-positive
		// id) to the wire's 0 clear; pass real category ids through unchanged.
		// categoryValues collapses the server's -1/0 no-category echoes back to
		// "-1" on read so state matches config. Wire-probed 2026-06-01.
		wireID := max(*id, 0)
		out.Category = &proclassic.CategoryObject{ID: &wireID}
	}
	if id := helpers.StringIDPtr(plan.SiteID); id != nil {
		out.Site = &proclassic.SiteObject{ID: id}
	}
	if n := buildNotifications(plan); n != nil {
		out.Notifications = n
	}

	planPackages := map[string]string{}
	if !plan.VersionPackages.IsNull() && !plan.VersionPackages.IsUnknown() {
		diags.Append(plan.VersionPackages.ElementsAs(ctx, &planPackages, false)...)
		if diags.HasError() {
			return nil, diags
		}
	}

	versions := buildVersionItems(planPackages, priorKeys)
	if len(versions) > 0 {
		out.Versions = &proclassic.PatchSoftwareTitleVersions{Version: &versions}
	}

	return out, diags
}

// buildNotifications returns a notifications block when either toggle is a known
// value, or nil so the SDK omitempty drops it (server keeps / defaults). Each
// configured bool is sent explicitly (false is meaningful).
func buildNotifications(plan PatchSoftwareTitleResourceModel) *proclassic.PatchSoftwareTitleNotifications {
	var web, email *bool
	if !plan.WebNotification.IsNull() && !plan.WebNotification.IsUnknown() {
		v := plan.WebNotification.ValueBool()
		web = &v
	}
	if !plan.EmailNotification.IsNull() && !plan.EmailNotification.IsUnknown() {
		v := plan.EmailNotification.ValueBool()
		email = &v
	}
	if web == nil && email == nil {
		return nil
	}
	return &proclassic.PatchSoftwareTitleNotifications{
		WebNotification:   web,
		EmailNotification: email,
	}
}

// buildVersionItems produces the merge-by-software_version version entries:
//   - one assign entry per plan key (software_version + package id)
//   - one clear entry (empty package) per prior-state key absent from the plan
//
// A non-nil but empty *PatchSoftwareTitleVersionsVersionItemPackage marshals to
// an empty <package></package> element, which the server treats as "clear this
// version's package" (verified live). Omitting the package element entirely
// would instead retain the existing assignment, so clears must emit the empty
// element. Entries are sorted by software_version for deterministic payloads.
func buildVersionItems(planPackages map[string]string, priorKeys []string) []proclassic.PatchSoftwareTitleVersionsVersionItem {
	type op struct {
		sv     string
		pkgID  *int
		assign bool
	}
	ops := map[string]op{}

	for sv, pkgStr := range planPackages {
		if n, err := strconv.Atoi(pkgStr); err == nil {
			id := n
			ops[sv] = op{sv: sv, pkgID: &id, assign: true}
		}
	}
	for _, sv := range priorKeys {
		if _, stillPlanned := planPackages[sv]; stillPlanned {
			continue
		}
		ops[sv] = op{sv: sv, assign: false}
	}

	keys := make([]string, 0, len(ops))
	for k := range ops {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	items := make([]proclassic.PatchSoftwareTitleVersionsVersionItem, 0, len(keys))
	for _, k := range keys {
		o := ops[k]
		sv := o.sv
		item := proclassic.PatchSoftwareTitleVersionsVersionItem{SoftwareVersion: &sv}
		if o.assign {
			item.Package = &proclassic.PatchSoftwareTitleVersionsVersionItemPackage{ID: o.pkgID}
		} else {
			// Empty (but non-nil) package element clears the assignment.
			item.Package = &proclassic.PatchSoftwareTitleVersionsVersionItemPackage{}
		}
		items = append(items, item)
	}
	return items
}

// versionPackageKeys returns the declared version_packages keys from a model's
// map, used as the managed-subset key set for Read reconciliation and Update
// clear-diffing. Returns nil for null/unknown maps.
func versionPackageKeys(ctx context.Context, m types.Map) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if m.IsNull() || m.IsUnknown() {
		return nil, diags
	}
	elems := map[string]string{}
	diags.Append(m.ElementsAs(ctx, &elems, false)...)
	if diags.HasError() {
		return nil, diags
	}
	keys := make([]string, 0, len(elems))
	for k := range elems {
		keys = append(keys, k)
	}
	return keys, diags
}
