// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// SDK endpoints used:
//   proclassic.CreateRestrictedSoftwareByID   (POST   /restrictedsoftware/id/0)
//   proclassic.GetRestrictedSoftwareByID      (GET    /restrictedsoftware/id/{id})
//   proclassic.UpdateRestrictedSoftwareByID   (PUT    /restrictedsoftware/id/{id})
//   proclassic.DeleteRestrictedSoftwareByID   (DELETE /restrictedsoftware/id/{id})
//   proclassic.ListRestrictedSoftware         (data source / list resource)
//   proclassic.GetRestrictedSoftwareByName    (data source name lookup)
//
// Status: current. Last reviewed 2026-05-31.
//
// Server invariants (wire-probed):
//   - Create POSTs to id="0"; the server allocates the integer ID and returns
//     it at the top level (<restricted_software><id>).
//   - match_exact_process_name server-defaults to true; the other general bools
//     and display_message default false/empty.
//   - PUT returns 201 with an empty body — Update must GET to refresh state.
//   - A scope reference to a nonexistent object (computer group, building, …)
//     returns HTTP 409 Conflict (distinct from the literal-'&' content 409).
//   - DELETE returns 200; a repeat DELETE / GET of a removed record returns 404,
//     so Read and Delete self-heal via helpers.IsNotFoundError.
//
// Update semantics: like all classic endpoints the PUT is a partial-merge at
// top-section granularity. The provider always sends the full plan payload, so
// in-place edits to general/scope converge. Removing the entire optional scope
// block from config omits it from the payload, so the server retains the
// previously-stored scope — a known ProClassic limitation; null the individual
// scope fields rather than deleting the block to clear it.

package restricted_software

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// Create creates a new Jamf Pro restricted software record. Classic POSTs to
// id="0"; the server allocates the real integer ID and returns it.
func (r *RestrictedSoftwareResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RestrictedSoftwareResourceModel
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

	payload, buildDiags := buildRestrictedSoftwareInput(createCtx, plan)
	resp.Diagnostics.Append(buildDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateRestrictedSoftwareByID(createCtx, "0", payload)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Jamf Pro restricted software", err.Error())
		return
	}
	id := extractRestrictedSoftwareID(created)
	if id == "" {
		resp.Diagnostics.AddError(
			"Create response missing record ID",
			"Jamf Pro returned 201 Created with no restricted software ID; cannot persist state.",
		)
		return
	}

	got, err := r.client.GetRestrictedSoftwareByID(createCtx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading created Jamf Pro restricted software", err.Error())
		return
	}
	resp.Diagnostics.Append(assignRestrictedSoftwareResourceModel(createCtx, &plan, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, restrictedSoftwareIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "created Jamf Pro restricted software", map[string]any{"id": plan.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes Terraform state with the latest record representation.
func (r *RestrictedSoftwareResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RestrictedSoftwareResourceModel
	isImport := req.State.Raw.IsNull()

	if isImport {
		if req.Identity == nil {
			resp.Diagnostics.AddError(
				"Missing resource identity",
				"Terraform requested a refresh for this restricted software record without existing state or identity data, so the provider cannot determine which record to read.",
			)
			return
		}
		var identity restrictedSoftwareIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.ID.IsNull() || identity.ID.IsUnknown() || identity.ID.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Missing record ID",
				"The resource identity did not include an 'id' attribute, so the provider cannot refresh the restricted software record.",
			)
			return
		}
		state.ID = identity.ID
		state.Timeouts = helpers.NewResourceTimeoutsNullValue(restrictedSoftwareTimeoutAttributeTypes)
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
		resp.Diagnostics.AddError("Missing ID", "Cannot read Jamf Pro restricted software without ID.")
		return
	}

	got, err := r.client.GetRestrictedSoftwareByID(readCtx, state.ID.ValueString())
	if err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro restricted software not found, removing from state", map[string]any{"id": state.ID.ValueString()})
			resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, restrictedSoftwareIdentityModel{ID: state.ID})...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Jamf Pro restricted software", err.Error())
		return
	}

	resp.Diagnostics.Append(assignRestrictedSoftwareResourceModel(readCtx, &state, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, restrictedSoftwareIdentityModel{ID: state.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates a Jamf Pro restricted software record. Classic
// UpdateRestrictedSoftwareByID returns 201 with an empty body — we must GET to
// refresh state.
func (r *RestrictedSoftwareResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RestrictedSoftwareResourceModel
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

	payload, buildDiags := buildRestrictedSoftwareInput(updateCtx, plan)
	resp.Diagnostics.Append(buildDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdateRestrictedSoftwareByID(updateCtx, plan.ID.ValueString(), payload); err != nil {
		resp.Diagnostics.AddError("Error updating Jamf Pro restricted software", err.Error())
		return
	}

	got, err := r.client.GetRestrictedSoftwareByID(updateCtx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading updated Jamf Pro restricted software", err.Error())
		return
	}
	resp.Diagnostics.Append(assignRestrictedSoftwareResourceModel(updateCtx, &plan, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, restrictedSoftwareIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes a Jamf Pro restricted software record.
func (r *RestrictedSoftwareResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RestrictedSoftwareResourceModel
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
		resp.Diagnostics.AddError("Missing ID", "Cannot delete Jamf Pro restricted software without ID.")
		return
	}

	if err := r.client.DeleteRestrictedSoftwareByID(deleteCtx, state.ID.ValueString()); err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro restricted software already removed", map[string]any{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError("Error deleting Jamf Pro restricted software", fmt.Sprintf("API error: %v", err))
	}
}
