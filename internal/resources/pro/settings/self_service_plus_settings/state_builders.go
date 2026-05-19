// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package self_service_plus_settings

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// assignSelfServicePlusSettingsResourceModel populates a resource model from an SDK response.
// Missing enabled in the response is treated as false — the API omits the field when unset.
func assignSelfServicePlusSettingsResourceModel(state *SelfServicePlusSettingsResourceModel, s *pro.SelfServicePlusSettings) {
	if s.Enabled != nil {
		state.Enabled = types.BoolValue(*s.Enabled)
	} else {
		state.Enabled = types.BoolValue(false)
	}
}

// assignSelfServicePlusSettingsDataSourceModel populates a data source model from an SDK response.
func assignSelfServicePlusSettingsDataSourceModel(state *SelfServicePlusSettingsDataSourceModel, s *pro.SelfServicePlusSettings) {
	if s.Enabled != nil {
		state.Enabled = types.BoolValue(*s.Enabled)
	} else {
		state.Enabled = types.BoolValue(false)
	}
}
