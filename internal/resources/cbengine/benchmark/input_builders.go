// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package benchmark

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/compliancebenchmarks"
)

// buildBenchmarkRequest constructs the API request body from the Terraform plan model.
func buildBenchmarkRequest(data *BenchmarkResourceModel) *compliancebenchmarks.BenchmarkRequestV2 {
	reqBody := &compliancebenchmarks.BenchmarkRequestV2{
		Title:            data.Title.ValueString(),
		Description:      data.Description.ValueStringPointer(),
		SourceBaselineID: data.SourceBaselineID.ValueString(),
		Sources:          make([]compliancebenchmarks.Source, len(data.Sources)),
		Rules:            make([]compliancebenchmarks.RuleRequest, len(data.Rules)),
		Target: compliancebenchmarks.TargetV2{
			DeviceGroups: []string{data.TargetDeviceGroup.ValueString()},
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
