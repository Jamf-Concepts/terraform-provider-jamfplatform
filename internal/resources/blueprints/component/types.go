package component

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// componentDataSourceModel defines the data structure for the component data source.
type componentDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Identifier  types.String `tfsdk:"identifier"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Meta        types.Object `tfsdk:"meta"`
}
