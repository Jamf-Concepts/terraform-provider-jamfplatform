// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package user_group

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// UserGroupResourceModel represents the Terraform resource model for a Jamf Pro user group.
type UserGroupResourceModel struct {
	ID                       types.String              `tfsdk:"id"`
	Name                     types.String              `tfsdk:"name"`
	GroupType                types.String              `tfsdk:"group_type"`
	NotifyOnMembershipChange types.Bool                `tfsdk:"notify_on_membership_change"`
	SiteID                   types.String              `tfsdk:"site_id"`
	SiteName                 types.String              `tfsdk:"site_name"`
	Criteria                 []UserGroupCriterionModel `tfsdk:"criteria"`
	Members                  types.Set                 `tfsdk:"members"`
	MemberCount              types.Int64               `tfsdk:"member_count"`
	Timeouts                 resourceTimeouts.Value    `tfsdk:"timeouts"`
}

// UserGroupCriterionModel represents a single smart-group criterion.
type UserGroupCriterionModel struct {
	Priority              types.Int64  `tfsdk:"priority"`
	Name                  types.String `tfsdk:"name"`
	SearchType            types.String `tfsdk:"search_type"`
	Value                 types.String `tfsdk:"value"`
	AndOr                 types.String `tfsdk:"and_or"`
	HasOpeningParenthesis types.Bool   `tfsdk:"has_opening_parenthesis"`
	HasClosingParenthesis types.Bool   `tfsdk:"has_closing_parenthesis"`
}

// UserGroupDataSourceModel represents the Terraform data source model for a Jamf Pro user group.
// Either id or name must be supplied (enforced by ExactlyOneOf at config validation).
type UserGroupDataSourceModel struct {
	ID                       types.String              `tfsdk:"id"`
	Name                     types.String              `tfsdk:"name"`
	GroupType                types.String              `tfsdk:"group_type"`
	NotifyOnMembershipChange types.Bool                `tfsdk:"notify_on_membership_change"`
	SiteID                   types.String              `tfsdk:"site_id"`
	SiteName                 types.String              `tfsdk:"site_name"`
	Criteria                 []UserGroupCriterionModel `tfsdk:"criteria"`
	Users                    []UserGroupUserModel      `tfsdk:"users"`
	MemberCount              types.Int64               `tfsdk:"member_count"`
	Timeouts                 datasourceTimeouts.Value  `tfsdk:"timeouts"`
}

// UserGroupUserModel is the Computed user representation surfaced on the data source.
type UserGroupUserModel struct {
	ID           types.String `tfsdk:"id"`
	Username     types.String `tfsdk:"username"`
	FullName     types.String `tfsdk:"full_name"`
	PhoneNumber  types.String `tfsdk:"phone_number"`
	EmailAddress types.String `tfsdk:"email_address"`
}

// userGroupIdentityModel represents the identity object for user group resources and list results.
type userGroupIdentityModel struct {
	ID types.String `tfsdk:"id"`
}
