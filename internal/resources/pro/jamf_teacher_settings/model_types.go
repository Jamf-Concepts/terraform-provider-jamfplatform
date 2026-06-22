// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package jamf_teacher_settings

import (
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// JamfTeacherSettingsResourceModel is the Terraform model for
// jamfplatform_pro_jamf_teacher_settings.
//
// The resource is a singleton mapping the Jamf Pro Jamf Teacher settings page
// (Settings → Jamf apps → Jamf Teacher): the enable toggle, the time zone, the
// restriction clearing time and length limits, and the Safelisted Apps tab.
//
// safelisted_apps is Optional+Computed (a nested collection with a Computed
// half), so it must be a framework types.Set — not a Go typed slice — to decode
// the Unknown plan value the framework reports before the server value is
// known.
type JamfTeacherSettingsResourceModel struct {
	ID types.String `tfsdk:"id"`

	Enabled                       types.Bool   `tfsdk:"enabled"`
	Timezone                      types.String `tfsdk:"timezone"`
	RestrictionsEndTime           types.String `tfsdk:"restrictions_end_time"`
	MaximumRestrictionTimeSeconds types.Int64  `tfsdk:"maximum_restriction_time_seconds"`
	SafelistedApps                types.Set    `tfsdk:"safelisted_apps"`

	Timeouts resourceTimeouts.Value `tfsdk:"timeouts"`
}

// safelistedAppModel is the element model for one safelisted_apps entry.
type safelistedAppModel struct {
	Name     types.String `tfsdk:"name"`
	BundleID types.String `tfsdk:"bundle_id"`
}

// jamfTeacherSettingsIdentityModel is the identity object used on import.
type jamfTeacherSettingsIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
