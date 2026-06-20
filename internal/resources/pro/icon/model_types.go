// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package icon

import (
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// IconResourceModel represents the Terraform resource model for a Jamf Pro icon.
type IconResourceModel struct {
	ID             types.String           `tfsdk:"id"`
	IconFileSource types.String           `tfsdk:"icon_file_source"`
	SourceHash     types.String           `tfsdk:"source_hash"`
	URL            types.String           `tfsdk:"url"`
	Timeouts       resourceTimeouts.Value `tfsdk:"timeouts"`
}
