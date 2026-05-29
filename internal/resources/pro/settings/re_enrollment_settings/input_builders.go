// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package re_enrollment_settings

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
)

// buildReenrollmentInput converts the Terraform plan model into a Re-enrollment
// settings PUT payload.
//
// The write is a full replace over all six fields, so every field is populated
// unconditionally:
//
//   - The five flush booleans are sent as concrete *bool. An omitted
//     (null/unknown) attribute resolves via ValueBool() to false, which is the
//     intended full-replace semantic — an unticked checkbox.
//
//   - FlushMDMQueue (clear_management_history) is a Required enum, so it is
//     always known on the wire — sent verbatim from the plan.
func buildReenrollmentInput(plan ReEnrollmentSettingsResourceModel) *pro.Reenrollment {
	policyLogs := plan.ClearPolicyLogs.ValueBool()
	locationInfo := plan.ClearLocationInformation.ValueBool()
	locationInfoHistory := plan.ClearLocationInformationHistory.ValueBool()
	extensionAttrs := plan.ClearExtensionAttributes.ValueBool()
	softwareUpdatePlans := plan.ClearSoftwareUpdatePlans.ValueBool()

	return &pro.Reenrollment{
		FlushMDMQueue:                            plan.ClearManagementHistory.ValueString(),
		IsFlushPolicyHistoryEnabled:              &policyLogs,
		IsFlushLocationInformationEnabled:        &locationInfo,
		IsFlushLocationInformationHistoryEnabled: &locationInfoHistory,
		IsFlushExtensionAttributesEnabled:        &extensionAttrs,
		IsFlushSoftwareUpdatePlansEnabled:        &softwareUpdatePlans,
	}
}
