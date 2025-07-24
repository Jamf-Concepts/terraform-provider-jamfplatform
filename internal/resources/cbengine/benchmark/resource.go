// Copyright 2025 Jamf Software LLC.

package benchmark

import (
	"context"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// BenchmarkResourceSchema returns the Terraform schema for the Jamf Compliance Benchmark resource.
func BenchmarkResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Resource schema for creating a Jamf Compliance Benchmark.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the benchmark (maps to benchmarkId in the API).",
				Computed:    true,
			},
			"title": schema.StringAttribute{
				Description: "Benchmark title.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Benchmark description.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source_baseline_id": schema.StringAttribute{
				Description: "Source baseline ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"sources": schema.ListNestedAttribute{
				Description: "List of sources.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"branch": schema.StringAttribute{
							Description: "Source branch.",
							Required:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"revision": schema.StringAttribute{
							Description: "Source revision.",
							Required:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
					},
				},
			},
			"rules": schema.ListNestedAttribute{
				Description: "List of rule IDs to include in the benchmark, with enabled flag and computed fields.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Rule ID.",
							Required:    true,
						},
						"enabled": schema.BoolAttribute{
							Description: "Whether the rule is enabled.",
							Required:    true,
						},
						"section_name": schema.StringAttribute{
							Description: "Section name for the rule.",
							Computed:    true,
						},
						"title": schema.StringAttribute{
							Description: "Title of the rule.",
							Computed:    true,
						},
						"references": schema.ListAttribute{
							Description: "References for the rule.",
							ElementType: types.StringType,
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Description of the rule.",
							Computed:    true,
						},
						"supported_os": schema.ListNestedAttribute{
							Description: "Supported OS for the rule.",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"os_type": schema.StringAttribute{
										Description: "OS type.",
										Computed:    true,
									},
									"os_version": schema.Int64Attribute{
										Description: "OS version.",
										Computed:    true,
									},
									"management_type": schema.StringAttribute{
										Description: "Management type.",
										Computed:    true,
									},
								},
							},
						},
						"os_specific_defaults": schema.MapNestedAttribute{
							Description: "OS specific defaults for the rule.",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"title": schema.StringAttribute{
										Description: "OS-specific rule title.",
										Computed:    true,
									},
									"description": schema.StringAttribute{
										Description: "OS-specific rule description.",
										Computed:    true,
									},
									"odv": schema.SingleNestedAttribute{
										Description: "ODV recommendation for this OS.",
										Computed:    true,
										Attributes: map[string]schema.Attribute{
											"value": schema.StringAttribute{
												Description: "Recommended ODV value.",
												Computed:    true,
											},
											"hint": schema.StringAttribute{
												Description: "Recommended ODV hint.",
												Computed:    true,
											},
										},
									},
								},
							},
						},
						"odv": schema.SingleNestedAttribute{
							Description: "Organization defined value for the rule.",
							Optional:    true,
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"value": schema.StringAttribute{
									Description: "ODV value.",
									Optional:    true,
									Computed:    true,
								},
								"hint": schema.StringAttribute{
									Description: "ODV hint.",
									Computed:    true,
								},
								"placeholder": schema.StringAttribute{
									Description: "ODV placeholder.",
									Computed:    true,
								},
								"type": schema.StringAttribute{
									Description: "ODV type.",
									Computed:    true,
								},
								"validation": schema.SingleNestedAttribute{
									Description: "ODV validation constraints.",
									Computed:    true,
									Attributes: map[string]schema.Attribute{
										"min": schema.Int64Attribute{
											Description: "Minimum value constraint for INTEGER type.",
											Computed:    true,
										},
										"max": schema.Int64Attribute{
											Description: "Maximum value constraint for INTEGER type.",
											Computed:    true,
										},
										"enum_values": schema.ListAttribute{
											Description: "Enumeration values for ENUM type.",
											ElementType: types.StringType,
											Computed:    true,
										},
										"regex": schema.StringAttribute{
											Description: "Regular expression pattern for REGEX type.",
											Computed:    true,
										},
									},
								},
							},
						},
						"rule_relation": schema.SingleNestedAttribute{
							Description: "Rule dependencies.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"depends_on": schema.ListAttribute{
									Description: "IDs of rules this rule depends on.",
									ElementType: types.StringType,
									Computed:    true,
								},
							},
						},
					},
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"target": schema.SingleNestedAttribute{
				Description: "Target configuration.",
				Required:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"device_groups": schema.ListAttribute{
						Description: "Device groups.",
						ElementType: types.StringType,
						Required:    true,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.RequiresReplace(),
						},
					},
				},
			},
			"enforcement_mode": schema.StringAttribute{
				Description: "Enforcement mode (MONITOR or MONITOR_AND_ENFORCE).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tenant_id": schema.StringAttribute{
				Description: "Tenant ID for the benchmark.",
				Computed:    true,
			},
			"deleted": schema.BoolAttribute{
				Description: "Whether the benchmark is deleted.",
				Computed:    true,
			},
			"update_available": schema.BoolAttribute{
				Description: "Whether an update is available for the benchmark.",
				Computed:    true,
			},
			"last_updated_at": schema.StringAttribute{
				Description: "Timestamp of the last update to the benchmark.",
				Computed:    true,
			},
		},
	}
}

// NewBenchmarkResource returns a new instance of BenchmarkResource.
func NewBenchmarkResource() resource.Resource {
	return &BenchmarkResource{}
}

// Metadata sets the resource type name for the Terraform provider.
func (r *BenchmarkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cbengine_benchmark"
}

// Schema sets the Terraform schema for the resource.
func (r *BenchmarkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = BenchmarkResourceSchema()
}

// Configure sets up the API client for the resource from the provider configuration.
func (r *BenchmarkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
