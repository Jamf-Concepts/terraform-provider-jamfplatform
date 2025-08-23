// Copyright 2025 Jamf Software LLC.

package blueprint

import (
	"context"
	"encoding/json"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

	steps := make([]client.BlueprintStep, len(plan.Steps))
	for i, step := range plan.Steps {
		components := make([]client.BlueprintComponent, len(step.Components))
		for j, comp := range step.Components {
			component := client.BlueprintComponent{
				Identifier: comp.Identifier.ValueString(),
			}
			if !comp.Configuration.IsNull() && !comp.Configuration.IsUnknown() {
				configStr := comp.Configuration.ValueString()
				if configStr != "" {
					normalizedConfig := normalizeJSON(configStr)
					component.Configuration = json.RawMessage(normalizedConfig)
				}
			}
			components[j] = component
		}
		steps[i] = client.BlueprintStep{
			Name:       step.Name.ValueString(),
			Components: components,
		}
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

	steps := make([]client.BlueprintStep, len(plan.Steps))
	for i, step := range plan.Steps {
		components := make([]client.BlueprintComponent, len(step.Components))
		for j, comp := range step.Components {
			component := client.BlueprintComponent{
				Identifier: comp.Identifier.ValueString(),
			}
			if !comp.Configuration.IsNull() && !comp.Configuration.IsUnknown() {
				configStr := comp.Configuration.ValueString()
				if configStr != "" {
					normalizedConfig := normalizeJSON(configStr)
					component.Configuration = json.RawMessage(normalizedConfig)
				}
			}
			components[j] = component
		}
		steps[i] = client.BlueprintStep{
			Name:       step.Name.ValueString(),
			Components: components,
		}
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

// updateModelFromAPIResponse updates the Terraform model with data from the API response.
func updateModelFromAPIResponse(model *blueprintResourceModel, blueprint *client.BlueprintDetail) {
	model.ID = types.StringValue(blueprint.ID)
	model.Name = types.StringValue(blueprint.Name)
	model.Description = types.StringValue(blueprint.Description)
	model.Created = types.StringValue(blueprint.Created)
	model.Updated = types.StringValue(blueprint.Updated)
	model.DeploymentState = types.StringValue(blueprint.DeploymentState.State)

	deviceGroups := make([]types.String, len(blueprint.Scope.DeviceGroups))
	for i, dg := range blueprint.Scope.DeviceGroups {
		deviceGroups[i] = types.StringValue(dg)
	}
	model.DeviceGroups = deviceGroups

	steps := make([]stepModel, len(blueprint.Steps))
	for i, step := range blueprint.Steps {
		components := make([]componentModel, len(step.Components))
		for j, comp := range step.Components {
			component := componentModel{
				Identifier: types.StringValue(comp.Identifier),
			}
			if comp.Configuration != nil {
				normalizedConfig := normalizeJSON(string(comp.Configuration))
				component.Configuration = types.StringValue(normalizedConfig)
			} else {
				component.Configuration = types.StringNull()
			}
			components[j] = component
		}
		steps[i] = stepModel{
			Name:       types.StringValue(step.Name),
			Components: components,
		}
	}
	model.Steps = steps
}

// normalizeJSON takes a JSON string and returns it with sorted keys to ensure consistent comparison
func normalizeJSON(jsonStr string) string {
	if jsonStr == "" {
		return ""
	}

	var obj interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return jsonStr
	}

	normalized, err := json.Marshal(obj)
	if err != nil {
		return jsonStr
	}

	return string(normalized)
}
