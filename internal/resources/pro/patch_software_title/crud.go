// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// SDK endpoints used:
//   proclassic.CreatePatchSoftwareTitleByID                (POST id="0" mints the title)
//   proclassic.GetPatchSoftwareTitleByID
//   proclassic.UpdatePatchSoftwareTitleByID                (PUT, returns 201 empty body → GET after)
//   proclassic.DeletePatchSoftwareTitleByID
//   proclassic.ListPatchSoftwareTitles                     (list resource)
//   pro.ListPatchSoftwareTitleExtensionAttributesV3        (read EAs; v3 config id == classic title id)
//   pro.UpdatePatchSoftwareTitleConfigurationV3            (accept EAs; merge-patch, accept is one-way)
//
// DEPRECATION (classic CRUD): the /patchsoftwaretitles classic endpoints are
// flagged deprecated in the Jamf API spec (the SDK funcs carry `// Deprecated:`
// pointing at /v2/patch-software-title-configurations). They remain the only
// functional create surface, and not merely for want of a shipped successor —
// the configurations POST cannot be one by construction. Wire-probed 2026-09-01
// on Jamf Pro 11.31.1: its required softwareTitleId is the classic title id
// itself, the catalogue's name_id is refused (400 SOFTWARE_TITLE_ID_NOT_FOUND on
// field softwareTitleId), and the only way to mint an id — classic POST to
// id="0" — creates the v3 configuration in the same act, so a follow-up v3 POST
// with that id answers 400 ALREADY_EXISTS_FOR_SITE. GET
// /proclassic/patchsoftwaretitles still answers 200. Revisit only if Jamf gives
// the configurations surface an id-minting create, or removes the classic
// endpoints.
//
// EXTENSION-ATTRIBUTE SIDE-CHANNEL (v3): the v2 configurations client was
// withdrawn from the SDK at this bump and the two calls in
// extension_attributes.go moved to v3, closing the
// patch-software-title-configurations half of #311. v3 is the top version rather
// than an intermediate step — wire-probed 2026-09-01 on Jamf Pro 11.31.1,
// /pro/v3/patch-software-title-configurations answers 200 with no Deprecation
// header, /pro/v2 still answers 200 carrying `deprecation: date="Tue, 14 Jul 2026
// 00:00:00 GMT"` (so the SDK withdrawal was spec-only), and /pro/v4 answers 403
// BAD_PERMISSIONS, which this repo reads as unrouted rather than unprivileged.
//
// Note the two deprecations point in opposite directions: the classic endpoints
// are deprecated in favour of the configurations endpoints, whose v2 is itself
// deprecated. Read and update could move to v3 today; create cannot, per the
// probe above, so the surfaces cannot be retired as one.
//
// Status: extension-attribute side-channel current on v3. Classic CRUD deprecated
// by Jamf with no published removal date, retained deliberately because the
// configurations surface has no id-minting create. Last reviewed 2026-09-01.

package patch_software_title

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// Create mints a new patch software title then applies metadata + package
// assignments. The classic POST to id="0" allocates the real ID and seeds the
// full versions catalog from the patch definition (name_id + source_id define
// the title). category/site/notifications/version_packages are then applied via
// a single follow-up PUT, after which we GET to refresh state.
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

	created, err := r.client.CreatePatchSoftwareTitleByID(createCtx, "0", buildPatchSoftwareTitleCreateInput(plan)) //nolint:staticcheck // SA1019: classic /patchsoftwaretitles intentionally used; v2 create unusable — see file header note
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

	// Apply metadata + package assignments. priorKeys is empty on Create, so the
	// builder only emits assigns (no clears).
	putBody, putDiags := buildPatchSoftwareTitleUpdateInput(createCtx, plan, nil)
	resp.Diagnostics.Append(putDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.UpdatePatchSoftwareTitleByID(createCtx, id.ValueString(), putBody); err != nil { //nolint:staticcheck // SA1019: classic /patchsoftwaretitles intentionally used; v2 create unusable — see file header note
		resp.Diagnostics.AddError("Error applying Jamf Pro patch software title settings", err.Error())
		return
	}

	got, err := r.client.GetPatchSoftwareTitleByID(createCtx, id.ValueString()) //nolint:staticcheck // SA1019: classic /patchsoftwaretitles intentionally used; v2 create unusable — see file header note
	if err != nil {
		resp.Diagnostics.AddError("Error reading created Jamf Pro patch software title", err.Error())
		return
	}

	declaredKeys, keyDiags := versionPackageKeys(createCtx, plan.VersionPackages)
	resp.Diagnostics.Append(keyDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(assignPatchSoftwareTitleResourceModel(createCtx, &plan, got, declaredKeys)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read the title's extension attributes (best-effort warning on failure).
	resp.Diagnostics.Append(r.refreshExtensionAttributes(createCtx, id.ValueString(), &plan)...)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, patchSoftwareTitleIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Persist state before attempting the accept side-call: the classic title is
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
// reconstruct the managed subset from).
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

	got, err := r.client.GetPatchSoftwareTitleByID(readCtx, state.ID.ValueString()) //nolint:staticcheck // SA1019: classic /patchsoftwaretitles intentionally used; v2 create unusable — see file header note
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

	resp.Diagnostics.Append(assignPatchSoftwareTitleResourceModel(readCtx, &state, got, declaredKeys)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.refreshExtensionAttributes(readCtx, state.ID.ValueString(), &state)...)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, patchSoftwareTitleIdentityModel{ID: state.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update applies metadata + package changes. The PUT carries one assign entry
// per plan key and one empty-package clear entry per key dropped from prior
// state (verified: empty <package></package> clears, omitted package retains).
// UpdatePatchSoftwareTitleByID returns 201 with an empty body, so we GET after.
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

	putBody, putDiags := buildPatchSoftwareTitleUpdateInput(updateCtx, plan, priorKeys)
	resp.Diagnostics.Append(putDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.UpdatePatchSoftwareTitleByID(updateCtx, plan.ID.ValueString(), putBody); err != nil { //nolint:staticcheck // SA1019: classic /patchsoftwaretitles intentionally used; v2 create unusable — see file header note
		resp.Diagnostics.AddError("Error updating Jamf Pro patch software title", err.Error())
		return
	}

	got, err := r.client.GetPatchSoftwareTitleByID(updateCtx, plan.ID.ValueString()) //nolint:staticcheck // SA1019: classic /patchsoftwaretitles intentionally used; v2 create unusable — see file header note
	if err != nil {
		resp.Diagnostics.AddError("Error reading updated Jamf Pro patch software title", err.Error())
		return
	}

	declaredKeys, keyDiags := versionPackageKeys(updateCtx, plan.VersionPackages)
	resp.Diagnostics.Append(keyDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(assignPatchSoftwareTitleResourceModel(updateCtx, &plan, got, declaredKeys)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.refreshExtensionAttributes(updateCtx, plan.ID.ValueString(), &plan)...)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, patchSoftwareTitleIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Persist the classic changes before the accept side-call so a fatal accept
	// failure does not lose them; a later apply retries the accept.
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

// Delete removes a Jamf Pro patch software title.
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

	if err := r.client.DeletePatchSoftwareTitleByID(deleteCtx, state.ID.ValueString()); err != nil { //nolint:staticcheck // SA1019: classic /patchsoftwaretitles intentionally used; v2 create unusable — see file header note
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro patch software title already removed", map[string]any{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError("Error deleting Jamf Pro patch software title", fmt.Sprintf("API error: %v", err))
	}
}
