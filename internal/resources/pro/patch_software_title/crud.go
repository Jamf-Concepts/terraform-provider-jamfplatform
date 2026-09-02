// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// SDK endpoints used:
//   proclassic.CreatePatchSoftwareTitleByID                (POST id="0" mints the title — the only classic call left)
//   proclassic.ListPatchInternalSources                    (source_id resolution: name → id, import/data source only)
//   proclassic.ListPatchExternalSources                    (source_id resolution: name → id, import/data source only)
//   pro.GetPatchSoftwareTitleConfigurationV3               (read; v3 configuration id == classic title id)
//   pro.UpdatePatchSoftwareTitleConfigurationV3            (merge-patch; returns the full object, so no GET after)
//   pro.DeletePatchSoftwareTitleConfigurationV3            (204; removes the classic title with it)
//   pro.ListPatchSoftwareTitleConfigurationsV3             (data source name lookup, list resource)
//   pro.ListPatchSoftwareTitleDefinitionsV3                (available_versions — not on the configuration body)
//   pro.ListPatchSoftwareTitleExtensionAttributesV3        (read EAs)
//
// CREATE IS THE ONLY CLASSIC CALL, and not for want of a shipped successor —
// the configurations POST cannot be one by construction. Wire-probed 2026-09-01
// on Jamf Pro 11.31.1: its required softwareTitleId is the classic title id
// itself, the catalogue's name_id is refused (400 SOFTWARE_TITLE_ID_NOT_FOUND on
// field softwareTitleId), and the only way to mint an id — classic POST to
// id="0" — creates the v3 configuration in the same act, so a follow-up v3 POST
// with that id answers 400 ALREADY_EXISTS_FOR_SITE. The classic endpoints are
// flagged deprecated in the Jamf API spec (the SDK func carries `// Deprecated:`
// pointing at /v2/patch-software-title-configurations), so the one remaining
// call is annotated rather than removed. Revisit only if Jamf gives the
// configurations surface an id-minting create.
//
// Everything else moved to v3, wire-probed 2026-09-02 on Jamf Pro 11.31.1 in
// production EU. Four behaviours drive the code and are easy to get wrong:
//
//   - The merge-patch needs Content-Type application/merge-patch+json; plain
//     application/json is refused 415. The SDK sets it.
//   - PATCH answers 200 carrying the full configuration, unlike the classic PUT
//     (201, empty body), so Update no longer reads back what it just wrote.
//   - `packages` is a FULL REPLACEMENT. A supplied array is the complete set of
//     assignments — anything absent from it is cleared, and `[]` clears the lot —
//     while omitting the key leaves the server's set untouched. The classic
//     endpoint merged per version instead, which is what let version_packages
//     promise that assignments made outside Terraform survive an apply. Update
//     therefore reads the live set and folds the plan over it (unionVersionPackages)
//     rather than sending the plan alone.
//   - categoryId / siteId accept a positive id or the literal "-1" and reject
//     everything else, "0" included. This retires the classic endpoint's
//     opposite convention (0 cleared a category, -1 was a silent no-op) along
//     with the translation either way.
//
// Two things v3 does not carry: source_id (it names the patch source but never
// numbers it — see resolveSourceID) and the version catalogue behind
// available_versions (a separate /definitions read, whose default
// absoluteOrderId:asc order is the newest-first order the classic <versions>
// block used, so no re-sorting).
//
// /pro/v3 answers 200 with no Deprecation header; /pro/v2 still answers 200
// carrying `deprecation: date="Tue, 14 Jul 2026 00:00:00 GMT"`; /pro/v4 answers
// 403 BAD_PERMISSIONS, which this repo reads as unrouted rather than
// unprivileged. v3 is therefore the top version, not an intermediate step.
//
// Status: current on v3 but for the id-minting create, which has no successor.
// Last reviewed 2026-09-02.

package patch_software_title

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// Create mints a new patch software title then applies metadata + package
// assignments. The classic POST to id="0" allocates the real ID and seeds the
// full versions catalog from the patch definition (name_id + source_id define
// the title), creating the v3 configuration in the same act.
// category/site/notifications/version_packages are then applied through a
// single v3 merge-patch, which answers with the stored configuration.
func (r *PatchSoftwareTitleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PatchSoftwareTitleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, plan.Timeouts.IsNull(), plan.Timeouts.IsUnknown(), defaultCreateTimeout, plan.Timeouts.Create)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	createCtx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	created, err := r.client.CreatePatchSoftwareTitleByID(createCtx, "0", buildPatchSoftwareTitleCreateInput(plan)) //nolint:staticcheck // SA1019: classic /patchsoftwaretitles is the only id-minting create — see file header note
	if err != nil {
		resp.Diagnostics.AddError("Error creating Jamf Pro patch software title", err.Error())
		return
	}
	if created == nil || created.ID == nil {
		resp.Diagnostics.AddError(
			"Create response missing patch software title ID",
			"Jamf Pro returned 201 Created with no patch software title ID; cannot persist state.",
		)
		return
	}
	id := helpers.StringValueFromIntPtr(created.ID)
	plan.ID = id

	planPackages, pkgDiags := versionPackageMap(createCtx, plan.VersionPackages)
	resp.Diagnostics.Append(pkgDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A title minted moments ago has no assignments, so the desired set is
	// already the complete set — no live read to fold over. Nothing configured
	// means nothing to send, so the key is omitted entirely.
	var desired map[string]string
	if len(planPackages) > 0 {
		desired = planPackages
	}

	got, err := r.proClient.UpdatePatchSoftwareTitleConfigurationV3(createCtx, id.ValueString(), buildPatchSoftwareTitleConfigurationPatch(plan, desired))
	if err != nil {
		resp.Diagnostics.AddError("Error applying Jamf Pro patch software title settings", err.Error())
		return
	}

	avail, availDiags := r.readAvailableVersions(createCtx, id.ValueString())
	resp.Diagnostics.Append(availDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(assignPatchSoftwareTitleResourceModel(createCtx, &plan, got, avail, keysOf(planPackages))...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read the title's extension attributes (best-effort warning on failure).
	resp.Diagnostics.Append(r.refreshExtensionAttributes(createCtx, id.ValueString(), &plan)...)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, patchSoftwareTitleIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Persist state before attempting the accept side-call: the title is
	// already minted, so a fatal accept failure must still leave it in state
	// (else it orphans server-side). A later apply then retries via Update.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.AcceptExtensionAttributes.ValueBool() {
		if err := r.acceptPendingExtensionAttributes(createCtx, id.ValueString(), plan.CategoryID.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error accepting Jamf Pro patch software title extension attributes", err.Error())
			return
		}
		resp.Diagnostics.Append(r.refreshExtensionAttributes(createCtx, id.ValueString(), &plan)...)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	tflog.Trace(ctx, "created Jamf Pro patch software title", map[string]any{"id": plan.ID.ValueString()})
}

// Read refreshes state. version_packages is rebuilt from only the keys recorded
// in prior state (the managed subset). On import there is no prior state, so the
// map comes back null and ImportStateVerify must ignore it (no prior keys to
// reconstruct the managed subset from) — and source_id, which the v3 payload
// does not carry, is resolved from the reported patch source name.
func (r *PatchSoftwareTitleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PatchSoftwareTitleResourceModel
	isImport := req.State.Raw.IsNull()

	if isImport {
		if req.Identity == nil {
			resp.Diagnostics.AddError(
				"Missing resource identity",
				"Terraform requested a refresh for this patch software title without existing state or identity data, so the provider cannot determine which title to read.",
			)
			return
		}
		var identity patchSoftwareTitleIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.ID.IsNull() || identity.ID.IsUnknown() || identity.ID.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Missing patch software title ID",
				"The resource identity did not include an 'id' attribute, so the provider cannot refresh the patch software title.",
			)
			return
		}
		state.ID = identity.ID
		state.Timeouts = helpers.NewResourceTimeoutsNullValue(patchSoftwareTitleTimeoutAttributeTypes)
	} else {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	readTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, state.Timeouts.IsNull(), state.Timeouts.IsUnknown(), defaultReadTimeout, state.Timeouts.Read)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	if state.ID.IsNull() || state.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot read Jamf Pro patch software title without ID.")
		return
	}

	declaredKeys, keyDiags := versionPackageKeys(readCtx, state.VersionPackages)
	resp.Diagnostics.Append(keyDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	got, err := r.proClient.GetPatchSoftwareTitleConfigurationV3(readCtx, state.ID.ValueString())
	if err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro patch software title not found, removing from state", map[string]any{"id": state.ID.ValueString()})
			resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, patchSoftwareTitleIdentityModel{ID: state.ID})...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Jamf Pro patch software title", err.Error())
		return
	}

	avail, availDiags := r.readAvailableVersions(readCtx, state.ID.ValueString())
	resp.Diagnostics.Append(availDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// source_id is carried forward by assignPatchSoftwareTitleResourceModel,
	// which never touches it. Resolve it from the reported source name only when
	// state has no number to carry — an import. It backs a Required,
	// RequiresReplace attribute, so a wrong or absent value would show up as a
	// plan that destroys the freshly imported title; a failure here is fatal
	// rather than a warning for that reason.
	needsSource := state.SourceID.IsNull() || state.SourceID.IsUnknown()

	resp.Diagnostics.Append(assignPatchSoftwareTitleResourceModel(readCtx, &state, got, avail, declaredKeys)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if needsSource {
		sourceID, err := resolveSourceID(readCtx, r.client, got.PatchSourceName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to determine the patch software title's source_id",
				fmt.Sprintf("Jamf Pro reports this title's patch source as %q, but the provider could not match that name to a patch source ID: %v", got.PatchSourceName, err),
			)
			return
		}
		state.SourceID = sourceID
	}

	resp.Diagnostics.Append(r.refreshExtensionAttributes(readCtx, state.ID.ValueString(), &state)...)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, patchSoftwareTitleIdentityModel{ID: state.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update applies metadata + package changes through a single v3 merge-patch,
// which answers with the stored configuration.
//
// Because the v3 `packages` array is a full replacement, the desired
// assignments are folded over the server's live set so that assignments made
// outside Terraform survive — the managed-subset contract version_packages
// documents. That fold needs the live set, so it costs a read; when neither the
// plan nor prior state declares any assignment there is nothing to manage and
// the key is omitted instead, skipping the read.
func (r *PatchSoftwareTitleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state PatchSoftwareTitleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, plan.Timeouts.IsNull(), plan.Timeouts.IsUnknown(), defaultUpdateTimeout, plan.Timeouts.Update)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	updateCtx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	priorKeys, keyDiags := versionPackageKeys(updateCtx, state.VersionPackages)
	resp.Diagnostics.Append(keyDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	planPackages, pkgDiags := versionPackageMap(updateCtx, plan.VersionPackages)
	resp.Diagnostics.Append(pkgDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var desired map[string]string
	if len(planPackages) > 0 || len(priorKeys) > 0 {
		live, err := r.proClient.GetPatchSoftwareTitleConfigurationV3(updateCtx, plan.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading Jamf Pro patch software title package assignments", err.Error())
			return
		}
		desired = unionVersionPackages(assignedPackagesByVersion(live.Packages), planPackages, priorKeys)
	}

	got, err := r.proClient.UpdatePatchSoftwareTitleConfigurationV3(updateCtx, plan.ID.ValueString(), buildPatchSoftwareTitleConfigurationPatch(plan, desired))
	if err != nil {
		resp.Diagnostics.AddError("Error updating Jamf Pro patch software title", err.Error())
		return
	}

	avail, availDiags := r.readAvailableVersions(updateCtx, plan.ID.ValueString())
	resp.Diagnostics.Append(availDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(assignPatchSoftwareTitleResourceModel(updateCtx, &plan, got, avail, keysOf(planPackages))...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.refreshExtensionAttributes(updateCtx, plan.ID.ValueString(), &plan)...)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, patchSoftwareTitleIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Persist the configuration changes before the accept side-call so a fatal
	// accept failure does not lose them; a later apply retries the accept.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.AcceptExtensionAttributes.ValueBool() {
		if err := r.acceptPendingExtensionAttributes(updateCtx, plan.ID.ValueString(), plan.CategoryID.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error accepting Jamf Pro patch software title extension attributes", err.Error())
			return
		}
		resp.Diagnostics.Append(r.refreshExtensionAttributes(updateCtx, plan.ID.ValueString(), &plan)...)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}

// Delete removes a Jamf Pro patch software title. Deleting the v3 configuration
// removes the classic title with it — wire-probed 2026-09-02: the DELETE answers
// 204 and the classic GET then answers 404, so no classic-side object is left
// behind.
func (r *PatchSoftwareTitleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PatchSoftwareTitleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, state.Timeouts.IsNull(), state.Timeouts.IsUnknown(), defaultDeleteTimeout, state.Timeouts.Delete)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	deleteCtx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	if state.ID.IsNull() || state.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot delete Jamf Pro patch software title without ID.")
		return
	}

	if err := r.proClient.DeletePatchSoftwareTitleConfigurationV3(deleteCtx, state.ID.ValueString()); err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro patch software title already removed", map[string]any{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError("Error deleting Jamf Pro patch software title", fmt.Sprintf("API error: %v", err))
	}
}

// readAvailableVersions reads the title's version catalogue from the
// /definitions sub-resource, which is where v3 keeps it — the configuration
// body carries only the versions that have a package assigned.
func (r *PatchSoftwareTitleResource) readAvailableVersions(ctx context.Context, id string) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	defs, err := r.proClient.ListPatchSoftwareTitleDefinitionsV3(ctx, id, nil, "")
	if err != nil {
		diags.AddError("Error reading Jamf Pro patch software title versions", err.Error())
		return nil, diags
	}
	return definitionVersions(defs), diags
}

// keysOf returns a map's keys, or nil for an empty map so callers reading it as
// a declared-key set see "nothing declared".
func keysOf(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
