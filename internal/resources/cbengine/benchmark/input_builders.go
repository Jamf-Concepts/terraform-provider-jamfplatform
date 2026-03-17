// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package benchmark

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// buildBenchmarkRequest constructs the API request body from the Terraform plan model.
func buildBenchmarkRequest(data *BenchmarkResourceModel) *jamfplatform.CBEngineBenchmarkRequestV2 {
	reqBody := &jamfplatform.CBEngineBenchmarkRequestV2{
		Title:            data.Title.ValueString(),
		Description:      data.Description.ValueString(),
		SourceBaselineID: data.SourceBaselineID.ValueString(),
		Sources:          make([]jamfplatform.CBEngineSourceV1, len(data.Sources)),
		Rules:            make([]jamfplatform.CBEngineRuleRequestV2, len(data.Rules)),
		Target: jamfplatform.CBEngineTargetV2{
			DeviceGroups: []string{data.TargetDeviceGroup.ValueString()},
		},
		EnforcementMode: data.EnforcementMode.ValueString(),
	}
	for i, s := range data.Sources {
		reqBody.Sources[i] = jamfplatform.CBEngineSourceV1{
			Branch:   s.Branch.ValueString(),
			Revision: s.Revision.ValueString(),
		}
	}
	for i, rule := range data.Rules {
		rr := jamfplatform.CBEngineRuleRequestV2{
			ID:      rule.ID.ValueString(),
			Enabled: rule.Enabled.ValueBool(),
		}
		if value := helpers.StringPointerValue(rule.ODVValue); value != nil {
			rr.ODV = &jamfplatform.CBEngineODVRequestV2{Value: *value}
		}
		reqBody.Rules[i] = rr
	}
	return reqBody
}
