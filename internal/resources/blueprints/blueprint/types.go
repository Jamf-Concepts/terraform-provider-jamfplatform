package blueprint

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// BlueprintResource implements the Terraform resource for Jamf Blueprint.
type BlueprintResource struct {
	client *client.Client
}

// blueprintResourceModel represents the Terraform resource model for a Jamf Blueprint.
type blueprintResourceModel struct {
	ID              types.String     `tfsdk:"id"`
	Name            types.String     `tfsdk:"name"`
	Description     types.String     `tfsdk:"description"`
	DeviceGroups    []types.String   `tfsdk:"device_groups"`
	Components      []componentModel `tfsdk:"component"`
	Created         types.String     `tfsdk:"created"`
	Updated         types.String     `tfsdk:"updated"`
	DeploymentState types.String     `tfsdk:"deployment_state"`
}

// blueprintDataSource implements the Terraform data source for Jamf Blueprint.
type blueprintDataSource struct {
	client *client.Client
}

// blueprintDataSourceModel defines the data structure for the blueprint data source.
type blueprintDataSourceModel struct {
	ID              types.String     `tfsdk:"id"`
	Name            types.String     `tfsdk:"name"`
	BlueprintID     types.String     `tfsdk:"blueprint_id"`
	Description     types.String     `tfsdk:"description"`
	Created         types.String     `tfsdk:"created"`
	Updated         types.String     `tfsdk:"updated"`
	DeploymentState types.String     `tfsdk:"deployment_state"`
	DeviceGroups    []types.String   `tfsdk:"device_groups"`
	Components      []componentModel `tfsdk:"component"`
}

// componentModel defines the data structure for a blueprint component.
type componentModel struct {
	Identifier    types.String `tfsdk:"identifier"`
	Configuration types.Map    `tfsdk:"configuration"`
}
