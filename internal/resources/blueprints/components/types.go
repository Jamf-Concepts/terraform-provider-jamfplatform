// Copyright 2025 Jamf Software LLC.

package components

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// componentsDataSourceModel defines the data structure for the components data source.
type componentsDataSourceModel struct {
	Components []componentListModel `tfsdk:"components"`
}

// componentListModel defines the data structure for a component in the list.
type componentListModel struct {
	Identifier  types.String `tfsdk:"identifier"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Meta        types.Object `tfsdk:"meta"`
}
