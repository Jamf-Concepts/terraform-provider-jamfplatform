package component

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// componentDataSource defines the data source for blueprint components.
type componentDataSource struct {
	client *client.Client
}

// componentDataSourceModel defines the data structure for the component data source.
type componentDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Identifier  types.String `tfsdk:"identifier"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	SupportedOs types.Map    `tfsdk:"supported_os"`
}
