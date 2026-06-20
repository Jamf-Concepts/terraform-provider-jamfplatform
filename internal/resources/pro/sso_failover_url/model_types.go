// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_failover_url

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SsoFailoverURLResourceModel is the Terraform model for
// jamfplatform_pro_sso_failover_url. RegenerationTrigger is a user-controlled
// string: changing it forces an Update which calls POST /generate to mint a
// fresh failover URL.
type SsoFailoverURLResourceModel struct {
	ID                  types.String           `tfsdk:"id"`
	RegenerationTrigger types.String           `tfsdk:"regeneration_trigger"`
	FailoverURL         types.String           `tfsdk:"failover_url"`
	GenerationTime      types.Int64            `tfsdk:"generation_time"`
	GenerationTimeUTC   types.String           `tfsdk:"generation_time_utc"`
	Timeouts            resourceTimeouts.Value `tfsdk:"timeouts"`
}

// SsoFailoverURLDataSourceModel is the read-only mirror.
type SsoFailoverURLDataSourceModel struct {
	ID                types.String             `tfsdk:"id"`
	FailoverURL       types.String             `tfsdk:"failover_url"`
	GenerationTime    types.Int64              `tfsdk:"generation_time"`
	GenerationTimeUTC types.String             `tfsdk:"generation_time_utc"`
	Timeouts          datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// ssoFailoverURLIdentityModel is the identity object used on import.
type ssoFailoverURLIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
