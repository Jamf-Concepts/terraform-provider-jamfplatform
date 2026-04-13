// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package benchmark

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// assignBenchmarkModelFromResponse maps the API response into the Terraform benchmark resource model.
func assignBenchmarkModelFromResponse(model *BenchmarkResourceModel, bench *jamfplatform.BenchmarkResponseV2) {
	if model == nil || bench == nil {
		return
	}

	model.ID = types.StringValue(bench.BenchmarkID)
	model.Title = types.StringValue(bench.Title)
	model.Description = types.StringValue(bench.Description)
	model.TenantID = types.StringValue(bench.TenantID)
	model.Deleted = types.BoolValue(bench.Deleted)
	model.UpdateAvailable = types.BoolValue(bench.UpdateAvailable)
	model.CanSwitchToEnforce = types.BoolValue(bench.CanSwitchToEnforce)
	model.LastUpdatedAt = types.StringValue(bench.LastUpdatedAt)
	model.Sources = buildSourceModels(bench.Sources)
	model.Rules = buildRuleModels(bench.Rules)
	if bench.Target != nil {
		model.TargetDeviceGroup = buildTargetDeviceGroup(bench.Target.DeviceGroups)
	} else {
		model.TargetDeviceGroup = types.StringNull()
	}
	model.EnforcementMode = types.StringValue(bench.EnforcementMode)
}

// assignBenchmarkDataSourceFromResponse maps the API response into the Terraform benchmark data source model.
func assignBenchmarkDataSourceFromResponse(model *BenchmarkDataSourceModel, bench *jamfplatform.BenchmarkResponseV2) {
	if model == nil || bench == nil {
		return
	}

	model.BenchmarkID = types.StringValue(bench.BenchmarkID)
	model.TenantID = types.StringValue(bench.TenantID)
	model.Title = types.StringValue(bench.Title)
	model.Description = types.StringValue(bench.Description)
	model.Sources = buildSourceModels(bench.Sources)
	model.Rules = buildRuleModels(bench.Rules)
	if bench.Target != nil {
		model.TargetDeviceGroup = buildTargetDeviceGroup(bench.Target.DeviceGroups)
	} else {
		model.TargetDeviceGroup = types.StringNull()
	}
	model.EnforcementMode = types.StringValue(bench.EnforcementMode)
	model.Deleted = types.BoolValue(bench.Deleted)
	model.UpdateAvailable = types.BoolValue(bench.UpdateAvailable)
	model.CanSwitchToEnforce = types.BoolValue(bench.CanSwitchToEnforce)
	model.LastUpdatedAt = types.StringValue(bench.LastUpdatedAt)
}

// buildSourceModels converts API source representations into Terraform source models.
func buildSourceModels(sources []jamfplatform.Source) []SourceModel {
	result := make([]SourceModel, len(sources))
	for i, s := range sources {
		result[i] = SourceModel{
			Branch:   types.StringValue(s.Branch),
			Revision: types.StringValue(s.Revision),
		}
	}
	return result
}

// buildRuleModels converts API rule responses into Terraform rule models.
func buildRuleModels(rules []jamfplatform.RuleInfo) []RuleModel {
	result := make([]RuleModel, len(rules))
	for i, r := range rules {
		result[i] = buildRuleModel(r)
	}
	return result
}

// buildRuleModel converts a single API rule response into a Terraform rule model.
func buildRuleModel(r jamfplatform.RuleInfo) RuleModel {
	return RuleModel{
		ID:                      types.StringValue(r.ID),
		SectionName:             types.StringValue(r.SectionName),
		Enabled:                 types.BoolValue(r.Enabled),
		Title:                   types.StringValue(r.Title),
		Description:             types.StringValue(r.Description),
		References:              buildStringList(r.References),
		SupportedOS:             buildSupportedOSList(r.SupportedOs),
		OSSpecificDefaults:      buildOSSpecificDefaultsMap(r.OsSpecificDefaults),
		ODVValue:                odvStringValue(r.ODV, func(o *jamfplatform.OrganizationDefinedValue) string { return o.Value }),
		ODVHint:                 odvStringValue(r.ODV, func(o *jamfplatform.OrganizationDefinedValue) string { return o.Hint }),
		ODVPlaceholder:          odvStringValue(r.ODV, func(o *jamfplatform.OrganizationDefinedValue) string { return o.Placeholder }),
		ODVType:                 odvStringValue(r.ODV, func(o *jamfplatform.OrganizationDefinedValue) string { return o.Type }),
		ODVValidationMin:        buildODVValidationMin(r.ODV),
		ODVValidationMax:        buildODVValidationMax(r.ODV),
		ODVValidationEnumValues: buildODVValidationEnumValues(r.ODV),
		ODVValidationRegex:      buildODVValidationRegex(r.ODV),
		DependsOn:               buildDependsOnList(r.RuleRelation),
		Reportable:              types.BoolValue(r.Reportable),
		SmartCard:               types.BoolValue(r.SmartCard),
	}
}

// buildTargetDeviceGroup extracts the first device group ID or returns null.
func buildTargetDeviceGroup(deviceGroups []string) types.String {
	if len(deviceGroups) > 0 {
		return types.StringValue(deviceGroups[0])
	}
	return types.StringNull()
}

// buildStringList converts a string slice into a Terraform list of strings, returning null for empty.
func buildStringList(values []string) types.List {
	if len(values) == 0 {
		return types.ListNull(types.StringType)
	}
	vals := make([]attr.Value, len(values))
	for i, v := range values {
		vals[i] = types.StringValue(v)
	}
	result, _ := types.ListValue(types.StringType, vals)
	return result
}

// buildSupportedOSList converts API OS info into a Terraform list of objects.
func buildSupportedOSList(supportedOS []jamfplatform.OsInfo) types.List {
	if len(supportedOS) == 0 {
		return types.ListNull(osInfoObjectType)
	}
	osVals := make([]attr.Value, len(supportedOS))
	for i, osInfo := range supportedOS {
		osVals[i], _ = types.ObjectValue(
			osInfoObjectType.AttrTypes,
			map[string]attr.Value{
				"os_type":         types.StringValue(osInfo.OsType),
				"os_version":      types.Int64Value(int64(osInfo.OsVersion)),
				"management_type": types.StringValue(osInfo.ManagementType),
			},
		)
	}
	result, _ := types.ListValue(osInfoObjectType, osVals)
	return result
}

// buildOSSpecificDefaultsMap converts API OS-specific defaults into a Terraform map of objects.
func buildOSSpecificDefaultsMap(defaults map[string]jamfplatform.OsSpecificRuleInfo) types.Map {
	if len(defaults) == 0 {
		return types.MapNull(osSpecificDefaultObjectType)
	}
	vals := make(map[string]attr.Value, len(defaults))
	for key, def := range defaults {
		var odvValue, odvHint types.String
		if def.ODV != nil {
			odvValue = types.StringValue(def.ODV.Value)
			odvHint = types.StringValue(def.ODV.Hint)
		} else {
			odvValue = types.StringNull()
			odvHint = types.StringNull()
		}
		vals[key], _ = types.ObjectValue(
			osSpecificDefaultObjectType.AttrTypes,
			map[string]attr.Value{
				"title":       types.StringValue(def.Title),
				"description": types.StringValue(def.Description),
				"odv_value":   odvValue,
				"odv_hint":    odvHint,
			},
		)
	}
	result, _ := types.MapValue(osSpecificDefaultObjectType, vals)
	return result
}

// odvStringValue extracts a string field from an ODV response, returning null if ODV is nil.
func odvStringValue(odv *jamfplatform.OrganizationDefinedValue, getter func(*jamfplatform.OrganizationDefinedValue) string) types.String {
	if odv == nil {
		return types.StringNull()
	}
	return types.StringValue(getter(odv))
}

// buildODVValidationMin extracts the min validation value from the ODV response.
func buildODVValidationMin(odv *jamfplatform.OrganizationDefinedValue) types.Int64 {
	if odv == nil || odv.Validation == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(odv.Validation.Min))
}

// buildODVValidationMax extracts the max validation value from the ODV response.
func buildODVValidationMax(odv *jamfplatform.OrganizationDefinedValue) types.Int64 {
	if odv == nil || odv.Validation == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(odv.Validation.Max))
}

// buildODVValidationEnumValues extracts the enum validation values from the ODV response.
func buildODVValidationEnumValues(odv *jamfplatform.OrganizationDefinedValue) types.List {
	if odv == nil || odv.Validation == nil || len(odv.Validation.EnumValues) == 0 {
		return types.ListNull(types.StringType)
	}
	return buildStringList(odv.Validation.EnumValues)
}

// buildODVValidationRegex extracts the regex validation pattern from the ODV response.
func buildODVValidationRegex(odv *jamfplatform.OrganizationDefinedValue) types.String {
	if odv == nil || odv.Validation == nil {
		return types.StringNull()
	}
	return types.StringValue(odv.Validation.Regex)
}

// buildDependsOnList extracts the depends_on list from a rule relation.
func buildDependsOnList(relation *jamfplatform.RuleRelation) types.List {
	if relation == nil || len(relation.DependsOn) == 0 {
		return types.ListNull(types.StringType)
	}
	return buildStringList(relation.DependsOn)
}
