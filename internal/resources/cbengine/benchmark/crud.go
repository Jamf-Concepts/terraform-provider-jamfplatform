// Copyright 2025 Jamf Software LLC.

package benchmark

import (
	"context"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Create creates a new Jamf Compliance Benchmark resource in Terraform.
func (r *BenchmarkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BenchmarkResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	reqBody := &client.CBEngineBenchmarkRequest{
		Title:            data.Title.ValueString(),
		Description:      data.Description.ValueString(),
		SourceBaselineID: data.SourceBaselineID.ValueString(),
		Sources:          make([]client.CBEngineSource, len(data.Sources)),
		Rules:            make([]client.CBEngineRuleRequest, len(data.Rules)),
		Target: client.CBEngineTarget{
			DeviceGroups: []string{data.TargetDeviceGroup.ValueString()},
		},
		EnforcementMode: data.EnforcementMode.ValueString(),
	}
	for i, s := range data.Sources {
		reqBody.Sources[i] = client.CBEngineSource{
			Branch:   s.Branch.ValueString(),
			Revision: s.Revision.ValueString(),
		}
	}
	for i, rule := range data.Rules {
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

	data.ID = types.StringValue(bench.BenchmarkID)
	data.TenantID = types.StringValue(bench.TenantID)
	data.Deleted = types.BoolValue(bench.Deleted)
	data.UpdateAvailable = types.BoolValue(bench.UpdateAvailable)
	data.LastUpdatedAt = types.StringValue(bench.LastUpdatedAt.Format("2006-01-02T15:04:05Z07:00"))

	for i, r := range bench.Rules {
		if i >= len(data.Rules) {
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

		data.Rules[i].SectionName = types.StringValue(r.SectionName)
		data.Rules[i].Title = types.StringValue(r.Title)
		data.Rules[i].References = references
		data.Rules[i].Description = types.StringValue(r.Description)
		data.Rules[i].SupportedOS = supportedOS
		data.Rules[i].OSSpecificDefaults = osSpecificDefaults
		data.Rules[i].ODVValue = odvValue
		data.Rules[i].ODVHint = odvHint
		data.Rules[i].ODVPlaceholder = odvPlaceholder
		data.Rules[i].ODVType = odvType
		data.Rules[i].ODVValidationMin = odvValidationMin
		data.Rules[i].ODVValidationMax = odvValidationMax
		data.Rules[i].ODVValidationEnumValues = odvValidationEnumValues
		data.Rules[i].ODVValidationRegex = odvValidationRegex
		data.Rules[i].DependsOn = dependsOn
	}

	tflog.Trace(ctx, "created a resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the current state of a Jamf Compliance Benchmark resource from the API and updates the Terraform state.
func (r *BenchmarkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BenchmarkResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() || data.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot read benchmark without ID.")
		return
	}

	bench, err := r.client.GetCBEngineBenchmarkByID(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			tflog.Info(ctx, "Benchmark not found, removing from state", map[string]interface{}{
				"benchmark_id": data.ID.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error reading benchmark", err.Error())
		return
	}

	data.Title = types.StringValue(bench.Title)
	data.Description = types.StringValue(bench.Description)
	data.TenantID = types.StringValue(bench.TenantID)
	data.Deleted = types.BoolValue(bench.Deleted)
	data.UpdateAvailable = types.BoolValue(bench.UpdateAvailable)
	data.LastUpdatedAt = types.StringValue(bench.LastUpdatedAt.Format("2006-01-02T15:04:05Z07:00"))

	data.Sources = make([]SourceModel, len(bench.Sources))
	for i, s := range bench.Sources {
		data.Sources[i] = SourceModel{
			Branch:   types.StringValue(s.Branch),
			Revision: types.StringValue(s.Revision),
		}
	}

	data.Rules = make([]RuleModel, len(bench.Rules))
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

		data.Rules[i] = RuleModel{
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
		data.TargetDeviceGroup = types.StringValue(bench.Target.DeviceGroups[0])
	} else {
		data.TargetDeviceGroup = types.StringNull()
	}
	data.EnforcementMode = types.StringValue(bench.EnforcementMode)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update is not supported for Jamf Compliance Benchmark resources. The resource must be recreated to apply changes.
func (r *BenchmarkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update Not Supported", "This resource must be destroyed and recreated to apply changes.")
}

// Delete deletes a Jamf Compliance Benchmark resource from the API and removes it from the Terraform state.
func (r *BenchmarkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BenchmarkResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() || data.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot delete benchmark without ID.")
		return
	}

	err := r.client.DeleteCBEngineBenchmark(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			tflog.Info(ctx, "Benchmark already deleted", map[string]interface{}{
				"benchmark_id": data.ID.ValueString(),
			})
			return
		}

		resp.Diagnostics.AddError(
			"Error deleting benchmark",
			"Could not delete benchmark: "+err.Error(),
		)
		return
	}
}
