// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// SDK endpoints used:
//   proclassic.CreateVPPAssignmentByID  (POST id/0 → 201, id-only body — GET after)
//   proclassic.GetVPPAssignmentByID
//   proclassic.UpdateVPPAssignmentByID  (PUT → 201, NO body — GET after)
//   proclassic.DeleteVPPAssignmentByID  (→ 200)
//   proclassic.ListVPPAssignments       (data source / list resource)
//
// Writes are a MERGE (omit=retain). General scalars (name, vpp_admin_account_id)
// are emitted as planned. Content collections are opt-out (null omits, empty
// clears, populated replaces). Scope is always-emitted as a full skeleton so
// collections full-replace and clear.
//
// Status: current. Last reviewed 2026-06-08.

package vpp_assignment

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

func (r *VPPAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan VPPAssignmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, d := helpers.ResolveTimeout(ctx, plan.Timeouts.IsNull(), plan.Timeouts.IsUnknown(), defaultCreateTimeout, plan.Timeouts.Create)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	createCtx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	input, d := buildVPPAssignmentInput(createCtx, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateVPPAssignmentByID(createCtx, "0", input)
	if helpers.IsDirectoryGroupMatchConflict(err) {
		// Bootstrap apply: the referenced directory is still coming up. Retry until
		// the scope group resolves (or a real wrong-name conflict persists).
		err = helpers.RetryOnDirectoryGroupMatchConflict(createCtx, func() error {
			var e error
			created, e = r.client.CreateVPPAssignmentByID(createCtx, "0", input)
			return e
		})
	}
	if err != nil {
		resp.Diagnostics.AddError("Error creating Jamf Pro VPP assignment", err.Error())
		return
	}
	if created == nil || created.ID == nil {
		resp.Diagnostics.AddError("Create response missing ID", "Jamf Pro returned 201 Created with no VPP assignment ID.")
		return
	}
	plan.ID = helpers.StringValueFromIntPtr(created.ID)

	got, err := r.client.GetVPPAssignmentByID(createCtx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading created Jamf Pro VPP assignment", err.Error())
		return
	}
	assignVPPAssignmentResourceModel(createCtx, &plan, got, false)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, vppAssignmentIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Trace(ctx, "created Jamf Pro VPP assignment", map[string]any{"id": plan.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *VPPAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state VPPAssignmentResourceModel
	isImport := req.State.Raw.IsNull()

	if isImport {
		if req.Identity == nil {
			resp.Diagnostics.AddError("Missing resource identity", "Refresh requested without state or identity data.")
			return
		}
		var identity vppAssignmentIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.ID.IsNull() || identity.ID.IsUnknown() || identity.ID.ValueString() == "" {
			resp.Diagnostics.AddError("Missing ID", "The resource identity did not include an 'id'.")
			return
		}
		state.ID = identity.ID
		state.Timeouts = helpers.NewResourceTimeoutsNullValue(vppAssignmentTimeoutAttributeTypes)
		state.IosAppAdamIDs = types.SetNull(types.Int64Type)
		state.MacAppAdamIDs = types.SetNull(types.Int64Type)
		state.EbookAdamIDs = types.SetNull(types.Int64Type)
	} else {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	readTimeout, d := helpers.ResolveTimeout(ctx, state.Timeouts.IsNull(), state.Timeouts.IsUnknown(), defaultReadTimeout, state.Timeouts.Read)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	if state.ID.IsNull() || state.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot read Jamf Pro VPP assignment without ID.")
		return
	}

	got, err := r.client.GetVPPAssignmentByID(readCtx, state.ID.ValueString())
	if err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro VPP assignment not found, removing from state", map[string]any{"id": state.ID.ValueString()})
			resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, vppAssignmentIdentityModel{ID: state.ID})...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Jamf Pro VPP assignment", err.Error())
		return
	}
	assignVPPAssignmentResourceModel(readCtx, &state, got, false)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, vppAssignmentIdentityModel{ID: state.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *VPPAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan VPPAssignmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, d := helpers.ResolveTimeout(ctx, plan.Timeouts.IsNull(), plan.Timeouts.IsUnknown(), defaultUpdateTimeout, plan.Timeouts.Update)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	updateCtx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	input, d := buildVPPAssignmentInput(updateCtx, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	// PUT returns no body — must GET to refresh server-derived fields.
	if err := helpers.RetryOnDirectoryGroupMatchConflict(updateCtx, func() error {
		return r.client.UpdateVPPAssignmentByID(updateCtx, plan.ID.ValueString(), input)
	}); err != nil {
		resp.Diagnostics.AddError("Error updating Jamf Pro VPP assignment", err.Error())
		return
	}

	got, err := r.client.GetVPPAssignmentByID(updateCtx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading updated Jamf Pro VPP assignment", err.Error())
		return
	}
	assignVPPAssignmentResourceModel(updateCtx, &plan, got, false)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, vppAssignmentIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *VPPAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state VPPAssignmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, d := helpers.ResolveTimeout(ctx, state.Timeouts.IsNull(), state.Timeouts.IsUnknown(), defaultDeleteTimeout, state.Timeouts.Delete)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	deleteCtx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	if state.ID.IsNull() || state.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot delete Jamf Pro VPP assignment without ID.")
		return
	}

	if err := r.client.DeleteVPPAssignmentByID(deleteCtx, state.ID.ValueString()); err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro VPP assignment already removed", map[string]any{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError("Error deleting Jamf Pro VPP assignment", fmt.Sprintf("API error: %v", err))
	}
}
