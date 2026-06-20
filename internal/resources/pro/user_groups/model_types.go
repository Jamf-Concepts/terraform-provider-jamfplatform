// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package user_groups

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/filters"
)

// UserGroupsDataSourceModel represents the Terraform data source model for user
// group searches.
type UserGroupsDataSourceModel struct {
	ID         types.String                      `tfsdk:"id"`
	UserGroups []UserGroupsDataSourceResultModel `tfsdk:"user_groups"`
	Filter     *filters.ClassicFilterModel       `tfsdk:"filter"`
	Timeouts   datasourceTimeouts.Value          `tfsdk:"timeouts"`
}

// UserGroupsDataSourceResultModel represents a single user group in the
// search results. Only the fields the classic list endpoint returns are
// exposed — id, name, group_type, notify_on_membership_change. Per-item
// criteria, users, and site require a singular
// `jamfplatform_pro_user_group` lookup.
type UserGroupsDataSourceResultModel struct {
	ID                       types.String `tfsdk:"id"`
	Name                     types.String `tfsdk:"name"`
	GroupType                types.String `tfsdk:"group_type"`
	NotifyOnMembershipChange types.Bool   `tfsdk:"notify_on_membership_change"`
}
