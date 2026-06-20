// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package jamf_parent_settings

import (
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// JamfParentSettingsResourceModel is the Terraform model for
// jamfplatform_pro_jamf_parent_settings.
//
// The resource is a singleton mapping the Jamf Pro Jamf Parent settings page
// (Settings → Jamf apps → Jamf Parent): the enable toggle, the student device
// group, the per-day restricted times, the time zone, the passcode-clear and
// revoke-on-wipe toggles, and the Safelisted Apps tab.
//
// safelisted_apps is Optional+Computed (a nested collection with a Computed
// half), so it must be a framework types.Set — not a Go typed slice — to
// decode the Unknown plan value the framework reports before the server value
// is known. restricted_times is Required and user-authored, but stays a
// framework types.Map in the model (framework nested collections decode via
// ElementsAs in the input builder).
type JamfParentSettingsResourceModel struct {
	ID types.String `tfsdk:"id"`

	Enabled                 types.Bool   `tfsdk:"enabled"`
	Timezone                types.String `tfsdk:"timezone"`
	DeviceGroupID           types.Int64  `tfsdk:"device_group_id"`
	RestrictedTimes         types.Map    `tfsdk:"restricted_times"`
	AllowClearPasscode      types.Bool   `tfsdk:"allow_clear_passcode"`
	RevokeOnWipeAndReEnroll types.Bool   `tfsdk:"revoke_on_wipe_and_re_enroll"`
	SafelistedApps          types.Set    `tfsdk:"safelisted_apps"`

	Timeouts resourceTimeouts.Value `tfsdk:"timeouts"`
}

// restrictedTimeModel is the element model for one restricted_times map entry
// (one day of the week).
type restrictedTimeModel struct {
	BeginTime types.String `tfsdk:"begin_time"`
	EndTime   types.String `tfsdk:"end_time"`
}

// safelistedAppModel is the element model for one safelisted_apps entry.
//
// The 2-field sub-schema is deliberately duplicated from jamf_teacher_settings
// — two consumers is below the 3-consumer extraction bar (STYLE_GUIDE §Shared
// abstractions).
type safelistedAppModel struct {
	Name     types.String `tfsdk:"name"`
	BundleID types.String `tfsdk:"bundle_id"`
}

// jamfParentSettingsIdentityModel is the identity object used on import.
type jamfParentSettingsIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
