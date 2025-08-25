// Copyright 2025 Jamf Software LLC.

package components

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ComponentsDataSource defines the data source for blueprint components.
type ComponentsDataSource struct {
	client *client.Client
}

// ComponentsDataSourceModel defines the data structure for the components data source.
type ComponentsDataSourceModel struct {
	Components []ComponentListModel `tfsdk:"components"`
}

// ComponentListModel defines the data structure for a component in the list.
type ComponentListModel struct {
	Identifier  types.String `tfsdk:"identifier"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	SupportedOs types.Map    `tfsdk:"supported_os"`
}
