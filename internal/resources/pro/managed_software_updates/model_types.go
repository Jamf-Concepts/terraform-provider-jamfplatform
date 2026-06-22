// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package managed_software_updates

import (
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ManagedSoftwareUpdateResourceModel represents the Terraform resource model for the Jamf
// Pro Managed Software Updates feature. `enabled` is the only writable attribute (it maps to
// the SDK `toggle` field); the four sub-enables are server-managed and surfaced read-only.
type ManagedSoftwareUpdateResourceModel struct {
	ID                           types.String           `tfsdk:"id"`
	Enabled                      types.Bool             `tfsdk:"enabled"`
	DssEnabled                   types.Bool             `tfsdk:"dss_enabled"`
	RecipeEnabled                types.Bool             `tfsdk:"recipe_enabled"`
	ForceInstallLocalDateEnabled types.Bool             `tfsdk:"force_install_local_date_enabled"`
	CustomVersionEnabled         types.Bool             `tfsdk:"custom_version_enabled"`
	Timeouts                     resourceTimeouts.Value `tfsdk:"timeouts"`
}

// managedSoftwareUpdateIdentityModel represents the identity object used on import.
type managedSoftwareUpdateIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
