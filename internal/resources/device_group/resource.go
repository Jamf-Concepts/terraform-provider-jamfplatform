// Copyright 2025 Jamf Software LLC.

package device_group

import (
	"context"
	"fmt"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &DeviceGroupResource{}
var _ resource.ResourceWithImportState = &DeviceGroupResource{}

// NewDeviceGroupResource returns a new instance of DeviceGroupResource.
func NewDeviceGroupResource() resource.Resource {
	return &DeviceGroupResource{}
}

// Metadata sets the resource type name for the Terraform provider.
func (r *DeviceGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_group"
}

// Schema returns the Terraform schema for the device group resource.
func (r *DeviceGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Jamf device groups (static or smart) via the Platform API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier assigned by the API.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Display name for the device group.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Description: "Optional description for the device group.",
				Optional:    true,
			},
			"device_type": schema.StringAttribute{
				Description: "Jamf device type. Changes require resource replacement. Valid values are 'computer' or 'mobile'.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("computer", "mobile"),
				},
			},
			"group_type": schema.StringAttribute{
				Description: "Group implementation type. Changes require resource replacement. Valid values are 'static' or 'smart'.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("static", "smart"),
				},
			},
			"members": schema.SetAttribute{
				Description: "Optional device IDs to manage for STATIC groups. When omitted, the provider leaves membership unchanged. Ignored for SMART groups.",
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"member_count": schema.Int64Attribute{
				Description: "Total members reported by the API.",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"criteria": schema.ListNestedBlock{
				Description: "Smart-group criteria evaluated by the Jamf inventory service.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"order": schema.Int64Attribute{
							Description: "Execution order for the criterion. Defaults to the block index if omitted.",
							Optional:    true,
						},
						"criteria": schema.StringAttribute{
							Description: "Inventory attribute to evaluate.",
							Required:    true,
						},
						"operator": schema.StringAttribute{
							Description: "Operator to apply. Valid values are 'is', 'is not', 'has', 'does not have', 'member of', 'not member of', 'before (yyyy-mm-dd)', 'after (yyyy-mm-dd)', 'more than x days ago', 'less than x days ago', 'like', 'not like', 'greater than', 'more than', 'less than', 'greater than or equal', 'less than or equal', 'matches regex', 'does not match regex'.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf(
									"is",
									"is not",
									"has",
									"does not have",
									"member of",
									"not member of",
									"before (yyyy-mm-dd)",
									"after (yyyy-mm-dd)",
									"more than x days ago",
									"less than x days ago",
									"like",
									"not like",
									"greater than",
									"more than",
									"less than",
									"greater than or equal",
									"less than or equal",
									"matches regex",
									"does not match regex",
								),
							},
						},
						"value": schema.StringAttribute{
							Description: "Optional comparison value used by the operator.",
							Required:    true,
						},
						"and_or": schema.StringAttribute{
							Description: "How this criterion joins to the next. Defaults to and if omitted.",
							Computed:    true,
							Optional:    true,
							Default:     stringdefault.StaticString("and"),
							Validators: []validator.String{
								stringvalidator.OneOf("and", "or"),
							},
						},
						"has_opening_parenthesis": schema.BoolAttribute{
							Description: "Whether the criterion begins a parenthetical grouping.",
							Optional:    true,
						},
						"has_closing_parenthesis": schema.BoolAttribute{
							Description: "Whether the criterion ends a parenthetical grouping.",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

// Configure sets up the API client for the resource from the provider configuration.
func (r *DeviceGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ImportState handles the import of existing Device Group resources.
func (r *DeviceGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
