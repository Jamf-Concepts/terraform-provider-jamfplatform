// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// SDK endpoints used:
//   proclassic.CreateMacApplicationByID       (POST /macapplications/id/0)
//   proclassic.GetMacApplicationByID          (GET  /macapplications/id/{id})
//   proclassic.UpdateMacApplicationByID       (PUT  /macapplications/id/{id})
//   proclassic.DeleteMacApplicationByID       (DELETE)
//   proclassic.ListMacApplications            (data source / list resource)
//   proclassic.GetMacApplicationByName        (data source name lookup)
//
// Status: current. Last reviewed 2026-05-31.
//
// Update semantics: the classic /macapplications PUT is a partial-merge, not a
// full-replace (wire-probed). The provider sends the full plan payload on every
// Update, so in-place edits to managed sections converge cleanly. Removing an
// entire optional block (scope / self_service / vpp) from config omits it from
// the payload, so the server retains the previously-stored block — a known
// limitation matching ProClassic precedent. To clear a block, null its
// individual fields rather than deleting the block.

package mac_app_store_app

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// Create creates a new Jamf Pro Mac App Store app. Classic POSTs to id="0"; the
// server allocates the real integer ID and returns it in the response body.
func (r *MacAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MacAppResourceModel
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

	payload, buildDiags := buildMacAppInput(createCtx, plan)
	resp.Diagnostics.Append(buildDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateMacApplicationByID(createCtx, "0", payload)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Jamf Pro Mac App Store app", err.Error())
		return
	}
	id := extractMacAppID(created)
	if id == "" {
		resp.Diagnostics.AddError(
			"Create response missing app ID",
			"Jamf Pro returned 201 Created with no app ID; cannot persist state.",
		)
		return
	}

	got, err := r.client.GetMacApplicationByID(createCtx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading created Jamf Pro Mac App Store app", err.Error())
		return
	}
	resp.Diagnostics.Append(assignMacAppResourceModel(createCtx, &plan, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, macAppIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "created Jamf Pro Mac App Store app", map[string]any{"id": plan.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes Terraform state with the latest app representation.
func (r *MacAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MacAppResourceModel
	isImport := req.State.Raw.IsNull()

	if isImport {
		if req.Identity == nil {
			resp.Diagnostics.AddError(
				"Missing resource identity",
				"Terraform requested a refresh for this app without existing state or identity data, so the provider cannot determine which app to read.",
			)
			return
		}
		var identity macAppIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.ID.IsNull() || identity.ID.IsUnknown() || identity.ID.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Missing app ID",
				"The resource identity did not include an 'id' attribute, so the provider cannot refresh the app.",
			)
			return
		}
		state.ID = identity.ID
		state.Timeouts = helpers.NewResourceTimeoutsNullValue(macAppTimeoutAttributeTypes)
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
		resp.Diagnostics.AddError("Missing ID", "Cannot read Jamf Pro Mac App Store app without ID.")
		return
	}

	got, err := r.client.GetMacApplicationByID(readCtx, state.ID.ValueString())
	if err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro Mac App Store app not found, removing from state", map[string]any{"id": state.ID.ValueString()})
			resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, macAppIdentityModel{ID: state.ID})...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Jamf Pro Mac App Store app", err.Error())
		return
	}

	resp.Diagnostics.Append(assignMacAppResourceModel(readCtx, &state, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, macAppIdentityModel{ID: state.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates a Jamf Pro Mac App Store app. Classic UpdateMacApplicationByID
// returns 201 with an empty body — we must GET to refresh state.
func (r *MacAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan MacAppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
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

	payload, buildDiags := buildMacAppInput(updateCtx, plan)
	resp.Diagnostics.Append(buildDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdateMacApplicationByID(updateCtx, plan.ID.ValueString(), payload); err != nil {
		resp.Diagnostics.AddError("Error updating Jamf Pro Mac App Store app", err.Error())
		return
	}

	got, err := r.client.GetMacApplicationByID(updateCtx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading updated Jamf Pro Mac App Store app", err.Error())
		return
	}
	resp.Diagnostics.Append(assignMacAppResourceModel(updateCtx, &plan, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, macAppIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes a Jamf Pro Mac App Store app.
func (r *MacAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MacAppResourceModel
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
		resp.Diagnostics.AddError("Missing ID", "Cannot delete Jamf Pro Mac App Store app without ID.")
		return
	}

	// Unlike /ebooks and /mobiledeviceapplications, the classic /macapplications
	// DELETE is clean and synchronous (wire-probed 2026-06-04: DELETE → 200 OK,
	// GET-by-id → 404 immediately), so a single delete with a 404-as-success
	// branch is sufficient. The SDK no longer treats DELETE→404 as success, so
	// the IsNotFoundError check is required here.
	if err := r.client.DeleteMacApplicationByID(deleteCtx, state.ID.ValueString()); err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro Mac App Store app already removed", map[string]any{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError("Error deleting Jamf Pro Mac App Store app", fmt.Sprintf("API error: %v", err))
	}
}
