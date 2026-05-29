// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package re_enrollment_settings

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// assignReEnrollmentSettingsResourceModel populates resource state from a
// Re-enrollment settings GET response. The server always returns a concrete
// value for every field, so each attribute lands as a known value.
func assignReEnrollmentSettingsResourceModel(state *ReEnrollmentSettingsResourceModel, s *pro.Reenrollment) {
	if s == nil {
		return
	}
	state.ClearPolicyLogs = helpers.BoolPointerValueOrNull(s.IsFlushPolicyHistoryEnabled)
	state.ClearLocationInformation = helpers.BoolPointerValueOrNull(s.IsFlushLocationInformationEnabled)
	state.ClearLocationInformationHistory = helpers.BoolPointerValueOrNull(s.IsFlushLocationInformationHistoryEnabled)
	state.ClearExtensionAttributes = helpers.BoolPointerValueOrNull(s.IsFlushExtensionAttributesEnabled)
	state.ClearSoftwareUpdatePlans = helpers.BoolPointerValueOrNull(s.IsFlushSoftwareUpdatePlansEnabled)
	state.ClearManagementHistory = stringValueOrNull(s.FlushMDMQueue)
}

// assignReEnrollmentSettingsDataSourceModel populates data source state from a
// Re-enrollment settings GET response.
func assignReEnrollmentSettingsDataSourceModel(state *ReEnrollmentSettingsDataSourceModel, s *pro.Reenrollment) {
	if s == nil {
		return
	}
	state.ClearPolicyLogs = helpers.BoolPointerValueOrNull(s.IsFlushPolicyHistoryEnabled)
	state.ClearLocationInformation = helpers.BoolPointerValueOrNull(s.IsFlushLocationInformationEnabled)
	state.ClearLocationInformationHistory = helpers.BoolPointerValueOrNull(s.IsFlushLocationInformationHistoryEnabled)
	state.ClearExtensionAttributes = helpers.BoolPointerValueOrNull(s.IsFlushExtensionAttributesEnabled)
	state.ClearSoftwareUpdatePlans = helpers.BoolPointerValueOrNull(s.IsFlushSoftwareUpdatePlansEnabled)
	state.ClearManagementHistory = stringValueOrNull(s.FlushMDMQueue)
}

// stringValueOrNull maps an empty wire string to a null state value.
func stringValueOrNull(v string) types.String {
	if v == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}
