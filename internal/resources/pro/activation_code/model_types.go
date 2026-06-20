// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package activation_code

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ActivationCodeResourceModel represents the Terraform resource model for the Jamf Pro
// activation code singleton. Both fields round-trip through a single ProClassic
// GET/PUT — the GET returns the code in clear, so normal drift detection applies.
type ActivationCodeResourceModel struct {
	ID               types.String           `tfsdk:"id"`
	OrganizationName types.String           `tfsdk:"organization_name"`
	Code             types.String           `tfsdk:"code"`
	Timeouts         resourceTimeouts.Value `tfsdk:"timeouts"`
}

// ActivationCodeDataSourceModel represents the Terraform data source model.
type ActivationCodeDataSourceModel struct {
	ID               types.String             `tfsdk:"id"`
	OrganizationName types.String             `tfsdk:"organization_name"`
	Code             types.String             `tfsdk:"code"`
	Timeouts         datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// activationCodeIdentityModel represents the identity object used on import.
type activationCodeIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
