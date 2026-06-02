// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_internal_source

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/patch/availabletitles"
)

// PatchInternalSourceDataSourceModel represents the Terraform data source model
// for a Jamf Pro patch internal source. Internal sources are Jamf-managed (the
// built-in "Jamf" source) and not user-creatable, so there is no resource model.
// Either id or name must be supplied (enforced by ExactlyOneOf at config
// validation); the remaining attributes are populated from the SDK response.
type PatchInternalSourceDataSourceModel struct {
	ID              types.String             `tfsdk:"id"`
	Name            types.String             `tfsdk:"name"`
	Enabled         types.Bool               `tfsdk:"enabled"`
	Endpoint        types.String             `tfsdk:"endpoint"`
	AvailableTitles []availabletitles.Model  `tfsdk:"available_titles"`
	Timeouts        datasourceTimeouts.Value `tfsdk:"timeouts"`
}
