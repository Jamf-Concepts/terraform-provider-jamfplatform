// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// SDK endpoints used:
//   securitycloud.CreateDeviceGroupV1
//   securitycloud.GetDeviceGroupV1
//   securitycloud.UpdateDeviceGroupV1
//   securitycloud.DeleteDeviceGroupV1
//   securitycloud.ListDeviceGroupsV2 (data sources, list resource, and the
//                                     singular data source's name lookup)
//
// Deliberately not used:
//   securitycloud.ListDeviceGroupsV1   deprecated in the spec (deprecation-date
//                                      2026-08-12); content identical to V2.
//   securitycloud.ResolveDeviceGroupV2ByName
//                                      cannot express the built-in group. Where
//                                      the match is the implicit "Default Group",
//                                      which the list returns with no id key, the
//                                      resolver fails with "matched element has no
//                                      id field" and discards the matched element,
//                                      so the singular data source could never
//                                      reach its own id-less refusal. The name
//                                      lookup matches over ListDeviceGroupsV2
//                                      locally instead — see groupsNamedExactly.
//   securitycloud.UpdateDeviceGroupV2  the spec's nominated successor to the
//                                      deprecated V1 update, but the gateway does
//                                      not route PUT /v2/groups/{id} — 403, same
//                                      body as a deliberately bogus path, under
//                                      the privilege that makes V1 succeed.
//                                      Re-verified 2026-08-29. Raised upstream, so
//                                      Update still calls the deprecated V1 with a
//                                      staticcheck suppression.
//   securitycloud.ApplyDeviceGroupV*   generated name-keyed upsert; Terraform owns
//                                      the create-versus-update decision.
//
// Status: current. Last reviewed 2026-08-29.

package device_group

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// Create creates a new Jamf Security Cloud device group.
//
// The create response already carries the stored representation — id and name —
// but the group is read back anyway. The name is the one field the server may
// return differently from what was sent, and although the plan-time validator
// rejects the whitespace that would cause it, reading back keeps the resource
// honest if the server ever normalises something else.
//
// The empty-id guard before that read is not defensive noise: an empty id builds
// the collection path, and this API answers GET /v1/groups/ with the whole group
// list rather than a 404 (wire-probed 2026-08-29), so an unguarded read-back would
// surface as a decode failure rather than as the missing identifier it is.
//
// A read-back that fails after the create has succeeded still commits state. The group
// exists on the tenant by then, and returning without state would orphan it: group
// names are unique per tenant, so the retry is refused as a name already in use —
// naming the group the operator does not know they created. What is committed is the
// plan carrying the new ID, which the next refresh reconciles; an errored apply does
// not run Terraform's plan-consistency check. Nothing needs nulling first: the ID is
// this schema's only Computed value.
func (r *DeviceGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DeviceGroupResourceModel
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

	created, err := r.client.CreateDeviceGroupV1(createCtx, buildGroupCreateInput(plan))
	if err != nil {
		if !appendWriteDiagnostics(&resp.Diagnostics, err) {
			resp.Diagnostics.AddError("Error creating Jamf Security Cloud device group", err.Error())
		}
		return
	}

	if created.ID == "" {
		resp.Diagnostics.AddError(
			"Jamf Security Cloud returned no device group ID",
			"The create call succeeded but the response carried no identifier, so the group cannot be read back "+
				"or tracked in state. The group may still have been created — check the Jamf Security Cloud admin "+
				"UI before retrying, and remove it if it is there.",
		)
		return
	}

	plan.ID = types.StringValue(created.ID)

	got, err := r.client.GetDeviceGroupV1(createCtx, created.ID)
	if err != nil {
		resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, deviceGroupIdentityModel{ID: plan.ID})...)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError(
			"Error reading created Jamf Security Cloud device group",
			"The group was created with ID \""+created.ID+"\" but could not be read back, so Terraform has "+
				"recorded its ID and the configured name without confirming what was stored. The next plan will "+
				"refresh it — do not re-create it: group names are unique per tenant, so a second create would be "+
				"refused as a name already in use. Underlying error: "+err.Error(),
		)
		return
	}
	assignDeviceGroupResourceModel(&plan, got)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, deviceGroupIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "created Jamf Security Cloud device group", map[string]any{"id": created.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest device group representation.
func (r *DeviceGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DeviceGroupResourceModel
	isImport := req.State.Raw.IsNull()

	if isImport {
		if req.Identity == nil {
			resp.Diagnostics.AddError(
				"Missing resource identity",
				"Terraform requested a refresh for this device group without existing state or identity data, so the provider cannot determine which group to read.",
			)
			return
		}
		var identity deviceGroupIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.ID.IsNull() || identity.ID.IsUnknown() || identity.ID.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Missing device group ID",
				"The resource identity did not include an 'id' attribute, so the provider cannot refresh the device group.",
			)
			return
		}
		state.ID = identity.ID
		state.Timeouts = helpers.NewResourceTimeoutsNullValue(deviceGroupTimeoutAttributeTypes)
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
		resp.Diagnostics.AddError("Missing ID", "Cannot read Jamf Security Cloud device group without ID.")
		return
	}

	got, err := r.client.GetDeviceGroupV1(readCtx, state.ID.ValueString())
	if err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Security Cloud device group not found, removing from state", map[string]any{"id": state.ID.ValueString()})
			resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, deviceGroupIdentityModel{ID: state.ID})...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Jamf Security Cloud device group", err.Error())
		return
	}

	assignDeviceGroupResourceModel(&state, got)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, deviceGroupIdentityModel{ID: state.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update renames a Jamf Security Cloud device group.
//
// A rename is the only update this resource can make. The PUT echoes the stored
// object, but state is taken from a fresh read for the same reason Create does it.
//
// UpdateDeviceGroupV1 is called despite being marked deprecated in the spec
// (deprecation-date 2026-08-25), because its nominated successor does not exist on
// the wire. STYLE_GUIDE §Deprecated with no generated successor says to verify the
// successor before concluding anything, so it was verified twice against the EU
// sandbox on 2026-08-29: PUT /securitycloud/v2/groups/{id} answers 403
// BAD_PERMISSIONS through both curl and the SDK, using the same token and the same
// device-groups:update privilege that makes the v1 PUT return 200. A control probe
// on a deliberately bogus path returns the identical body, which is what
// identifies it as an unrouted endpoint rather than a privilege gap; PATCH is not
// served at either version either. Raised upstream.
//
// Revisit when the v2 route starts answering, but note the swap is not call-for-call:
// the request body is identical (*UpdateGroupRequest both sides), yet v1 answers 200
// with the stored object and v2 answers 204 with none, so the SDK generates
// `UpdateDeviceGroupV2(ctx, id, req) error` against v1's `(*Group, error)`. The call
// site below discards the object already, so the change is to
// `if err := r.client.UpdateDeviceGroupV2(...)` plus dropping this suppression — the
// arity differs and the current line will not compile as-is.
func (r *DeviceGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DeviceGroupResourceModel
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

	if plan.ID.IsNull() || plan.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot update Jamf Security Cloud device group without ID.")
		return
	}

	//nolint:staticcheck // SA1019: the v2 successor is unrouted — see the doc comment above.
	if _, err := r.client.UpdateDeviceGroupV1(updateCtx, plan.ID.ValueString(), buildGroupUpdateInput(plan)); err != nil {
		if !appendWriteDiagnostics(&resp.Diagnostics, err) {
			resp.Diagnostics.AddError("Error updating Jamf Security Cloud device group", err.Error())
		}
		return
	}

	got, err := r.client.GetDeviceGroupV1(updateCtx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading updated Jamf Security Cloud device group", err.Error())
		return
	}
	assignDeviceGroupResourceModel(&plan, got)

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, deviceGroupIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes a Jamf Security Cloud device group.
//
// The endpoint is idempotent: it answers 204 for a group that was already gone
// and for an identifier that never existed at all, so the not-found branch every
// other resource here needs is unreachable. It is kept anyway, because an
// endpoint that starts returning 404 should not turn a converged destroy into an
// error.
//
// The delete is not refused when something still references the group. See the
// package doc comment — the referrer is silently left assigned to nothing, which
// is why the resource description warns about it rather than the diagnostics.
func (r *DeviceGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DeviceGroupResourceModel
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
		resp.Diagnostics.AddError("Missing ID", "Cannot delete Jamf Security Cloud device group without ID.")
		return
	}

	if err := r.client.DeleteDeviceGroupV1(deleteCtx, state.ID.ValueString()); err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Security Cloud device group already removed", map[string]any{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError("Error deleting Jamf Security Cloud device group", fmt.Sprintf("API error: %v", err))
	}
}
