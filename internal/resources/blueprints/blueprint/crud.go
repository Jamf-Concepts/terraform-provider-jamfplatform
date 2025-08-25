// Copyright 2025 Jamf Software LLC.

package blueprint

import (
	"context"
	"encoding/json"

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

	deviceGroups := make([]string, len(data.DeviceGroups))
	for i, dg := range data.DeviceGroups {
		deviceGroups[i] = dg.ValueString()
	}

	components := make([]client.BlueprintComponent, len(data.Components))
	for i, comp := range data.Components {
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
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
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

	err = r.client.DeployBlueprint(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deploying blueprint",
			"Blueprint was created successfully but could not be deployed: "+err.Error(),
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

	blueprint, err := r.client.GetBlueprintByID(ctx, data.ID.ValueString())
	if err != nil {
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

	deviceGroups := make([]string, len(data.DeviceGroups))
	for i, dg := range data.DeviceGroups {
		deviceGroups[i] = dg.ValueString()
	}

	components := make([]client.BlueprintComponent, len(data.Components))
	for i, comp := range data.Components {
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
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Scope: client.BlueprintUpdateScope{
			DeviceGroups: deviceGroups,
		},
		Steps: steps,
	}

	err := r.client.UpdateBlueprint(ctx, data.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating blueprint",
			"Could not update blueprint: "+err.Error(),
		)
		return
	}

	err = r.client.DeployBlueprint(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deploying blueprint",
			"Blueprint was updated successfully but could not be deployed: "+err.Error(),
		)
		return
	}

	blueprint, err := r.client.GetBlueprintByID(ctx, data.ID.ValueString())
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

	if data.ID.IsNull() || data.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot delete blueprint without ID.")
		return
	}

	err := r.client.DeleteBlueprint(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting blueprint",
			"Could not delete blueprint: "+err.Error(),
		)
		return
	}
}
