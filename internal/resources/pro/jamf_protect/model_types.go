// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package jamf_protect

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/filters"
)

// JamfProtectResourceModel is the Terraform model for the Jamf Protect
// registration singleton.
//
// `password` is `WriteOnly`: the user-supplied plaintext is read from
// req.Config on writes and never persisted in Terraform state (Jamf Pro also
// never returns it — the spec marks it write-only). `password_wo_version` is
// the rotation companion: bumping it is the only signal that the stored
// password changed, and it triggers an in-place re-register (POST) carrying
// the current `password`.
//
// `client_id` is sent as `clientId` on register and echoed back as
// `apiClientId` on reads — the state builder maps the echo back so an
// out-of-band re-registration with different credentials surfaces as drift.
type JamfProtectResourceModel struct {
	ID types.String `tfsdk:"id"`

	// Registration credentials. Any change (api_url, client_id, or a
	// password_wo_version bump) triggers an in-place re-register POST —
	// the server overwrites the existing registration without unregistering
	// first, and a failed credential check leaves it intact.
	APIURL            types.String `tfsdk:"api_url"`
	ClientID          types.String `tfsdk:"client_id"`
	Password          types.String `tfsdk:"password"`
	PasswordWoVersion types.Int64  `tfsdk:"password_wo_version"`

	// AutoInstall is the only PUT-mutable field. Server default false.
	AutoInstall types.Bool `tfsdk:"auto_install"`

	// Server-derived echoes — plain Computed (no UseStateForUnknown:
	// registration_id is re-minted on every re-register, and the sync pair
	// is volatile).
	RegistrationID   types.String `tfsdk:"registration_id"`
	APIClientName    types.String `tfsdk:"api_client_name"`
	PlatformPlanSync types.Bool   `tfsdk:"platform_plan_sync"`
	LastSyncTime     types.String `tfsdk:"last_sync_time"`
	SyncStatus       types.String `tfsdk:"sync_status"`

	Timeouts resourceTimeouts.Value `tfsdk:"timeouts"`
}

// jamfProtectIdentityModel is the import identity for the singleton.
type jamfProtectIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

// JamfProtectPlansDataSourceModel is the model for the plural plans catalog
// data source. The catalog persists after unregistering, so the data source
// works (and may return stale rows) even when the tenant is not registered.
type JamfProtectPlansDataSourceModel struct {
	ID       types.String             `tfsdk:"id"`
	Filters  []filters.FilterModel    `tfsdk:"filter"`
	Sort     types.List               `tfsdk:"sort"`
	Plans    []JamfProtectPlanModel   `tfsdk:"plans"`
	Timeouts datasourceTimeouts.Value `tfsdk:"timeouts"`
}

// JamfProtectPlanModel is one synced Jamf Protect plan row.
type JamfProtectPlanModel struct {
	ID               types.String `tfsdk:"id"`
	UUID             types.String `tfsdk:"uuid"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	ProfileID        types.Int64  `tfsdk:"profile_id"`
	ProfileName      types.String `tfsdk:"profile_name"`
	ProfileVersion   types.Int64  `tfsdk:"profile_version"`
	ScopeDescription types.String `tfsdk:"scope_description"`
	SiteID           types.String `tfsdk:"site_id"`
}
