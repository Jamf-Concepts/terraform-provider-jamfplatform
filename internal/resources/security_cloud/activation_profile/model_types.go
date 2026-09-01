// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package activation_profile

import (
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ActivationProfileResourceModel represents the Terraform resource model for a
// Jamf Security Cloud activation profile.
type ActivationProfileResourceModel struct {
	ID           types.String           `tfsdk:"id"`
	Name         types.String           `tfsdk:"name"`
	Platforms    types.Set              `tfsdk:"platforms"`
	Capabilities *CapabilitiesModel     `tfsdk:"capabilities"`
	DeviceGroup  types.String           `tfsdk:"device_group_id"`
	Paused       types.Bool             `tfsdk:"paused"`
	Timeouts     resourceTimeouts.Value `tfsdk:"timeouts"`
}

// CapabilitiesModel represents the service capabilities enabled on an activation
// profile.
type CapabilitiesModel struct {
	ContentControls types.Bool   `tfsdk:"content_controls"`
	NetworkSecurity types.Bool   `tfsdk:"network_security"`
	Note            types.String `tfsdk:"note"`
}
