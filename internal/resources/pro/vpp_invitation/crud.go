// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// SDK endpoints used:
//   proclassic.CreateVPPInvitationByID  (POST id/0 → 201, id-only body — GET after)
//   proclassic.GetVPPInvitationByID
//   proclassic.UpdateVPPInvitationByID  (PUT → 201, NO body — GET after)
//   proclassic.DeleteVPPInvitationByID  (→ 200)
//   proclassic.ListVPPInvitations       (data source / list resource)
//
// Writes are a MERGE (omit=retain). General scalars are emitted as planned; scope
// is always-emitted as a full skeleton so collections full-replace and clear.
//
// Status: current. Last reviewed 2026-06-08.

package vpp_invitation

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

func (r *VPPInvitationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan VPPInvitationResourceModel
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

	input, d := buildVPPInvitationInput(createCtx, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateVPPInvitationByID(createCtx, "0", input)
	if helpers.IsDirectoryGroupMatchConflict(err) {
		// Bootstrap apply: the referenced directory is still coming up. Retry until
		// the scope group resolves (or a real wrong-name conflict persists).
		err = helpers.RetryOnDirectoryGroupMatchConflict(createCtx, func() error {
			var e error
			created, e = r.client.CreateVPPInvitationByID(createCtx, "0", input)
			return e
		})
	}
	if err != nil {
		resp.Diagnostics.AddError("Error creating Jamf Pro VPP invitation", err.Error())
		return
	}
	if created == nil || created.ID == nil {
		resp.Diagnostics.AddError("Create response missing ID", "Jamf Pro returned 201 Created with no VPP invitation ID.")
		return
	}
	plan.ID = helpers.StringValueFromIntPtr(created.ID)

	got, err := r.client.GetVPPInvitationByID(createCtx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading created Jamf Pro VPP invitation", err.Error())
		return
	}
	assignVPPInvitationResourceModel(createCtx, &plan, got, false)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, vppInvitationIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Trace(ctx, "created Jamf Pro VPP invitation", map[string]any{"id": plan.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *VPPInvitationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state VPPInvitationResourceModel
	isImport := req.State.Raw.IsNull()

	if isImport {
		if req.Identity == nil {
			resp.Diagnostics.AddError("Missing resource identity", "Refresh requested without state or identity data.")
			return
		}
		var identity vppInvitationIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.ID.IsNull() || identity.ID.IsUnknown() || identity.ID.ValueString() == "" {
			resp.Diagnostics.AddError("Missing ID", "The resource identity did not include an 'id'.")
			return
		}
		state.ID = identity.ID
		state.Timeouts = helpers.NewResourceTimeoutsNullValue(vppInvitationTimeoutAttributeTypes)
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
		resp.Diagnostics.AddError("Missing ID", "Cannot read Jamf Pro VPP invitation without ID.")
		return
	}

	got, err := r.client.GetVPPInvitationByID(readCtx, state.ID.ValueString())
	if err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro VPP invitation not found, removing from state", map[string]any{"id": state.ID.ValueString()})
			resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, vppInvitationIdentityModel{ID: state.ID})...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Jamf Pro VPP invitation", err.Error())
		return
	}
	assignVPPInvitationResourceModel(readCtx, &state, got, false)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, vppInvitationIdentityModel{ID: state.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *VPPInvitationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan VPPInvitationResourceModel
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

	input, d := buildVPPInvitationInput(updateCtx, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	// PUT returns no body — must GET to refresh server-derived fields.
	if err := helpers.RetryOnDirectoryGroupMatchConflict(updateCtx, func() error {
		return r.client.UpdateVPPInvitationByID(updateCtx, plan.ID.ValueString(), input)
	}); err != nil {
		resp.Diagnostics.AddError("Error updating Jamf Pro VPP invitation", err.Error())
		return
	}

	got, err := r.client.GetVPPInvitationByID(updateCtx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading updated Jamf Pro VPP invitation", err.Error())
		return
	}
	assignVPPInvitationResourceModel(updateCtx, &plan, got, false)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, vppInvitationIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *VPPInvitationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state VPPInvitationResourceModel
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
		resp.Diagnostics.AddError("Missing ID", "Cannot delete Jamf Pro VPP invitation without ID.")
		return
	}

	if err := r.client.DeleteVPPInvitationByID(deleteCtx, state.ID.ValueString()); err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro VPP invitation already removed", map[string]any{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError("Error deleting Jamf Pro VPP invitation", fmt.Sprintf("API error: %v", err))
	}
}
