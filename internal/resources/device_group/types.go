// Copyright 2025 Jamf Software LLC.

package device_group

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DeviceGroupResource implements the Terraform resource for Jamf device groups.
type DeviceGroupResource struct {
	client *client.Client
}

// DeviceGroupResourceModel represents the Terraform resource model for a Jamf device group.
type DeviceGroupResourceModel struct {
	ID          types.String               `tfsdk:"id"`
	Name        types.String               `tfsdk:"name"`
	Description types.String               `tfsdk:"description"`
	DeviceType  types.String               `tfsdk:"device_type"`
	GroupType   types.String               `tfsdk:"group_type"`
	Criteria    []DeviceGroupCriteriaModel `tfsdk:"criteria"`
	Members     types.Set                  `tfsdk:"members"`
	MemberCount types.Int64                `tfsdk:"member_count"`
}

// DeviceGroupCriteriaModel represents a smart group criterion definition.
type DeviceGroupCriteriaModel struct {
	Order                 types.Int64  `tfsdk:"order"`
	AttributeName         types.String `tfsdk:"criteria"`
	Operator              types.String `tfsdk:"operator"`
	AttributeValue        types.String `tfsdk:"value"`
	JoinType              types.String `tfsdk:"and_or"`
	HasOpeningParenthesis types.Bool   `tfsdk:"has_opening_parenthesis"`
	HasClosingParenthesis types.Bool   `tfsdk:"has_closing_parenthesis"`
}

// DeviceGroupDataSource implements the Terraform data source for Jamf device groups.
type DeviceGroupDataSource struct {
	client *client.Client
}

// DeviceGroupDataSourceModel represents the Terraform data source model for a Jamf device group.
type DeviceGroupDataSourceModel struct {
	ID          types.String               `tfsdk:"id"`
	Name        types.String               `tfsdk:"name"`
	Description types.String               `tfsdk:"description"`
	DeviceType  types.String               `tfsdk:"device_type"`
	GroupType   types.String               `tfsdk:"group_type"`
	Criteria    []DeviceGroupCriteriaModel `tfsdk:"criteria"`
	Members     types.Set                  `tfsdk:"members"`
	MemberCount types.Int64                `tfsdk:"member_count"`
}
