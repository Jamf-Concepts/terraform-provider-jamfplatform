// Copyright 2025 Jamf Software LLC.

package blueprint

import (
	"context"
	"fmt"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BlueprintResource{}
var _ resource.ResourceWithImportState = &BlueprintResource{}

// NewBlueprintResource returns a new instance of BlueprintResource.
func NewBlueprintResource() resource.Resource {
	return &BlueprintResource{}
}

// Metadata sets the resource type name for the Terraform provider.
func (r *BlueprintResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blueprints_blueprint"
}

// Schema returns the Terraform schema for the blueprint resource.
func (r *BlueprintResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Resource schema for creating and managing Jamf Blueprints. Blueprints are automatically deployed after successful creation or update.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the blueprint.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Blueprint name.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Blueprint description.",
				Optional:    true,
			},
			"device_groups": schema.ListAttribute{
				Description: "List of device group IDs to target.",
				Required:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"created": schema.StringAttribute{
				Description: "Creation timestamp.",
				Computed:    true,
			},
			"updated": schema.StringAttribute{
				Description: "Last updated timestamp.",
				Computed:    true,
			},
			"deployment_state": schema.StringAttribute{
				Description: "Current deployment state.",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"component": schema.ListNestedBlock{
				Description: "Component configuration. All components will be placed in a 'Declaration group' step automatically.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"identifier": schema.StringAttribute{
							Description: "Component identifier (e.g., com.jamf.ddm.disk-management).",
							Required:    true,
						},
						"configuration": schema.MapAttribute{
							Description: "Component configuration as key-value pairs. Each component has its own unique configuration options.",
							Optional:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

// Configure sets up the API client for the resource from the provider configuration.
func (r *BlueprintResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// ImportState handles the import of existing Blueprint resources.
func (r *BlueprintResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
