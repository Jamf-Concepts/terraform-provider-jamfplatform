// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package benchmark

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/compliancebenchmarks"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// buildDeviceGroupsRequest extracts the device group IDs the user supplied. Plural
// target_device_groups takes precedence when configured; otherwise the deprecated
// singular target_device_group is wrapped into a single-element slice. Schema
// validation guarantees exactly one is supplied so this never observes both.
func buildDeviceGroupsRequest(data *BenchmarkResourceModel) []string {
	if !data.TargetDeviceGroups.IsNull() && !data.TargetDeviceGroups.IsUnknown() {
		elements := data.TargetDeviceGroups.Elements()
		out := make([]string, 0, len(elements))
		for _, el := range elements {
			s, ok := el.(types.String)
			if !ok || s.IsNull() || s.IsUnknown() {
				continue
			}
			out = append(out, s.ValueString())
		}
		return out
	}
	return []string{data.TargetDeviceGroup.ValueString()}
}

// buildBenchmarkRequest constructs the API request body from the Terraform plan model.
// Device group targeting accepts either the deprecated singular target_device_group
// or the preferred plural target_device_groups set; ConflictsWith + AtLeastOneOf on
// the schema guarantee exactly one is configured.
func buildBenchmarkRequest(data *BenchmarkResourceModel) *compliancebenchmarks.BenchmarkRequestV2 {
	reqBody := &compliancebenchmarks.BenchmarkRequestV2{
		Title:            data.Title.ValueString(),
		Description:      data.Description.ValueStringPointer(),
		SourceBaselineID: data.SourceBaselineID.ValueString(),
		Sources:          make([]compliancebenchmarks.Source, len(data.Sources)),
		Rules:            make([]compliancebenchmarks.RuleRequest, len(data.Rules)),
		Target: compliancebenchmarks.TargetV2{
			DeviceGroups: buildDeviceGroupsRequest(data),
		},
		EnforcementMode: data.EnforcementMode.ValueString(),
	}
	for i, s := range data.Sources {
		reqBody.Sources[i] = compliancebenchmarks.Source{
			Branch:   s.Branch.ValueString(),
			Revision: s.Revision.ValueString(),
		}
	}
	for i, rule := range data.Rules {
		rr := compliancebenchmarks.RuleRequest{
			ID:      rule.ID.ValueString(),
			Enabled: rule.Enabled.ValueBool(),
		}
		if v := rule.ODVValue.ValueString(); v != "" {
			rr.ODV = &compliancebenchmarks.ODVRequest{Value: v}
		}
		reqBody.Rules[i] = rr
	}
	return reqBody
}
