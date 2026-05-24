// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// SDK endpoints used:
//   proclassic.GetOSXConfigurationProfileByID
//   proclassic.GetOSXConfigurationProfileByName     (data source name lookup)
//   proclassic.CreateOSXConfigurationProfileByID
//   proclassic.UpdateOSXConfigurationProfileByID
//   proclassic.DeleteOSXConfigurationProfileByID
//   proclassic.ListOSXConfigurationProfiles         (list resource)
//
// Status: current. Last reviewed 2026-05-24.

package macos_configuration_profile

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// Create handles POST /JSSResource/osxconfigurationprofiles/id/0. Classic
// allocates the ID and returns it in the response body's <id>. The provider
// then runs a GET to capture the server-canonical form (including the
// server-rewritten payload + minted UUIDs) into state.
func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Provider not configured", providerNotConfigured)
		return
	}
	var plan ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, td := helpers.ResolveTimeout(ctx, plan.Timeouts.IsNull(), plan.Timeouts.IsUnknown(), defaultCreateTimeout, plan.Timeouts.Create)
	resp.Diagnostics.Append(td...)
	if resp.Diagnostics.HasError() {
		return
	}
	createCtx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	input, bd := buildInput(createCtx, plan, "")
	resp.Diagnostics.Append(bd...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateOSXConfigurationProfileByID(createCtx, "0", input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Jamf Pro macOS configuration profile", err.Error())
		return
	}

	id := ""
	if created.ID != nil {
		id = helpers.StringValueFromIntPtr(created.ID).ValueString()
	} else if created.General != nil && created.General.ID != nil {
		id = helpers.StringValueFromIntPtr(created.General.ID).ValueString()
	}
	if id == "" {
		resp.Diagnostics.AddError("Create response missing profile ID", "Jamf Pro did not return an ID for the created macOS configuration profile.")
		return
	}

	got, err := r.client.GetOSXConfigurationProfileByID(createCtx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading created macOS configuration profile", err.Error())
		return
	}
	resp.Diagnostics.Append(assignResourceModel(createCtx, &plan, got)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, identityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Trace(createCtx, "created jamfplatform_pro_macos_configuration_profile", map[string]any{"id": id})
	resp.Diagnostics.Append(resp.State.Set(createCtx, &plan)...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Provider not configured", providerNotConfigured)
		return
	}
	var state ResourceModel
	isImport := req.State.Raw.IsNull()
	if isImport {
		if req.Identity == nil {
			resp.Diagnostics.AddError(
				"Missing resource identity",
				"Terraform requested a refresh without existing state or identity data; provider cannot determine which profile to read.",
			)
			return
		}
		var identity identityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.ID.IsNull() || identity.ID.IsUnknown() || identity.ID.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Missing profile ID",
				"Resource identity did not include an 'id' attribute; provider cannot refresh the profile.",
			)
			return
		}
		state.ID = identity.ID
	} else {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	readTimeout, td := helpers.ResolveTimeout(ctx, state.Timeouts.IsNull(), state.Timeouts.IsUnknown(), defaultReadTimeout, state.Timeouts.Read)
	resp.Diagnostics.Append(td...)
	if resp.Diagnostics.HasError() {
		return
	}
	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	got, err := r.client.GetOSXConfigurationProfileByID(readCtx, state.ID.ValueString())
	if err != nil {
		if helpers.IsNotFoundError(err) {
			resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, identityModel{ID: state.ID})...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.State.RemoveResource(readCtx)
			return
		}
		resp.Diagnostics.AddError("Error reading macOS configuration profile", err.Error())
		return
	}
	resp.Diagnostics.Append(assignResourceModel(readCtx, &state, got)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, identityModel{ID: state.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(readCtx, &state)...)
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Provider not configured", providerNotConfigured)
		return
	}
	var plan ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, td := helpers.ResolveTimeout(ctx, plan.Timeouts.IsNull(), plan.Timeouts.IsUnknown(), defaultUpdateTimeout, plan.Timeouts.Update)
	resp.Diagnostics.Append(td...)
	if resp.Diagnostics.HasError() {
		return
	}
	updateCtx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	var existingUUID string
	if state.General != nil && !state.General.UUID.IsNull() && !state.General.UUID.IsUnknown() {
		existingUUID = state.General.UUID.ValueString()
	}
	input, bd := buildInput(updateCtx, plan, existingUUID)
	resp.Diagnostics.Append(bd...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if err := r.client.UpdateOSXConfigurationProfileByID(updateCtx, id, input); err != nil {
		resp.Diagnostics.AddError("Error updating macOS configuration profile", err.Error())
		return
	}

	got, err := r.client.GetOSXConfigurationProfileByID(updateCtx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading updated macOS configuration profile", err.Error())
		return
	}
	resp.Diagnostics.Append(assignResourceModel(updateCtx, &plan, got)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, identityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(updateCtx, &plan)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Provider not configured", providerNotConfigured)
		return
	}
	var state ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, td := helpers.ResolveTimeout(ctx, state.Timeouts.IsNull(), state.Timeouts.IsUnknown(), defaultDeleteTimeout, state.Timeouts.Delete)
	resp.Diagnostics.Append(td...)
	if resp.Diagnostics.HasError() {
		return
	}
	deleteCtx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	if err := r.client.DeleteOSXConfigurationProfileByID(deleteCtx, state.ID.ValueString()); err != nil {
		if helpers.IsNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting macOS configuration profile", err.Error())
		return
	}
}

const providerNotConfigured = "The provider client was not configured. This is a bug in the provider — please report it."
