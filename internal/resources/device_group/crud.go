// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	deviceGroupCreateRetryDelay = 10 * time.Second
)

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

	reqBody := &jamfplatform.DeviceGroupCreateRepresentationV1{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		DeviceType:  strings.ToUpper(plan.DeviceType.ValueString()),
		GroupType:   strings.ToUpper(plan.GroupType.ValueString()),
	}

	switch strings.ToLower(plan.GroupType.ValueString()) {
	case "smart":
		reqBody.Criteria = expandDeviceGroupCriteria(plan.Criteria)
	case "static":
		if manageMembers {
			members, diags := helpers.SetToStringSlice(createCtx, plan.Members)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			reqBody.Members = members
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

	updateReq := &jamfplatform.DeviceGroupUpdateRepresentationV1{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	if strings.ToLower(plan.GroupType.ValueString()) == "smart" {
		updateReq.Criteria = expandDeviceGroupCriteria(plan.Criteria)
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
			patch := &jamfplatform.DeviceGroupMemberPatchRepresentationV1{
				Added:   added,
				Removed: removed,
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

	err := r.client.DeleteDeviceGroup(deleteCtx, state.ID.ValueString())
	if err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "device group already removed", map[string]any{
				"id": state.ID.ValueString(),
			})
			return
		}
		resp.Diagnostics.AddError("Error deleting device group", err.Error())
	}
}
