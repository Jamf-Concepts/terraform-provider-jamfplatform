// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/devicegroups"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	deviceGroupCreateRetryDelay = 10 * time.Second
	deviceGroupDeleteRetryDelay = 10 * time.Second
)

// isDeletePropagationConflict reports whether a device-group DELETE error is a
// transient 406/409 raised while a dependent resource that referenced the group
// (e.g. a just-deleted blueprint) is still propagating server-side. The group
// still exists in this case, so re-issuing the DELETE once the reference clears
// is the correct action. (Distinct from the classic apps' accepted-async
// deletes, which return a misleading 400 and must NOT be re-issued.) The
// transport no longer retries 4xx, so this retry is handled here explicitly.
// The 406/409 set is from observed delete-after-blueprint behaviour; widen it if
// acceptance testing surfaces another transient status on this path.
func isDeletePropagationConflict(err error) bool {
	apiErr, ok := errors.AsType[*jamfplatform.APIResponseError](err)
	if !ok {
		return false
	}
	return apiErr.HasStatus(http.StatusNotAcceptable) || apiErr.HasStatus(http.StatusConflict)
}

// Create creates a new Jamf Platform device group resource.
func (r *DeviceGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DeviceGroupResourceModel
	var config DeviceGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
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

	if plan.Members.IsNull() && helpers.IsConfiguredValue(config.Members) {
		plan.Members = config.Members
	}

	manageMembers := helpers.IsConfiguredValue(plan.Members)
	manageDescription := helpers.IsConfiguredValue(plan.Description)

	if err := validateDeviceGroupPlan(&plan); err != nil {
		resp.Diagnostics.AddError("Invalid device group configuration", err.Error())
		return
	}

	reqBody := &devicegroups.DeviceGroupCreateRepresentationV1{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueStringPointer(),
		DeviceType:  strings.ToUpper(plan.DeviceType.ValueString()),
		GroupType:   strings.ToUpper(plan.GroupType.ValueString()),
	}

	switch strings.ToLower(plan.GroupType.ValueString()) {
	case "smart":
		criteria := expandDeviceGroupCriteria(plan.Criteria)
		reqBody.Criteria = &criteria
	case "static":
		if manageMembers {
			members, diags := helpers.SetToStringSlice(createCtx, plan.Members)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			reqBody.Members = &members
		}
	}

	created, err := r.client.CreateDeviceGroup(createCtx, reqBody)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating device group",
			fmt.Sprintf("API error: %v", err),
		)
		return
	}

	plan.ID = types.StringValue(created.ID)

	if !r.refreshDeviceGroupState(createCtx, created.ID, &plan, manageMembers, manageDescription, &resp.Diagnostics) {
		return
	}

	// resolveJamfProID degrades all Pro bridging failures to warnings so the
	// Platform Create result is never discarded. Append diagnostics without a
	// HasError gate — orphaning a successfully-created group would force a
	// manual `terraform import` to recover.
	jamfProID, jamfProDiags := resolveJamfProID(createCtx, r.proClient, r.pd, plan.ID.ValueString())
	resp.Diagnostics.Append(jamfProDiags...)
	plan.JamfProID = jamfProID

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, deviceGroupIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "created device group", map[string]any{
		"id":         created.ID,
		"group_type": plan.GroupType.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read syncs the Terraform state with the latest API representation.
func (r *DeviceGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DeviceGroupResourceModel
	isImport := req.State.Raw.IsNull()

	if isImport {
		if req.Identity == nil {
			resp.Diagnostics.AddError(
				"Missing resource identity",
				"Terraform requested a refresh for this device group without existing state or identity data, so the provider cannot determine which device group to read.",
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
		resp.Diagnostics.AddError("Missing ID", "Cannot read device group without ID.")
		return
	}

	grp, err := r.client.GetDeviceGroup(readCtx, state.ID.ValueString())
	if err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "device group not found, removing from state", map[string]any{
				"id": state.ID.ValueString(),
			})
			resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, deviceGroupIdentityModel{ID: state.ID})...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading device group", err.Error())
		return
	}

	var members []string
	manageMembers := isImport || helpers.IsConfiguredValue(state.Members)
	manageDescription := isImport || helpers.IsConfiguredValue(state.Description)
	if strings.EqualFold(grp.GroupType, "STATIC") && manageMembers {
		var err error
		members, err = r.client.ListDeviceGroupMembers(readCtx, grp.ID)
		if err != nil {
			resp.Diagnostics.AddError("Error reading device group members", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(assignDeviceGroupModel(readCtx, &state, grp, members, manageMembers, manageDescription)...)
	if resp.Diagnostics.HasError() {
		return
	}

	jamfProID, jamfProDiags := resolveJamfProID(readCtx, r.proClient, r.pd, state.ID.ValueString())
	resp.Diagnostics.Append(jamfProDiags...)
	state.JamfProID = jamfProID

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, deviceGroupIdentityModel{ID: state.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates name/description/criteria and reconciles membership for static groups.
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

	if err := validateDeviceGroupPlan(&plan); err != nil {
		resp.Diagnostics.AddError("Invalid device group configuration", err.Error())
		return
	}

	manageMembers := helpers.IsConfiguredValue(plan.Members)
	manageDescription := helpers.IsConfiguredValue(plan.Description)

	updateReq := &devicegroups.DeviceGroupUpdateRepresentationV1{
		Name:        plan.Name.ValueStringPointer(),
		Description: plan.Description.ValueStringPointer(),
	}

	if strings.ToLower(plan.GroupType.ValueString()) == "smart" {
		criteria := expandDeviceGroupCriteria(plan.Criteria)
		updateReq.Criteria = &criteria
	}

	if err := r.client.UpdateDeviceGroup(updateCtx, plan.ID.ValueString(), updateReq); err != nil {
		resp.Diagnostics.AddError(
			"Error updating device group",
			fmt.Sprintf("API error: %v", err),
		)
		return
	}

	if strings.ToLower(plan.GroupType.ValueString()) == "static" && manageMembers {
		desired, diags := helpers.SetToStringSlice(updateCtx, plan.Members)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		current, err := r.client.ListDeviceGroupMembers(updateCtx, plan.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading device group members", err.Error())
			return
		}

		added, removed := diffStringSlices(current, desired)
		if len(added) > 0 || len(removed) > 0 {
			patch := &devicegroups.DeviceGroupMemberPatchRepresentationV1{}
			if len(added) > 0 {
				patch.Added = &added
			}
			if len(removed) > 0 {
				patch.Removed = &removed
			}
			if err := r.client.UpdateDeviceGroupMembers(updateCtx, plan.ID.ValueString(), patch); err != nil {
				resp.Diagnostics.AddError("Error updating device group members", err.Error())
				return
			}
		}
	}

	if !r.refreshDeviceGroupState(updateCtx, plan.ID.ValueString(), &plan, manageMembers, manageDescription, &resp.Diagnostics) {
		return
	}

	jamfProID, jamfProDiags := resolveJamfProID(updateCtx, r.proClient, r.pd, plan.ID.ValueString())
	resp.Diagnostics.Append(jamfProDiags...)
	plan.JamfProID = jamfProID

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, deviceGroupIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes the device group from Jamf Platform.
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
		resp.Diagnostics.AddError("Missing ID", "Cannot delete device group without ID.")
		return
	}

	// Retry the DELETE on a transient 406/409 (a dependent resource that
	// referenced this group is still propagating its own deletion); a clean
	// success or a not-found means the group is gone. Any other error is
	// terminal. Bounded by the delete timeout.
	id := state.ID.ValueString()
	pollErr := jamfplatform.PollUntil(deleteCtx, deviceGroupDeleteRetryDelay, func(c context.Context) (bool, error) {
		err := r.client.DeleteDeviceGroup(c, id)
		switch {
		case err == nil:
			return true, nil
		case helpers.IsNotFoundError(err):
			tflog.Info(c, "device group already removed", map[string]any{"id": id})
			return true, nil
		case isDeletePropagationConflict(err):
			tflog.Info(c, "device group delete blocked by a still-propagating dependency; retrying", map[string]any{"id": id})
			return false, nil
		default:
			return false, err
		}
	})
	if pollErr != nil {
		resp.Diagnostics.AddError("Error deleting device group", pollErr.Error())
	}
}
