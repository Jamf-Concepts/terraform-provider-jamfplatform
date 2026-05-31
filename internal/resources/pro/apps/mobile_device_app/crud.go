// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// SDK endpoints used:
//   proclassic.CreateMobileDeviceApplicationByID  (POST /mobiledeviceapplications/id/0)
//   proclassic.GetMobileDeviceApplicationByID     (GET  /mobiledeviceapplications/id/{id})
//   proclassic.UpdateMobileDeviceApplicationByID  (PUT  /mobiledeviceapplications/id/{id})
//   proclassic.DeleteMobileDeviceApplicationByID  (DELETE)
//   proclassic.ListMobileDeviceApplications       (data source / list resource)
//   proclassic.GetMobileDeviceApplicationByName   (data source name lookup)
//
// Status: current. Last reviewed 2026-05-31.
//
// Update semantics: the classic /mobiledeviceapplications PUT is a partial-merge,
// not a full-replace (wire-probed). The provider sends the full plan payload on
// every Update, including os_type (the server 409s on a PUT to an in-house app
// without it), so in-place edits to managed sections converge cleanly. Removing
// an entire optional block (scope / self_service / vpp / app_configuration) from
// config omits it from the payload, so the server retains the previously-stored
// block — a known limitation matching ProClassic precedent. To clear a block,
// null its individual fields rather than deleting the block.
//
// Delete semantics: the classic DELETE returns HTTP 400 but still deletes the
// app (maintainer-confirmed server bug). Delete therefore tolerates the error
// and verifies via a follow-up GET — a not-found GET means the delete succeeded.

package mobile_device_app

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// Create creates a new Jamf Pro mobile device app. Classic POSTs to id="0"; the
// server allocates the real integer ID and returns it in the response body.
func (r *MobileAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MobileAppResourceModel
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

	payload, buildDiags := buildMobileAppInput(createCtx, plan)
	resp.Diagnostics.Append(buildDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateMobileDeviceApplicationByID(createCtx, "0", payload)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Jamf Pro mobile device app", err.Error())
		return
	}
	id := extractMobileAppID(created)
	if id == "" {
		resp.Diagnostics.AddError(
			"Create response missing app ID",
			"Jamf Pro returned 201 Created with no app ID; cannot persist state.",
		)
		return
	}

	got, err := r.client.GetMobileDeviceApplicationByID(createCtx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading created Jamf Pro mobile device app", err.Error())
		return
	}
	resp.Diagnostics.Append(assignMobileAppResourceModel(createCtx, &plan, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, mobileAppIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "created Jamf Pro mobile device app", map[string]any{"id": plan.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes Terraform state with the latest app representation.
func (r *MobileAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MobileAppResourceModel
	isImport := req.State.Raw.IsNull()

	if isImport {
		if req.Identity == nil {
			resp.Diagnostics.AddError(
				"Missing resource identity",
				"Terraform requested a refresh for this app without existing state or identity data, so the provider cannot determine which app to read.",
			)
			return
		}
		var identity mobileAppIdentityModel
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
		state.Timeouts = helpers.NewResourceTimeoutsNullValue(mobileAppTimeoutAttributeTypes)
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
		resp.Diagnostics.AddError("Missing ID", "Cannot read Jamf Pro mobile device app without ID.")
		return
	}

	got, err := r.client.GetMobileDeviceApplicationByID(readCtx, state.ID.ValueString())
	if err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro mobile device app not found, removing from state", map[string]any{"id": state.ID.ValueString()})
			resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, mobileAppIdentityModel{ID: state.ID})...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Jamf Pro mobile device app", err.Error())
		return
	}

	resp.Diagnostics.Append(assignMobileAppResourceModel(readCtx, &state, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, mobileAppIdentityModel{ID: state.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates a Jamf Pro mobile device app. Classic
// UpdateMobileDeviceApplicationByID returns 201 with an empty body — we must GET
// to refresh state.
func (r *MobileAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan MobileAppResourceModel
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

	payload, buildDiags := buildMobileAppInput(updateCtx, plan)
	resp.Diagnostics.Append(buildDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdateMobileDeviceApplicationByID(updateCtx, plan.ID.ValueString(), payload); err != nil {
		resp.Diagnostics.AddError("Error updating Jamf Pro mobile device app", err.Error())
		return
	}

	got, err := r.client.GetMobileDeviceApplicationByID(updateCtx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading updated Jamf Pro mobile device app", err.Error())
		return
	}
	resp.Diagnostics.Append(assignMobileAppResourceModel(updateCtx, &plan, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, mobileAppIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes a Jamf Pro mobile device app. The classic DELETE returns HTTP
// 400 even when it removes the app (server bug, maintainer-confirmed), but the
// SDK absorbs this: its 4xx eventual-consistency retry re-issues the DELETE,
// hits the now-absent id, and treats the resulting 404 as an idempotent-delete
// success (returns nil). So the bug needs no provider-side handling — a clean
// nil and an already-removed (404) app both short-circuit to success, and a
// persistent error surfaces as a real failure.
func (r *MobileAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MobileAppResourceModel
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
		resp.Diagnostics.AddError("Missing ID", "Cannot delete Jamf Pro mobile device app without ID.")
		return
	}

	if err := r.client.DeleteMobileDeviceApplicationByID(deleteCtx, state.ID.ValueString()); err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro mobile device app already removed", map[string]any{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError("Error deleting Jamf Pro mobile device app", fmt.Sprintf("API error: %v", err))
	}
}
