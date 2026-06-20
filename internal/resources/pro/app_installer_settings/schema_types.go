// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_installer_settings

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// appInstallerSettingsTimeoutAttributeTypes defines the timeout attribute types for the resource.
var appInstallerSettingsTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// deploymentSettingsAttrTypes is the types map for the deployment_settings
// SingleNestedAttribute. Required for types.Object encoding/decoding when the
// block is Optional+Computed (Unknown during plan until USFU resolves it).
var deploymentSettingsAttrTypes = map[string]attr.Type{
	"batch_size":       types.Int64Type,
	"batch_frequency":  types.Int64Type,
	"days":             types.SetType{ElemType: types.StringType},
	"server_time_from": types.StringType,
	"server_time_to":   types.StringType,
}

// endUserExperienceAttrTypes is the types map for the end_user_experience
// SingleNestedAttribute.
var endUserExperienceAttrTypes = map[string]attr.Type{
	"notification_frequency":  types.Int64Type,
	"notification_message":    types.StringType,
	"update_deadline":         types.Int64Type,
	"force_quit_message":      types.StringType,
	"force_quit_grace_period": types.Int64Type,
	"update_complete_message": types.StringType,
	"relaunch":                types.BoolType,
	"suppress":                types.BoolType,
}
