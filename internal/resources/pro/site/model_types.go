// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package site

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/filters"
)

// SiteResourceModel represents the Terraform resource model for a Jamf Pro site.
type SiteResourceModel struct {
	ID       types.String           `tfsdk:"id"`
	Name     types.String           `tfsdk:"name"`
	Timeouts resourceTimeouts.Value `tfsdk:"timeouts"`
}

// SiteDataSourceModel represents the Terraform data source model for a Jamf Pro site.
// Either id or name must be supplied (enforced by ExactlyOneOf at config validation).
type SiteDataSourceModel struct {
	ID       types.String             `tfsdk:"id"`
	Name     types.String             `tfsdk:"name"`
	Timeouts datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// siteIdentityModel represents the identity object for site resources and list results.
type siteIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

// SiteListResourceModel represents the config model for site list queries.
// Classic has no RSQL — the filter shape is the shared client-side substring block.
type SiteListResourceModel struct {
	Filter *filters.ClassicFilterModel `tfsdk:"filter"`
}
