// Copyright 2025 Jamf Software LLC.

package blueprint

import (
	"context"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewBlueprintResource returns a new instance of BlueprintResource.
func NewBlueprintResource() resource.Resource {
	return &BlueprintResource{}
}

// Metadata sets the resource type name for the Terraform provider.
func (r *BlueprintResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blueprints_blueprint"
}

// Configure sets up the API client for the resource from the provider configuration.
func (r *BlueprintResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	apiClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected ProviderData type",
			"Expected *client.Client, got something else.",
		)
		return
	}
	r.client = apiClient
}

// Schema returns the Terraform schema for the blueprint resource.
func (r *BlueprintResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Resource schema for creating and managing Jamf Blueprints.",
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
							Description: "Component configuration as key-value pairs. The provider will automatically convert this to the proper JSON format.",
							Optional:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}
