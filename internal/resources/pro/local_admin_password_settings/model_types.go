// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package local_admin_password_settings

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// LocalAdminPasswordSettingsResourceModel is the Terraform model for
// jamfplatform_pro_local_admin_password_settings.
//
// The resource is a singleton mapping the Jamf Pro LAPS section of the Security
// page (Settings → Computer Management → Security → "Password settings for
// managed local administrator accounts"). It exposes the three UI controls. The
// two interval controls are dropdowns of fixed presets; the provider translates
// those preset labels to and from the underlying durations at the input/state
// boundary, and the automatic-rotation "Never" option collapses the separate
// enable flag the UI hides behind the dropdown.
type LocalAdminPasswordSettingsResourceModel struct {
	ID types.String `tfsdk:"id"`

	LapsForPrestageAccountsEnabled types.Bool   `tfsdk:"laps_for_prestage_accounts_enabled"`
	RotationInterval               types.String `tfsdk:"rotation_interval"`
	RotationAfterViewingInterval   types.String `tfsdk:"rotation_after_viewing_interval"`

	Timeouts resourceTimeouts.Value `tfsdk:"timeouts"`
}

// LocalAdminPasswordSettingsDataSourceModel mirrors the resource model with every
// attribute Computed.
type LocalAdminPasswordSettingsDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	LapsForPrestageAccountsEnabled types.Bool   `tfsdk:"laps_for_prestage_accounts_enabled"`
	RotationInterval               types.String `tfsdk:"rotation_interval"`
	RotationAfterViewingInterval   types.String `tfsdk:"rotation_after_viewing_interval"`

	Timeouts datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// localAdminPasswordSettingsIdentityModel is the identity object used on import.
type localAdminPasswordSettingsIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
