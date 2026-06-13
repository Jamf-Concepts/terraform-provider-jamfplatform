// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package return_to_service

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ReturnToServiceResourceModel is the Terraform resource model for a Jamf Pro
// Return to Service configuration. The wire record carries exactly three fields
// — id, displayName, wifiProfileId — so the model has no derived/echo siblings;
// id is the only server-computed attribute.
type ReturnToServiceResourceModel struct {
	ID            types.String           `tfsdk:"id"`
	DisplayName   types.String           `tfsdk:"display_name"`
	WifiProfileID types.String           `tfsdk:"wifi_profile_id"`
	Timeouts      resourceTimeouts.Value `tfsdk:"timeouts"`
}

// ReturnToServiceDataSourceModel is the Terraform data source model. Either id
// or name must be supplied (enforced by ExactlyOneOf at config validation).
type ReturnToServiceDataSourceModel struct {
	ID            types.String             `tfsdk:"id"`
	DisplayName   types.String             `tfsdk:"display_name"`
	WifiProfileID types.String             `tfsdk:"wifi_profile_id"`
	Timeouts      datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// returnToServiceIdentityModel is the identity object for the resource and list
// results.
type returnToServiceIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
