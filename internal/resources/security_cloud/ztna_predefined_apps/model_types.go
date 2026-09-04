// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_predefined_apps

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// PredefinedAppsDataSourceModel represents the Terraform data source model for the
// Jamf-curated Zero Trust Network Access app templates.
type PredefinedAppsDataSourceModel struct {
	ID             types.String               `tfsdk:"id"`
	PredefinedApps []PredefinedAppResultModel `tfsdk:"predefined_apps"`
	Timeouts       datasourceTimeouts.Value   `tfsdk:"timeouts"`
}

// PredefinedAppResultModel represents a single predefined app template in the
// results. Hostnames is a types.List because a Computed nested collection must be,
// per STYLE_GUIDE §Sets vs Lists.
type PredefinedAppResultModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Hostnames types.List   `tfsdk:"hostnames"`
}
