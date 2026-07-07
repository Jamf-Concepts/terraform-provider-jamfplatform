// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// SDK endpoints used:
//   proclassic.CreatePatchPolicyBySoftwareTitleConfigID (POST /patchpolicies/softwaretitleconfig/id/{configID})
//   proclassic.GetPatchPolicyByID                       (GET    /patchpolicies/id/{id})
//   proclassic.UpdatePatchPolicyByID                    (PUT    /patchpolicies/id/{id}, 201 empty body → GET after)
//   proclassic.DeletePatchPolicyByID                    (DELETE /patchpolicies/id/{id})
//   proclassic.ListPatchPolicies                        (list resource; spec-deprecated — list-only)
//
// DEPRECATION: only the /patchpolicies *list* endpoints carry the SDK
// `// Deprecated:` marker (the spec points at /v2/patch-policies for listing).
// The by-ID CRUD funcs (Create-by-config-path, Get/Update/Delete by ID) are NOT
// deprecated and are the current, functional surface for managing a policy.
//
// Status: current. Last reviewed 2026-06-01.
//
// Server invariants (wire-probed):
//   - Create POSTs the policy <general> to the software-title-config path; the
//     server allocates the integer policy ID and returns it at the top level.
//     It DERIVES release_date, incremental_update, reboot, minimum_os, and
//     kill_apps from the target_version's patch definition (deliberately-wrong
//     submitted values are overwritten) — these are modelled Computed-only.
//   - The server fills full user_interaction defaults on create (install button
//     text "Update", deadline 7d, grace 15m, reminders 24h, …).
//   - distribution_method coerces invalid values to "prompt" server-side; the
//     provider validates to {selfservice, prompt} to surface the choice.
//   - target_version must be a version that has a package assigned on the title.
//   - Scope round-trips by id (entity scope only); the empty <users/> element the
//     wire emits is vestigial and is not modelled (write-only-unreadable).
//   - Scope category clearing (wire-probed 2026-06-01): when <scope> is sent, the
//     server CLEARS every category omitted from it (buildings/departments/
//     computers/limitations/exclusions all came back empty after a PUT that sent
//     only <computer_groups>). So the builder's omission semantics (a dropped
//     category → omitted from PUT → cleared) converge correctly; no always-emit
//     empty-element sentinel is needed. NB this is the OPPOSITE of the classic
//     /restrictedsoftware retain-on-omit behaviour — probed per-endpoint.
//   - GET does NOT echo <software_title_configuration_id> (wire-probed): it is a
//     create-time-only path parameter. Read therefore preserves the configured
//     value (preferCurrent*); on import it cannot be reconstructed, so it lands
//     in the acceptance ImportStateVerifyIgnore list.
//   - PUT returns 201 with an empty body — Update must GET to refresh state.
//   - DELETE returns 200; a repeat DELETE / GET of a removed policy returns 404,
//     so Read and Delete self-heal via helpers.IsNotFoundError.
//
// Update semantics: like all classic endpoints the PUT is a partial-merge at
// top-section granularity. The provider always sends the full writable plan
// payload, so in-place edits to general/scope/user_interaction converge.
// Removing the entire optional scope / user_interaction block from config omits
// it from the payload, so the server retains the previously-stored values — a
// known ProClassic limitation; null the individual fields rather than deleting
// the block to clear them.

package patch_policy

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// Create creates a new Jamf Pro patch policy against the supplied patch software
// title configuration. The server allocates the integer policy ID and returns
// it; we then GET by that ID to capture the server-derived general fields.
func (r *PatchPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PatchPolicyResourceModel
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

	payload, buildDiags := buildPatchPolicyInput(createCtx, plan)
	resp.Diagnostics.Append(buildDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	configID := plan.SoftwareTitleConfigurationID.ValueString()
	created, err := r.client.CreatePatchPolicyBySoftwareTitleConfigID(createCtx, configID, payload)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Jamf Pro patch policy", err.Error())
		return
	}
	id := extractPatchPolicyID(created)
	if id == "" {
		resp.Diagnostics.AddError(
			"Create response missing patch policy ID",
			"Jamf Pro returned 201 Created with no patch policy ID; cannot persist state.",
		)
		return
	}

	got, err := r.client.GetPatchPolicyByID(createCtx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading created Jamf Pro patch policy", err.Error())
		return
	}
	resp.Diagnostics.Append(assignPatchPolicyResourceModel(createCtx, &plan, got, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, patchPolicyIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "created Jamf Pro patch policy", map[string]any{"id": plan.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes Terraform state with the latest patch policy representation.
func (r *PatchPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PatchPolicyResourceModel
	isImport := req.State.Raw.IsNull()

	if isImport {
		if req.Identity == nil {
			resp.Diagnostics.AddError(
				"Missing resource identity",
				"Terraform requested a refresh for this patch policy without existing state or identity data, so the provider cannot determine which policy to read.",
			)
			return
		}
		var identity patchPolicyIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.ID.IsNull() || identity.ID.IsUnknown() || identity.ID.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Missing patch policy ID",
				"The resource identity did not include an 'id' attribute, so the provider cannot refresh the patch policy.",
			)
			return
		}
		state.ID = identity.ID
		state.Timeouts = helpers.NewResourceTimeoutsNullValue(patchPolicyTimeoutAttributeTypes)
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
		resp.Diagnostics.AddError("Missing ID", "Cannot read Jamf Pro patch policy without ID.")
		return
	}

	got, err := r.client.GetPatchPolicyByID(readCtx, state.ID.ValueString())
	if err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro patch policy not found, removing from state", map[string]any{"id": state.ID.ValueString()})
			resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, patchPolicyIdentityModel{ID: state.ID})...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Jamf Pro patch policy", err.Error())
		return
	}

	// firstHydration detects an unpopulated incoming model (see mac_app_store_app
	// / policy for the full rationale): name is schema-Required (no nested
	// general block on this resource) and always populated in genuinely
	// managed state, so state.Name.IsNull() can only mean this Read call is
	// doing first-time import hydration. Hydrate the wire-present optional
	// scope/user_interaction sections in that case; subsequent Reads revert to
	// only refreshing sections the current state already tracks.
	firstHydration := state.Name.IsNull()
	resp.Diagnostics.Append(assignPatchPolicyResourceModel(readCtx, &state, got, firstHydration)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, patchPolicyIdentityModel{ID: state.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates a Jamf Pro patch policy. Classic UpdatePatchPolicyByID returns
// 201 with an empty body — we must GET to refresh state.
func (r *PatchPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PatchPolicyResourceModel
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

	payload, buildDiags := buildPatchPolicyInput(updateCtx, plan)
	resp.Diagnostics.Append(buildDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdatePatchPolicyByID(updateCtx, plan.ID.ValueString(), payload); err != nil {
		resp.Diagnostics.AddError("Error updating Jamf Pro patch policy", err.Error())
		return
	}

	got, err := r.client.GetPatchPolicyByID(updateCtx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading updated Jamf Pro patch policy", err.Error())
		return
	}
	resp.Diagnostics.Append(assignPatchPolicyResourceModel(updateCtx, &plan, got, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, patchPolicyIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes a Jamf Pro patch policy.
func (r *PatchPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PatchPolicyResourceModel
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
		resp.Diagnostics.AddError("Missing ID", "Cannot delete Jamf Pro patch policy without ID.")
		return
	}

	if err := r.client.DeletePatchPolicyByID(deleteCtx, state.ID.ValueString()); err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro patch policy already removed", map[string]any{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError("Error deleting Jamf Pro patch policy", fmt.Sprintf("API error: %v", err))
	}
}
