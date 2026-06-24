// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// SDK endpoints used (Pro v1 /account-groups is read-only AND gateway-blocked,
// so every surface — resource, data source, list — goes through ProClassic):
//   proclassic.CreateAccountGroupByID   (POST id="0")
//   proclassic.GetAccountGroupByID      (read; data source id lookup; list expand)
//   proclassic.GetAccountGroupByName    (data source name lookup)
//   proclassic.UpdateAccountGroupByID   (PUT, 201 empty body)
//   proclassic.DeleteAccountGroupByID
//   proclassic.ListAccounts             (list-resource enumeration; privilege-catalog discovery in ModifyPlan)
//
// Status: current. Last reviewed 2026-06-24.

package account_group

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// Create creates a Jamf Pro account group. Classic POSTs to id="0"; the server
// allocates the integer ID. Privileges and members are trusted from the plan
// (not re-read from the server) so silent server expansion does not abort apply.
func (r *AccountGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AccountGroupResourceModel
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

	input, diags := buildAccountGroupInput(createCtx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateAccountGroupByID(createCtx, "0", input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Jamf Pro account group", err.Error())
		return
	}
	if created == nil || created.ID == nil {
		resp.Diagnostics.AddError(
			"Create response missing account group ID",
			"Jamf Pro returned success with no account group ID; cannot persist state.",
		)
		return
	}
	plan.ID = helpers.StringValueFromIntPtr(created.ID)

	got, err := r.client.GetAccountGroupByID(createCtx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading created Jamf Pro account group", err.Error())
		return
	}
	assignServerDerivedBaseFields(&plan, got)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, accountGroupIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Trace(ctx, "created Jamf Pro account group", map[string]any{"id": plan.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes Terraform state. Privileges and members are reconciled against
// what the configuration manages (intersect-on-read / null-aware).
func (r *AccountGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AccountGroupResourceModel
	isImport := req.State.Raw.IsNull()

	if isImport {
		if req.Identity == nil {
			resp.Diagnostics.AddError(
				"Missing resource identity",
				"Terraform requested a refresh for this account group without existing state or identity data.",
			)
			return
		}
		var identity accountGroupIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.ID.IsNull() || identity.ID.IsUnknown() || identity.ID.ValueString() == "" {
			resp.Diagnostics.AddError("Missing account group ID", "The resource identity did not include an 'id' attribute.")
			return
		}
		state.ID = identity.ID
		state.Timeouts = helpers.NewResourceTimeoutsNullValue(accountGroupTimeoutAttributeTypes)
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
		resp.Diagnostics.AddError("Missing ID", "Cannot read Jamf Pro account group without ID.")
		return
	}

	got, err := r.client.GetAccountGroupByID(readCtx, state.ID.ValueString())
	if err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro account group not found, removing from state", map[string]any{"id": state.ID.ValueString()})
			resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, accountGroupIdentityModel{ID: state.ID})...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Jamf Pro account group", err.Error())
		return
	}

	resp.Diagnostics.Append(assignAccountGroupResourceModel(readCtx, &state, got, isImport)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, accountGroupIdentityModel{ID: state.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates a Jamf Pro account group. UpdateAccountGroupByID returns 201
// with an empty body — GET to refresh server-derived base fields; privileges
// and members are trusted from the plan.
func (r *AccountGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AccountGroupResourceModel
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

	input, diags := buildAccountGroupInput(updateCtx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdateAccountGroupByID(updateCtx, plan.ID.ValueString(), input); err != nil {
		resp.Diagnostics.AddError("Error updating Jamf Pro account group", err.Error())
		return
	}

	got, err := r.client.GetAccountGroupByID(updateCtx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading updated Jamf Pro account group", err.Error())
		return
	}
	assignServerDerivedBaseFields(&plan, got)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, accountGroupIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes a Jamf Pro account group.
func (r *AccountGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AccountGroupResourceModel
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
		resp.Diagnostics.AddError("Missing ID", "Cannot delete Jamf Pro account group without ID.")
		return
	}

	if err := r.client.DeleteAccountGroupByID(deleteCtx, state.ID.ValueString()); err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro account group already removed", map[string]any{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError("Error deleting Jamf Pro account group", fmt.Sprintf("API error: %v", err))
	}
}
