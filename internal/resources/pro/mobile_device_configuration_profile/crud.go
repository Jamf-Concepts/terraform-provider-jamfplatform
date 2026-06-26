// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// SDK endpoints used:
//   proclassic.GetMobileDeviceConfigurationProfileByID
//   proclassic.GetMobileDeviceConfigurationProfileByName     (data source name lookup)
//   proclassic.CreateMobileDeviceConfigurationProfileByID
//   proclassic.UpdateMobileDeviceConfigurationProfileByID
//   proclassic.DeleteMobileDeviceConfigurationProfileByID
//   proclassic.ListMobileDeviceConfigurationProfiles         (list resource)
//
// Status: current. Last reviewed 2026-05-25.

package mobile_device_configuration_profile

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

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

	// Capture the user-authored payload before assignResourceModel
	// overwrites plan.General.Payloads with the server-canonical form.
	var userAuthoredPayload string
	if plan.General != nil && !plan.General.Payloads.IsNull() && !plan.General.Payloads.IsUnknown() {
		userAuthoredPayload = plan.General.Payloads.ValueString()
	}

	created, err := r.client.CreateMobileDeviceConfigurationProfileByID(createCtx, "0", input)
	if helpers.IsDirectoryGroupMatchConflict(err) {
		// Bootstrap apply: the referenced directory is still coming up. Retry until
		// the scope group resolves (or a real wrong-name conflict persists).
		err = helpers.RetryOnDirectoryGroupMatchConflict(createCtx, func() error {
			var e error
			created, e = r.client.CreateMobileDeviceConfigurationProfileByID(createCtx, "0", input)
			return e
		})
	}
	if err != nil {
		resp.Diagnostics.AddError("Error creating Jamf Pro mobile device configuration profile", err.Error())
		return
	}

	id := ""
	if created.ID != nil {
		id = helpers.StringValueFromIntPtr(created.ID).ValueString()
	} else if created.General != nil && created.General.ID != nil {
		id = helpers.StringValueFromIntPtr(created.General.ID).ValueString()
	}
	if id == "" {
		resp.Diagnostics.AddError("Create response missing profile ID", "Jamf Pro did not return an ID for the created mobile device configuration profile.")
		return
	}

	got, err := r.client.GetMobileDeviceConfigurationProfileByID(createCtx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading created mobile device configuration profile", err.Error())
		return
	}
	var rawServerPayload []byte
	if got != nil && got.General != nil && got.General.Payloads != nil {
		rawServerPayload = []byte(string(*got.General.Payloads))
	}
	resp.Diagnostics.Append(assignResourceModel(createCtx, &plan, got, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, identityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(writePrivatePayloadRefs(createCtx, resp.Private, userAuthoredPayload, rawServerPayload)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Trace(createCtx, "created jamfplatform_pro_mobile_device_configuration_profile", map[string]any{"id": id})
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

	got, err := r.client.GetMobileDeviceConfigurationProfileByID(readCtx, state.ID.ValueString())
	if err != nil {
		if helpers.IsNotFoundError(err) {
			resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, identityModel{ID: state.ID})...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.State.RemoveResource(readCtx)
			return
		}
		resp.Diagnostics.AddError("Error reading mobile device configuration profile", err.Error())
		return
	}
	var rawServerPayload []byte
	if got != nil && got.General != nil && got.General.Payloads != nil {
		rawServerPayload = []byte(string(*got.General.Payloads))
	}
	resp.Diagnostics.Append(assignResourceModel(readCtx, &state, got, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(reconcileReadDrift(readCtx, req.Private, &state, rawServerPayload)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(writePrivateServerNow(readCtx, resp.Private, rawServerPayload)...)
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

	// Capture the user-authored payload before assignResourceModel
	// overwrites plan.General.Payloads with the server-canonical form.
	var userAuthoredPayload string
	if plan.General != nil && !plan.General.Payloads.IsNull() && !plan.General.Payloads.IsUnknown() {
		userAuthoredPayload = plan.General.Payloads.ValueString()
	}

	id := state.ID.ValueString()
	if err := helpers.RetryOnDirectoryGroupMatchConflict(updateCtx, func() error {
		return r.client.UpdateMobileDeviceConfigurationProfileByID(updateCtx, id, input)
	}); err != nil {
		resp.Diagnostics.AddError("Error updating mobile device configuration profile", err.Error())
		return
	}

	got, err := r.client.GetMobileDeviceConfigurationProfileByID(updateCtx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading updated mobile device configuration profile", err.Error())
		return
	}
	var rawServerPayload []byte
	if got != nil && got.General != nil && got.General.Payloads != nil {
		rawServerPayload = []byte(string(*got.General.Payloads))
	}
	resp.Diagnostics.Append(assignResourceModel(updateCtx, &plan, got, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, identityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(writePrivatePayloadRefs(updateCtx, resp.Private, userAuthoredPayload, rawServerPayload)...)
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

	if err := r.client.DeleteMobileDeviceConfigurationProfileByID(deleteCtx, state.ID.ValueString()); err != nil {
		if helpers.IsNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting mobile device configuration profile", err.Error())
		return
	}
}

const providerNotConfigured = "The provider client was not configured. This is a bug in the provider — please report it."
