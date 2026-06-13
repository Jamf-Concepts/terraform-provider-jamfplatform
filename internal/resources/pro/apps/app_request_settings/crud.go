// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// SDK endpoints used:
//   pro.GetAppRequestSettingsV1
//   pro.UpdateAppRequestSettingsV1   (PUT — full-replace, returns the echo)
//
// GET/PUT-only singleton: there is no Create or Delete endpoint (one record per tenant).
// Create adopts the existing settings — it reads the live settings as a merge base so any
// Optional+Computed field the user omits keeps its current value, then PUTs and seeds state
// from the returned echo. Delete is a no-op (state-only removal). The PUT is a full replace
// (wire-probed 2026-06-13: omitting appStoreLocale → HTTP 500; omitting approverEmails →
// cleared), so buildAppRequestSettingsInput always emits every field.
//
// Status: current. Last reviewed 2026-06-13.

package app_request_settings

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// providerNotConfiguredError is the diagnostic emitted when a CRUD handler is invoked
// before Configure has populated r.client.
func providerNotConfiguredError() (string, string) {
	return "Provider not configured",
		"The Jamf Pro client was not configured before the CRUD operation fired. Verify the provider block, credentials, and that Configure ran without errors."
}

// Create adopts and writes the App Request settings singleton. There is no Create endpoint,
// so this reads the live settings as a merge base, PUTs the plan, and seeds state from the
// returned echo.
func (r *AppRequestSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError(providerNotConfiguredError())
		return
	}

	var plan AppRequestSettingsResourceModel
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

	current, err := r.client.GetAppRequestSettingsV1(createCtx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading existing Jamf Pro App Request settings", err.Error())
		return
	}

	input, diags := buildAppRequestSettingsInput(ctx, plan, current)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateAppRequestSettingsV1(createCtx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error setting Jamf Pro App Request settings", err.Error())
		return
	}
	resp.Diagnostics.Append(assignAppRequestSettingsResourceModel(ctx, &plan, updated)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = types.StringValue(helpers.SingletonID)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, appRequestSettingsIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "applied Jamf Pro App Request settings")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes Terraform state with the latest settings from the Jamf Pro API.
func (r *AppRequestSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError(providerNotConfiguredError())
		return
	}

	var state AppRequestSettingsResourceModel
	isImport := req.State.Raw.IsNull()

	if isImport {
		state.ID = types.StringValue(helpers.SingletonID)
		state.Timeouts = helpers.NewResourceTimeoutsNullValue(appRequestSettingsTimeoutAttributeTypes)
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

	got, err := r.client.GetAppRequestSettingsV1(readCtx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Jamf Pro App Request settings", err.Error())
		return
	}

	resp.Diagnostics.Append(assignAppRequestSettingsResourceModel(ctx, &state, got)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.ID = types.StringValue(helpers.SingletonID)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, appRequestSettingsIdentityModel{ID: state.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update writes the new settings to the Jamf Pro API. The merge base is nil — UseStateForUnknown
// has already carried omitted Optional+Computed fields into the plan as known prior values.
func (r *AppRequestSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError(providerNotConfiguredError())
		return
	}

	var plan AppRequestSettingsResourceModel
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

	input, diags := buildAppRequestSettingsInput(ctx, plan, nil)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateAppRequestSettingsV1(updateCtx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Jamf Pro App Request settings", err.Error())
		return
	}
	resp.Diagnostics.Append(assignAppRequestSettingsResourceModel(ctx, &plan, updated)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = types.StringValue(helpers.SingletonID)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, appRequestSettingsIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete is a no-op on the Jamf Pro API. The settings singleton cannot be deleted — the
// record persists on the tenant. Terraform removes the resource from state on its own after
// this handler returns.
func (r *AppRequestSettingsResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	tflog.Trace(ctx, "removing Jamf Pro App Request settings from Terraform state (singleton — no remote delete)")
}
