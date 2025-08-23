package blueprint

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// blueprintDataSourceModel defines the data structure for the blueprint data source.
type blueprintDataSourceModel struct {
	ID              types.String   `tfsdk:"id"`
	Name            types.String   `tfsdk:"name"`
	BlueprintID     types.String   `tfsdk:"blueprint_id"`
	Description     types.String   `tfsdk:"description"`
	Created         types.String   `tfsdk:"created"`
	Updated         types.String   `tfsdk:"updated"`
	DeploymentState types.String   `tfsdk:"deployment_state"`
	DeviceGroups    []types.String `tfsdk:"device_groups"`
	Steps           []stepModel    `tfsdk:"steps"`
}

// stepModel defines the data structure for a blueprint step.
type stepModel struct {
	Name       types.String     `tfsdk:"name"`
	Components []componentModel `tfsdk:"components"`
}

// componentModel defines the data structure for a blueprint component.
type componentModel struct {
	Identifier    types.String `tfsdk:"identifier"`
	Configuration types.String `tfsdk:"configuration"`
}
