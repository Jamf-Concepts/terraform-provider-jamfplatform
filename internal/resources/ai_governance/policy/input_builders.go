// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"encoding/json"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/aigovernance"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// buildCreateRequest assembles the create payload.
func buildCreateRequest(model *policyModel) *aigovernance.CreatePolicyRequest {
	return &aigovernance.CreatePolicyRequest{
		Name:          model.Name.ValueString(),
		Description:   optionalStringPointer(model),
		ToolID:        model.ToolID.ValueString(),
		SchemaVersion: model.SchemaVersion.ValueString(),
		Settings:      settingsPayload(model),
	}
}

// buildUpdateRequest assembles the update payload.
//
// Name and description are optional on the wire and mean "leave unchanged" when omitted, but they
// are always sent: Terraform's model is the desired state in full, so a name the operator removed
// from the configuration must be written back, not silently preserved. Schema version and settings
// are mandatory on every update — omitting either is refused — which makes settings a full replace.
func buildUpdateRequest(model *policyModel) *aigovernance.UpdatePolicyRequest {
	name := model.Name.ValueString()
	return &aigovernance.UpdatePolicyRequest{
		Name:          &name,
		Description:   optionalStringPointer(model),
		SchemaVersion: model.SchemaVersion.ValueString(),
		Settings:      settingsPayload(model),
	}
}

// optionalStringPointer maps an unset description onto a nil pointer.
func optionalStringPointer(model *policyModel) *string {
	if !helpers.IsConfiguredValue(model.Description) {
		return nil
	}
	value := model.Description.ValueString()
	return &value
}

// settingsPayload returns the settings body to send. The attribute is required and validated as a
// JSON object before this runs, so an unset value can only mean an empty settings object.
func settingsPayload(model *policyModel) json.RawMessage {
	if !helpers.IsConfiguredValue(model.SettingsJSON) {
		return json.RawMessage("{}")
	}
	return json.RawMessage(model.SettingsJSON.ValueString())
}
