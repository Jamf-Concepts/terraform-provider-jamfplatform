// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package self_service_branding_image

import (
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SelfServiceBrandingImageResourceModel represents the Terraform resource model
// for a Self Service branding image upload.
type SelfServiceBrandingImageResourceModel struct {
	ID              types.String           `tfsdk:"id"`
	ImageFileSource types.String           `tfsdk:"image_file_source"`
	SourceHash      types.String           `tfsdk:"source_hash"`
	URL             types.String           `tfsdk:"url"`
	Timeouts        resourceTimeouts.Value `tfsdk:"timeouts"`
}
