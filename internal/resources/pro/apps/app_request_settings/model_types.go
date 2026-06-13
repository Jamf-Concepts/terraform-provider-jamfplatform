// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_request_settings

import (
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AppRequestSettingsResourceModel represents the Terraform resource model for the Jamf Pro
// App Request settings singleton.
type AppRequestSettingsResourceModel struct {
	ID                   types.String           `tfsdk:"id"`
	Enabled              types.Bool             `tfsdk:"enabled"`
	AppStoreLocale       types.String           `tfsdk:"app_store_locale"`
	ApproverEmails       types.Set              `tfsdk:"approver_emails"`
	RequesterUserGroupID types.Int64            `tfsdk:"requester_user_group_id"`
	Timeouts             resourceTimeouts.Value `tfsdk:"timeouts"`
}

// appRequestSettingsIdentityModel represents the identity object used on import.
type appRequestSettingsIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

// appRequestSettingsTimeoutAttributeTypes defines the timeout attribute types for the App
// Request settings resource operations.
var appRequestSettingsTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}
