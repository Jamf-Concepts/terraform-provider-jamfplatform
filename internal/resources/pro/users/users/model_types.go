// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package users

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/filters"
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// UsersDataSourceModel represents the Terraform data source model for inventory
// user searches.
type UsersDataSourceModel struct {
	ID       types.String                 `tfsdk:"id"`
	Users    []UsersDataSourceResultModel `tfsdk:"users"`
	Filters  []filters.FilterModel        `tfsdk:"filter"`
	Timeouts datasourceTimeouts.Value     `tfsdk:"timeouts"`
}

// UsersDataSourceResultModel represents a single user in the search results.
type UsersDataSourceResultModel struct {
	ID                   types.String `tfsdk:"id"`
	Username             types.String `tfsdk:"username"`
	FullName             types.String `tfsdk:"full_name"`
	EmailAddress         types.String `tfsdk:"email_address"`
	PhoneNumber          types.String `tfsdk:"phone_number"`
	Position             types.String `tfsdk:"position"`
	ManagedAppleID       types.String `tfsdk:"managed_apple_id"`
	EnableCustomPhotoURL types.Bool   `tfsdk:"enable_custom_photo_url"`
	CustomPhotoURL       types.String `tfsdk:"custom_photo_url"`
}
