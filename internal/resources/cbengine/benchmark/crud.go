// Copyright 2025 Jamf Software LLC.

package benchmark

import (
	"context"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Create creates a new Jamf Compliance Benchmark resource in Terraform.
func (r *BenchmarkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan benchmarkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	reqBody := &client.CBEngineBenchmarkRequest{
		Title:            plan.Title.ValueString(),
		Description:      plan.Description.ValueString(),
		SourceBaselineID: plan.SourceBaselineID.ValueString(),
		Sources:          make([]client.CBEngineSource, len(plan.Sources)),
		Rules:            make([]client.CBEngineRuleRequest, len(plan.Rules)),
		Target: client.CBEngineTarget{
			DeviceGroups: []string{plan.TargetDeviceGroup.ValueString()},
		},
		EnforcementMode: plan.EnforcementMode.ValueString(),
	}
	for i, s := range plan.Sources {
		reqBody.Sources[i] = client.CBEngineSource{
			Branch:   s.Branch.ValueString(),
			Revision: s.Revision.ValueString(),
		}
	}
	for i, rule := range plan.Rules {
		rr := client.CBEngineRuleRequest{
			ID:      rule.ID.ValueString(),
			Enabled: rule.Enabled.ValueBool(),
		}
		if !rule.ODVValue.IsNull() && rule.ODVValue.ValueString() != "" {
			rr.ODV = &client.CBEngineODVRequest{
				Value: rule.ODVValue.ValueString(),
			}
		}
		reqBody.Rules[i] = rr
	}

	bench, err := r.client.CreateCBEngineBenchmark(ctx, reqBody)
	if err != nil {
		resp.Diagnostics.AddError("Error creating benchmark", err.Error())
		return
	}

	plan.ID = types.StringValue(bench.BenchmarkID)
	plan.TenantID = types.StringValue(bench.TenantID)
	plan.Deleted = types.BoolValue(bench.Deleted)
	plan.UpdateAvailable = types.BoolValue(bench.UpdateAvailable)
	plan.LastUpdatedAt = types.StringValue(bench.LastUpdatedAt.Format("2006-01-02T15:04:05Z07:00"))

	for i, r := range bench.Rules {
		if i >= len(plan.Rules) {
			break
		}

		var references types.List
		if len(r.References) == 0 {
			references = types.ListNull(types.StringType)
		} else {
			vals := make([]attr.Value, len(r.References))
			for j, ref := range r.References {
				vals[j] = types.StringValue(ref)
			}
			references, _ = types.ListValue(types.StringType, vals)
		}

		var supportedOS types.List
		osInfoObjType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"os_type":         types.StringType,
				"os_version":      types.Int64Type,
				"management_type": types.StringType,
			},
		}
		if len(r.SupportedOS) == 0 {
			supportedOS = types.ListNull(osInfoObjType)
		} else {
			osVals := make([]attr.Value, len(r.SupportedOS))
			for j, os := range r.SupportedOS {
				osVals[j], _ = types.ObjectValue(
					map[string]attr.Type{
						"os_type":         types.StringType,
						"os_version":      types.Int64Type,
						"management_type": types.StringType,
					},
					map[string]attr.Value{
						"os_type":         types.StringValue(os.OSType),
						"os_version":      types.Int64Value(int64(os.OSVersion)),
						"management_type": types.StringValue(os.ManagementType),
					},
				)
			}
			supportedOS, _ = types.ListValue(osInfoObjType, osVals)
		}

		osSpecObjType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"title":       types.StringType,
				"description": types.StringType,
				"odv_value":   types.StringType,
				"odv_hint":    types.StringType,
			},
		}
		var osSpecificDefaults types.Map
		if len(r.OSSpecificDefaults) == 0 {
			osSpecificDefaults = types.MapNull(osSpecObjType)
		} else {
			vals := make(map[string]attr.Value, len(r.OSSpecificDefaults))
			for k, v := range r.OSSpecificDefaults {
				var odvValue, odvHint types.String
				if v.ODV != nil {
					odvValue = types.StringValue(v.ODV.Value)
					odvHint = types.StringValue(v.ODV.Hint)
				} else {
					odvValue = types.StringNull()
					odvHint = types.StringNull()
				}
				vals[k], _ = types.ObjectValue(
					map[string]attr.Type{
						"title":       types.StringType,
						"description": types.StringType,
						"odv_value":   types.StringType,
						"odv_hint":    types.StringType,
					},
					map[string]attr.Value{
						"title":       types.StringValue(v.Title),
						"description": types.StringValue(v.Description),
						"odv_value":   odvValue,
						"odv_hint":    odvHint,
					},
				)
			}
			osSpecificDefaults, _ = types.MapValue(osSpecObjType, vals)
		}

		var odvValue, odvHint, odvPlaceholder, odvType, odvValidationRegex types.String
		var odvValidationMin, odvValidationMax types.Int64
		var odvValidationEnumValues types.List
		if r.ODV != nil {
			odvValue = types.StringValue(r.ODV.Value)
			odvHint = types.StringValue(r.ODV.Hint)
			odvPlaceholder = types.StringValue(r.ODV.Placeholder)
			odvType = types.StringValue(r.ODV.Type)
			if r.ODV.Validation != nil {
				if r.ODV.Validation.Min != nil {
					odvValidationMin = types.Int64Value(int64(*r.ODV.Validation.Min))
				} else {
					odvValidationMin = types.Int64Null()
				}
				if r.ODV.Validation.Max != nil {
					odvValidationMax = types.Int64Value(int64(*r.ODV.Validation.Max))
				} else {
					odvValidationMax = types.Int64Null()
				}
				enumValues := make([]attr.Value, len(r.ODV.Validation.EnumValues))
				for k, v := range r.ODV.Validation.EnumValues {
					enumValues[k] = types.StringValue(v)
				}
				if len(enumValues) == 0 {
					odvValidationEnumValues = types.ListNull(types.StringType)
				} else {
					odvValidationEnumValues, _ = types.ListValue(types.StringType, enumValues)
				}
				odvValidationRegex = types.StringValue(r.ODV.Validation.Regex)
			} else {
				odvValidationMin = types.Int64Null()
				odvValidationMax = types.Int64Null()
				odvValidationEnumValues = types.ListNull(types.StringType)
				odvValidationRegex = types.StringNull()
			}
		} else {
			odvValue = types.StringNull()
			odvHint = types.StringNull()
			odvPlaceholder = types.StringNull()
			odvType = types.StringNull()
			odvValidationMin = types.Int64Null()
			odvValidationMax = types.Int64Null()
			odvValidationEnumValues = types.ListNull(types.StringType)
			odvValidationRegex = types.StringNull()
		}

		var dependsOn types.List
		if r.RuleRelation == nil || len(r.RuleRelation.DependsOn) == 0 {
			dependsOn = types.ListNull(types.StringType)
		} else {
			vals := make([]attr.Value, len(r.RuleRelation.DependsOn))
			for j, dep := range r.RuleRelation.DependsOn {
				vals[j] = types.StringValue(dep)
			}
			dependsOn, _ = types.ListValue(types.StringType, vals)
		}

		plan.Rules[i].SectionName = types.StringValue(r.SectionName)
		plan.Rules[i].Title = types.StringValue(r.Title)
		plan.Rules[i].References = references
		plan.Rules[i].Description = types.StringValue(r.Description)
		plan.Rules[i].SupportedOS = supportedOS
		plan.Rules[i].OSSpecificDefaults = osSpecificDefaults
		plan.Rules[i].ODVValue = odvValue
		plan.Rules[i].ODVHint = odvHint
		plan.Rules[i].ODVPlaceholder = odvPlaceholder
		plan.Rules[i].ODVType = odvType
		plan.Rules[i].ODVValidationMin = odvValidationMin
		plan.Rules[i].ODVValidationMax = odvValidationMax
		plan.Rules[i].ODVValidationEnumValues = odvValidationEnumValues
		plan.Rules[i].ODVValidationRegex = odvValidationRegex
		plan.Rules[i].DependsOn = dependsOn
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read reads the current state of a Jamf Compliance Benchmark resource from the API and updates the Terraform state.
func (r *BenchmarkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state benchmarkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.IsNull() || state.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot read benchmark without ID.")
		return
	}

	bench, err := r.client.GetCBEngineBenchmarkByID(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading benchmark", err.Error())
		return
	}

	state.Title = types.StringValue(bench.Title)
	state.Description = types.StringValue(bench.Description)
	state.TenantID = types.StringValue(bench.TenantID)
	state.Deleted = types.BoolValue(bench.Deleted)
	state.UpdateAvailable = types.BoolValue(bench.UpdateAvailable)
	state.LastUpdatedAt = types.StringValue(bench.LastUpdatedAt.Format("2006-01-02T15:04:05Z07:00"))

	state.Sources = make([]sourceModel, len(bench.Sources))
	for i, s := range bench.Sources {
		state.Sources[i] = sourceModel{
			Branch:   types.StringValue(s.Branch),
			Revision: types.StringValue(s.Revision),
		}
	}

	state.Rules = make([]ruleModel, len(bench.Rules))
	for i, r := range bench.Rules {
		var references types.List
		if len(r.References) == 0 {
			references = types.ListNull(types.StringType)
		} else {
			vals := make([]attr.Value, len(r.References))
			for j, ref := range r.References {
				vals[j] = types.StringValue(ref)
			}
			references, _ = types.ListValue(types.StringType, vals)
		}

		var supportedOS types.List
		osInfoObjType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"os_type":         types.StringType,
				"os_version":      types.Int64Type,
				"management_type": types.StringType,
			},
		}
		if len(r.SupportedOS) == 0 {
			supportedOS = types.ListNull(osInfoObjType)
		} else {
			osVals := make([]attr.Value, len(r.SupportedOS))
			for j, os := range r.SupportedOS {
				osVals[j], _ = types.ObjectValue(
					map[string]attr.Type{
						"os_type":         types.StringType,
						"os_version":      types.Int64Type,
						"management_type": types.StringType,
					},
					map[string]attr.Value{
						"os_type":         types.StringValue(os.OSType),
						"os_version":      types.Int64Value(int64(os.OSVersion)),
						"management_type": types.StringValue(os.ManagementType),
					},
				)
			}
			supportedOS, _ = types.ListValue(osInfoObjType, osVals)
		}

		osSpecObjType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"title":       types.StringType,
				"description": types.StringType,
				"odv_value":   types.StringType,
				"odv_hint":    types.StringType,
			},
		}
		var osSpecificDefaults types.Map
		if len(r.OSSpecificDefaults) == 0 {
			osSpecificDefaults = types.MapNull(osSpecObjType)
		} else {
			vals := make(map[string]attr.Value, len(r.OSSpecificDefaults))
			for k, v := range r.OSSpecificDefaults {
				var odvValue, odvHint types.String
				if v.ODV != nil {
					odvValue = types.StringValue(v.ODV.Value)
					odvHint = types.StringValue(v.ODV.Hint)
				} else {
					odvValue = types.StringNull()
					odvHint = types.StringNull()
				}
				vals[k], _ = types.ObjectValue(
					map[string]attr.Type{
						"title":       types.StringType,
						"description": types.StringType,
						"odv_value":   types.StringType,
						"odv_hint":    types.StringType,
					},
					map[string]attr.Value{
						"title":       types.StringValue(v.Title),
						"description": types.StringValue(v.Description),
						"odv_value":   odvValue,
						"odv_hint":    odvHint,
					},
				)
			}
			osSpecificDefaults, _ = types.MapValue(osSpecObjType, vals)
		}

		var odvValue, odvHint, odvPlaceholder, odvType, odvValidationRegex types.String
		var odvValidationMin, odvValidationMax types.Int64
		var odvValidationEnumValues types.List
		if r.ODV != nil {
			odvValue = types.StringValue(r.ODV.Value)
			odvHint = types.StringValue(r.ODV.Hint)
			odvPlaceholder = types.StringValue(r.ODV.Placeholder)
			odvType = types.StringValue(r.ODV.Type)
			if r.ODV.Validation != nil {
				if r.ODV.Validation.Min != nil {
					odvValidationMin = types.Int64Value(int64(*r.ODV.Validation.Min))
				} else {
					odvValidationMin = types.Int64Null()
				}
				if r.ODV.Validation.Max != nil {
					odvValidationMax = types.Int64Value(int64(*r.ODV.Validation.Max))
				} else {
					odvValidationMax = types.Int64Null()
				}
				enumValues := make([]attr.Value, len(r.ODV.Validation.EnumValues))
				for k, v := range r.ODV.Validation.EnumValues {
					enumValues[k] = types.StringValue(v)
				}
				if len(enumValues) == 0 {
					odvValidationEnumValues = types.ListNull(types.StringType)
				} else {
					odvValidationEnumValues, _ = types.ListValue(types.StringType, enumValues)
				}
				odvValidationRegex = types.StringValue(r.ODV.Validation.Regex)
			} else {
				odvValidationMin = types.Int64Null()
				odvValidationMax = types.Int64Null()
				odvValidationEnumValues = types.ListNull(types.StringType)
				odvValidationRegex = types.StringNull()
			}
		} else {
			odvValue = types.StringNull()
			odvHint = types.StringNull()
			odvPlaceholder = types.StringNull()
			odvType = types.StringNull()
			odvValidationMin = types.Int64Null()
			odvValidationMax = types.Int64Null()
			odvValidationEnumValues = types.ListNull(types.StringType)
			odvValidationRegex = types.StringNull()
		}

		var dependsOn types.List
		if r.RuleRelation == nil || len(r.RuleRelation.DependsOn) == 0 {
			dependsOn = types.ListNull(types.StringType)
		} else {
			vals := make([]attr.Value, len(r.RuleRelation.DependsOn))
			for j, dep := range r.RuleRelation.DependsOn {
				vals[j] = types.StringValue(dep)
			}
			dependsOn, _ = types.ListValue(types.StringType, vals)
		}

		state.Rules[i] = ruleModel{
			ID:                      types.StringValue(r.ID),
			Enabled:                 types.BoolValue(r.Enabled),
			SectionName:             types.StringValue(r.SectionName),
			Title:                   types.StringValue(r.Title),
			References:              references,
			Description:             types.StringValue(r.Description),
			SupportedOS:             supportedOS,
			OSSpecificDefaults:      osSpecificDefaults,
			ODVValue:                odvValue,
			ODVHint:                 odvHint,
			ODVPlaceholder:          odvPlaceholder,
			ODVType:                 odvType,
			ODVValidationMin:        odvValidationMin,
			ODVValidationMax:        odvValidationMax,
			ODVValidationEnumValues: odvValidationEnumValues,
			ODVValidationRegex:      odvValidationRegex,
			DependsOn:               dependsOn,
		}
	}

	if len(bench.Target.DeviceGroups) > 0 {
		state.TargetDeviceGroup = types.StringValue(bench.Target.DeviceGroups[0])
	} else {
		state.TargetDeviceGroup = types.StringNull()
	}
	state.EnforcementMode = types.StringValue(bench.EnforcementMode)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is not supported for Jamf Compliance Benchmark resources. The resource must be recreated to apply changes.
func (r *BenchmarkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update Not Supported", "This resource must be destroyed and recreated to apply changes.")
}

// Delete deletes a Jamf Compliance Benchmark resource from the API and removes it from the Terraform state.
func (r *BenchmarkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state benchmarkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.IsNull() || state.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot delete benchmark without ID.")
		return
	}

	err := r.client.DeleteCBEngineBenchmark(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting benchmark", err.Error())
		return
	}
}
