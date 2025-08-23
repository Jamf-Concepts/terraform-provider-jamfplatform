// Copyright 2025 Jamf Software LLC.

package blueprint

import (
	"context"
	"encoding/json"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Create creates a new Blueprint resource in Terraform.
func (r *BlueprintResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan blueprintResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deviceGroups := make([]string, len(plan.DeviceGroups))
	for i, dg := range plan.DeviceGroups {
		deviceGroups[i] = dg.ValueString()
	}

	components := make([]client.BlueprintComponent, len(plan.Components))
	for i, comp := range plan.Components {
		component := client.BlueprintComponent{
			Identifier: comp.Identifier.ValueString(),
		}

		if !comp.Configuration.IsNull() && !comp.Configuration.IsUnknown() {
			configMap := make(map[string]string)
			diags := comp.Configuration.ElementsAs(ctx, &configMap, false)
			if diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}

			jsonObj := make(map[string]interface{})
			for key, value := range configMap {
				setNestedValue(jsonObj, key, value)
			}

			jsonBytes, err := json.Marshal(jsonObj)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error encoding component configuration",
					"Could not encode component configuration to JSON: "+err.Error(),
				)
				return
			}

			normalizedConfig := normalizeJSON(string(jsonBytes))
			component.Configuration = json.RawMessage(normalizedConfig)
		}
		components[i] = component
	}

	steps := []client.BlueprintStep{
		{
			Name:       "Declaration group",
			Components: components,
		},
	}

	reqBody := &client.BlueprintCreateRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Scope: client.BlueprintCreateScope{
			DeviceGroups: deviceGroups,
		},
		Steps: steps,
	}

	createResp, err := r.client.CreateBlueprint(ctx, reqBody)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating blueprint",
			"Could not create blueprint: "+err.Error(),
		)
		return
	}

	blueprint, err := r.client.GetBlueprintByID(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading created blueprint",
			"Could not read created blueprint: "+err.Error(),
		)
		return
	}

	// Deploy the blueprint after successful creation
	err = r.client.DeployBlueprint(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deploying blueprint",
			"Blueprint was created successfully but could not be deployed: "+err.Error(),
		)
		return
	}

	updateModelFromAPIResponse(&plan, blueprint)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read reads the Blueprint resource state from the API.
func (r *BlueprintResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state blueprintResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	blueprint, err := r.client.GetBlueprintByID(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading blueprint",
			"Could not read blueprint: "+err.Error(),
		)
		return
	}

	updateModelFromAPIResponse(&state, blueprint)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update updates the Blueprint resource.
func (r *BlueprintResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan blueprintResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deviceGroups := make([]string, len(plan.DeviceGroups))
	for i, dg := range plan.DeviceGroups {
		deviceGroups[i] = dg.ValueString()
	}

	components := make([]client.BlueprintComponent, len(plan.Components))
	for i, comp := range plan.Components {
		component := client.BlueprintComponent{
			Identifier: comp.Identifier.ValueString(),
		}

		if !comp.Configuration.IsNull() && !comp.Configuration.IsUnknown() {
			configMap := make(map[string]string)
			diags := comp.Configuration.ElementsAs(ctx, &configMap, false)
			if diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}

			jsonObj := make(map[string]interface{})
			for key, value := range configMap {
				setNestedValue(jsonObj, key, value)
			}

			jsonBytes, err := json.Marshal(jsonObj)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error encoding component configuration",
					"Could not encode component configuration to JSON: "+err.Error(),
				)
				return
			}

			normalizedConfig := normalizeJSON(string(jsonBytes))
			component.Configuration = json.RawMessage(normalizedConfig)
		}
		components[i] = component
	}

	steps := []client.BlueprintStep{
		{
			Name:       "Declaration group",
			Components: components,
		},
	}

	updateReq := &client.BlueprintUpdateRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Scope: client.BlueprintUpdateScope{
			DeviceGroups: deviceGroups,
		},
		Steps: steps,
	}

	err := r.client.UpdateBlueprint(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating blueprint",
			"Could not update blueprint: "+err.Error(),
		)
		return
	}

	blueprint, err := r.client.GetBlueprintByID(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading updated blueprint",
			"Could not read updated blueprint: "+err.Error(),
		)
		return
	}

	// Deploy the blueprint after successful update
	err = r.client.DeployBlueprint(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deploying blueprint",
			"Blueprint was updated successfully but could not be deployed: "+err.Error(),
		)
		return
	}

	updateModelFromAPIResponse(&plan, blueprint)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete deletes the Blueprint resource.
func (r *BlueprintResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state blueprintResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteBlueprint(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting blueprint",
			"Could not delete blueprint: "+err.Error(),
		)
		return
	}
}
