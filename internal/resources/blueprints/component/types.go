package component

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ComponentDataSource defines the data source implementation.
type ComponentDataSource struct {
	client *client.Client
}

// ComponentDataSourceModel defines the data source data model.
type ComponentDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Identifier  types.String `tfsdk:"identifier"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	SupportedOs types.Map    `tfsdk:"supported_os"`
}
