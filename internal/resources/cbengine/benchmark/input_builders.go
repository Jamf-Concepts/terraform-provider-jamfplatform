// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package benchmark

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// buildBenchmarkRequest constructs the API request body from the Terraform plan model.
func buildBenchmarkRequest(data *BenchmarkResourceModel) *client.CBEngineBenchmarkRequestV2 {
	reqBody := &client.CBEngineBenchmarkRequestV2{
		Title:            data.Title.ValueString(),
		Description:      data.Description.ValueString(),
		SourceBaselineID: data.SourceBaselineID.ValueString(),
		Sources:          make([]client.CBEngineSourceV1, len(data.Sources)),
		Rules:            make([]client.CBEngineRuleRequestV2, len(data.Rules)),
		Target: client.CBEngineTargetV2{
			DeviceGroups: []string{data.TargetDeviceGroup.ValueString()},
		},
		EnforcementMode: data.EnforcementMode.ValueString(),
	}
	for i, s := range data.Sources {
		reqBody.Sources[i] = client.CBEngineSourceV1{
			Branch:   s.Branch.ValueString(),
			Revision: s.Revision.ValueString(),
		}
	}
	for i, rule := range data.Rules {
		rr := client.CBEngineRuleRequestV2{
			ID:      rule.ID.ValueString(),
			Enabled: rule.Enabled.ValueBool(),
		}
		if value := helpers.StringPointerValue(rule.ODVValue); value != nil {
			rr.ODV = &client.CBEngineODVRequestV2{Value: *value}
		}
		reqBody.Rules[i] = rr
	}
	return reqBody
}
