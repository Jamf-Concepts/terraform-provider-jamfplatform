package blueprint

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// BlueprintResource implements the Terraform resource for Jamf Blueprint.
type BlueprintResource struct {
	client *client.Client
}

// BlueprintResourceModel represents the Terraform resource model for a Jamf Blueprint.
type BlueprintResourceModel struct {
	ID              types.String     `tfsdk:"id"`
	Name            types.String     `tfsdk:"name"`
	Description     types.String     `tfsdk:"description"`
	DeviceGroups    []types.String   `tfsdk:"device_groups"`
	Components      []ComponentModel `tfsdk:"component"`
	Created         types.String     `tfsdk:"created"`
	Updated         types.String     `tfsdk:"updated"`
	DeploymentState types.String     `tfsdk:"deployment_state"`
}

// BlueprintDataSource implements the Terraform data source for Jamf Blueprint.
type BlueprintDataSource struct {
	client *client.Client
}

// BlueprintDataSourceModel defines the data structure for the blueprint data source.
type BlueprintDataSourceModel struct {
	ID              types.String     `tfsdk:"id"`
	Name            types.String     `tfsdk:"name"`
	BlueprintID     types.String     `tfsdk:"blueprint_id"`
	Description     types.String     `tfsdk:"description"`
	Created         types.String     `tfsdk:"created"`
	Updated         types.String     `tfsdk:"updated"`
	DeploymentState types.String     `tfsdk:"deployment_state"`
	DeviceGroups    []types.String   `tfsdk:"device_groups"`
	Components      []ComponentModel `tfsdk:"component"`
}

// ComponentModel defines the data structure for a blueprint component.
type ComponentModel struct {
	Identifier    types.String `tfsdk:"identifier"`
	Configuration types.Map    `tfsdk:"configuration"`
}
