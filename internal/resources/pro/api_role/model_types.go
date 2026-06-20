// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_role

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/filters"
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ApiRoleResourceModel represents the Terraform resource model for a Jamf Pro API role.
type ApiRoleResourceModel struct {
	ID          types.String           `tfsdk:"id"`
	DisplayName types.String           `tfsdk:"display_name"`
	Privileges  types.Set              `tfsdk:"privileges"`
	Timeouts    resourceTimeouts.Value `tfsdk:"timeouts"`
}

// ApiRoleDataSourceModel represents the Terraform data source model for a Jamf Pro API role.
type ApiRoleDataSourceModel struct {
	ID          types.String             `tfsdk:"id"`
	DisplayName types.String             `tfsdk:"display_name"`
	Privileges  types.Set                `tfsdk:"privileges"`
	Timeouts    datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// apiRoleIdentityModel represents the identity object for API role resources and list results.
type apiRoleIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

// ApiRoleListResourceModel represents the config model for API role list queries.
type ApiRoleListResourceModel struct {
	Filters []filters.FilterModel `tfsdk:"filter"`
}
