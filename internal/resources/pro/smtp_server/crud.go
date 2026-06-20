// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// SDK endpoints used:
//   pro.GetSmtpServerV2
//   pro.UpdateSmtpServerV2
//
// Status: current. Last reviewed 2026-06-09.

package smtp_server

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// Create handles initial provisioning of the SMTP Server settings singleton. The
// Jamf Pro API has no Create endpoint — one record per tenant already exists — so
// this reads the live settings (merge base for adopting undeclared fields like
// `enabled`), funnels into Update against the plan, then reads back authoritative
// state.
func (r *SmtpServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError(helpers.ProviderNotConfiguredError())
		return
	}

	var plan, cfg SmtpServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
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

	current, err := r.client.GetSmtpServerV2(createCtx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading existing Jamf Pro SMTP Server settings", err.Error())
		return
	}

	if _, err := r.client.UpdateSmtpServerV2(createCtx, buildSmtpServerInput(plan, current, createSecret(cfg))); err != nil {
		resp.Diagnostics.AddError("Error setting Jamf Pro SMTP Server settings", err.Error())
		return
	}

	got, err := r.client.GetSmtpServerV2(createCtx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Jamf Pro SMTP Server settings after write", err.Error())
		return
	}
	resp.Diagnostics.Append(assignSmtpServerResourceModel(&plan, got, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = types.StringValue(helpers.SingletonID)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, smtpServerIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "applied Jamf Pro SMTP Server settings")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes Terraform state with the latest settings from the Jamf Pro API.
func (r *SmtpServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError(helpers.ProviderNotConfiguredError())
		return
	}

	var state SmtpServerResourceModel
	isImport := req.State.Raw.IsNull()

	if isImport {
		state.ID = types.StringValue(helpers.SingletonID)
		state.Timeouts = helpers.NewResourceTimeoutsNullValue(smtpServerTimeoutAttributeTypes)
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

	got, err := r.client.GetSmtpServerV2(readCtx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Jamf Pro SMTP Server settings", err.Error())
		return
	}

	resp.Diagnostics.Append(assignSmtpServerResourceModel(&state, got, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.ID = types.StringValue(helpers.SingletonID)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, smtpServerIdentityModel{ID: state.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update writes the new settings to the Jamf Pro API. Full-replace PUT; the
// WriteOnly secret for the active mode is sent only when its rotation trigger
// changed.
func (r *SmtpServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError(helpers.ProviderNotConfiguredError())
		return
	}

	var plan, state, cfg SmtpServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
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

	if _, err := r.client.UpdateSmtpServerV2(updateCtx, buildSmtpServerInput(plan, nil, updateSecret(cfg, plan, state))); err != nil {
		resp.Diagnostics.AddError("Error updating Jamf Pro SMTP Server settings", err.Error())
		return
	}

	got, err := r.client.GetSmtpServerV2(updateCtx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Jamf Pro SMTP Server settings after update", err.Error())
		return
	}
	resp.Diagnostics.Append(assignSmtpServerResourceModel(&plan, got, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = types.StringValue(helpers.SingletonID)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, smtpServerIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete is a no-op on the Jamf Pro API. Singleton objects cannot be deleted —
// the record persists on the tenant. Terraform removes the resource from state
// on its own after this handler returns.
func (r *SmtpServerResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	tflog.Trace(ctx, "removing Jamf Pro SMTP Server settings from Terraform state (singleton — no remote delete)")
}
