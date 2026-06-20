// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package service_discovery_enrollment

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ServiceDiscoveryEnrollmentResourceModel is the Terraform resource model for the
// service-discovery well-known settings singleton. well_known_setting carries a
// Computed echo sub-attribute (org_name), so the collection is a types.List, never a
// Go typed slice — a Computed value is Unknown at plan time and a typed slice cannot
// carry it.
type ServiceDiscoveryEnrollmentResourceModel struct {
	ID               types.String           `tfsdk:"id"`
	WellKnownSetting types.List             `tfsdk:"well_known_setting"`
	Timeouts         resourceTimeouts.Value `tfsdk:"timeouts"`
}

// ServiceDiscoveryEnrollmentDataSourceModel is the Terraform data source model. The
// data source reads every row Jamf Pro returns (discovery aid for the available
// server_uuids), so well_known_setting is Computed-only.
type ServiceDiscoveryEnrollmentDataSourceModel struct {
	ID               types.String             `tfsdk:"id"`
	WellKnownSetting types.List               `tfsdk:"well_known_setting"`
	Timeouts         datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// wellKnownSettingModel is one element of well_known_setting. server_uuid and
// enrollment_type are user-authored; org_name is a server-derived read-only echo
// (Computed).
type wellKnownSettingModel struct {
	ServerUUID     types.String `tfsdk:"server_uuid"`
	EnrollmentType types.String `tfsdk:"enrollment_type"`
	OrgName        types.String `tfsdk:"org_name"`
}

// serviceDiscoveryEnrollmentIdentityModel represents the identity object used on
// import.
type serviceDiscoveryEnrollmentIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
