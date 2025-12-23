// Copyright 2025 Jamf Software LLC.

package blueprint

import (
	"context"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Create creates a new Blueprint resource in Terraform.
func (r *BlueprintResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BlueprintResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout := defaultCreateTimeout
	if !data.Timeouts.IsNull() && !data.Timeouts.IsUnknown() {
		configuredTimeout, timeoutDiags := data.Timeouts.Create(ctx, defaultCreateTimeout)
		resp.Diagnostics.Append(timeoutDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		createTimeout = configuredTimeout
	}

	createCtx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	var deviceGroupsSet []string
	diags := data.DeviceGroups.ElementsAs(ctx, &deviceGroupsSet, false)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	allComponents, diags := r.collectAllComponents(ctx, &data)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	steps := []client.BlueprintStepV1{
		{
			Name:       "Declaration group",
			Components: allComponents,
		},
	}

	reqBody := &client.BlueprintCreateRequestV1{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Scope: client.BlueprintCreateScopeV1{
			DeviceGroups: deviceGroupsSet,
		},
		Steps: steps,
	}

	createResp, err := r.client.CreateBlueprintV1(createCtx, reqBody)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating blueprint",
			"Could not create blueprint: "+err.Error(),
		)
		return
	}

	err = r.client.DeployBlueprintV1(createCtx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Blueprint deployment failed",
			"Blueprint was created successfully but may not have been deployed: "+err.Error()+
				". The blueprint may have been deployed despite the error. Check your Jamf instance to verify the blueprint status.",
		)
	}

	blueprint, err := r.client.GetBlueprintByIDV1(createCtx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading created blueprint",
			"Could not read created blueprint: "+err.Error(),
		)
		return
	}

	updateModelFromAPIResponse(&data, blueprint)

	tflog.Trace(ctx, "created a resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Read reads the Blueprint resource state from the API.
func (r *BlueprintResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BlueprintResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout := defaultReadTimeout
	if !data.Timeouts.IsNull() && !data.Timeouts.IsUnknown() {
		configuredTimeout, timeoutDiags := data.Timeouts.Read(ctx, defaultReadTimeout)
		resp.Diagnostics.Append(timeoutDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		readTimeout = configuredTimeout
	}

	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	blueprint, err := r.client.GetBlueprintByIDV1(readCtx, data.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			tflog.Info(ctx, "Blueprint not found, removing from state", map[string]interface{}{
				"blueprint_id": data.ID.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading blueprint",
			"Could not read blueprint: "+err.Error(),
		)
		return
	}

	updateModelFromAPIResponse(&data, blueprint)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Update updates the Blueprint resource.
func (r *BlueprintResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BlueprintResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout := defaultUpdateTimeout
	if !data.Timeouts.IsNull() && !data.Timeouts.IsUnknown() {
		configuredTimeout, timeoutDiags := data.Timeouts.Update(ctx, defaultUpdateTimeout)
		resp.Diagnostics.Append(timeoutDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateTimeout = configuredTimeout
	}

	updateCtx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	var deviceGroupsSet []string
	diags := data.DeviceGroups.ElementsAs(ctx, &deviceGroupsSet, false)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	allComponents, diags := r.collectAllComponents(ctx, &data)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	steps := []client.BlueprintStepV1{
		{
			Name:       "Declaration group",
			Components: allComponents,
		},
	}

	updateReq := &client.BlueprintUpdateRequestV1{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Scope: client.BlueprintUpdateScopeV1{
			DeviceGroups: deviceGroupsSet,
		},
		Steps: steps,
	}

	err := r.client.UpdateBlueprintV1(updateCtx, data.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating blueprint",
			"Could not update blueprint: "+err.Error(),
		)
		return
	}

	err = r.client.DeployBlueprintV1(updateCtx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Blueprint deployment failed",
			"Blueprint was updated successfully but may not have been deployed: "+err.Error()+
				". The blueprint may have been deployed despite the error. Check your Jamf instance to verify the blueprint status.",
		)
	}

	blueprint, err := r.client.GetBlueprintByIDV1(updateCtx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading updated blueprint",
			"Could not read updated blueprint: "+err.Error(),
		)
		return
	}

	updateModelFromAPIResponse(&data, blueprint)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Delete deletes the Blueprint resource.
func (r *BlueprintResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BlueprintResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout := defaultDeleteTimeout
	if !data.Timeouts.IsNull() && !data.Timeouts.IsUnknown() {
		configuredTimeout, timeoutDiags := data.Timeouts.Delete(ctx, defaultDeleteTimeout)
		resp.Diagnostics.Append(timeoutDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		deleteTimeout = configuredTimeout
	}

	deleteCtx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	if data.ID.IsNull() || data.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot delete blueprint without ID.")
		return
	}

	err := r.client.DeleteBlueprintV1(deleteCtx, data.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			tflog.Info(ctx, "Blueprint already deleted", map[string]interface{}{
				"blueprint_id": data.ID.ValueString(),
			})
			return
		}

		if isServerError(err) {
			resp.Diagnostics.AddWarning(
				"Blueprint deletion encountered server error",
				"Delete operation encountered a server error: "+err.Error()+
					". The blueprint may have been deleted despite the error. Check your Jamf instance to verify the blueprint status.",
			)
			return
		}

		resp.Diagnostics.AddError(
			"Error deleting blueprint",
			"Could not delete blueprint: "+err.Error(),
		)
		return
	}
}
