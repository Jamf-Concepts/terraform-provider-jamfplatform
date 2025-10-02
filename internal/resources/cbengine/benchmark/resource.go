// Copyright 2025 Jamf Software LLC.

package benchmark

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
var _ resource.Resource = &BenchmarkResource{}
var _ resource.ResourceWithImportState = &BenchmarkResource{}

// NewBenchmarkResource returns a new instance of BenchmarkResource.
func NewBenchmarkResource() resource.Resource {
	return &BenchmarkResource{}
}

// Metadata sets the resource type name for the Terraform provider.
func (r *BenchmarkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cbengine_benchmark"
}

// Schema returns the Terraform schema for the benchmark resource.
func (r *BenchmarkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates a Jamf Compliance Benchmark. Creation is asynchronous: the API accepts the request and deploys associated artifacts to the MDM. The provider will poll the benchmark sync state until it reaches SYNCED or a terminal failure.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier assigned by the API (maps to benchmarkId).",
				Computed:    true,
			},
			"title": schema.StringAttribute{
				Description: "Benchmark title (max length 100). Required and replaces the resource when changed.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Optional human-readable description of the benchmark (max length 1000). Replaces the resource when changed.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source_baseline_id": schema.StringAttribute{
				Description: "mSCP baseline identifier used as the source for rules. Required and immutable for this resource (replace on change).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"sources": schema.ListNestedAttribute{
				Description: "List of mSCP sources (branch + revision) to include in the benchmark. Required; changing sources requires replace.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"branch": schema.StringAttribute{
							Description: "Source branch name.",
							Required:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"revision": schema.StringAttribute{
							Description: "Source revision identifier.",
							Required:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
					},
				},
			},
			"rules": schema.ListNestedAttribute{
				Description: "Ordered list of rules to include in the benchmark. Each entry references a rule id and whether it is enabled; additional metadata (title, section, ODV hints) are computed from the API.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Rule identifier from the baseline.",
							Required:    true,
						},
						"enabled": schema.BoolAttribute{
							Description: "Whether the rule is enabled in this benchmark.",
							Required:    true,
						},
						"section_name": schema.StringAttribute{
							Description: "Section name of the rule from the baseline.",
							Computed:    true,
						},
						"title": schema.StringAttribute{
							Description: "Rule title resolved from the baseline.",
							Computed:    true,
						},
						"references": schema.ListAttribute{
							Description: "Reference URLs or identifiers for the rule.",
							ElementType: types.StringType,
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Rule description from the baseline.",
							Computed:    true,
						},
						"supported_os": schema.ListNestedAttribute{
							Description: "Operating systems supported by the rule.",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"os_type": schema.StringAttribute{
										Description: "Operating system type (e.g. MAC_OS, IOS).",
										Computed:    true,
									},
									"os_version": schema.Int64Attribute{
										Description: "OS version integer.",
										Computed:    true,
									},
									"management_type": schema.StringAttribute{
										Description: "Management type for the OS.",
										Computed:    true,
									},
								},
							},
						},
						"os_specific_defaults": schema.MapNestedAttribute{
							Description: "OS-specific defaults for the rule.",
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
									"odv_value": schema.StringAttribute{
										Description: "Recommended organization-defined value for this OS.",
										Computed:    true,
									},
									"odv_hint": schema.StringAttribute{
										Description: "Hint for the organization-defined value.",
										Computed:    true,
									},
								},
							},
						},
						"odv_value": schema.StringAttribute{
							Description: "Optional organization-defined value to apply for this rule (if applicable).",
							Optional:    true,
							Computed:    true,
						},
						"odv_hint": schema.StringAttribute{
							Description: "Hint for ODV usage.",
							Computed:    true,
						},
						"odv_placeholder": schema.StringAttribute{
							Description: "Placeholder for ODV input.",
							Computed:    true,
						},
						"odv_type": schema.StringAttribute{
							Description: "ODV type (INTEGER, STRING, ENUM, REGEX) when applicable.",
							Computed:    true,
						},
						"odv_validation_min": schema.Int64Attribute{
							Description: "Minimum validation for INTEGER ODV types.",
							Computed:    true,
						},
						"odv_validation_max": schema.Int64Attribute{
							Description: "Maximum validation for INTEGER ODV types.",
							Computed:    true,
						},
						"odv_validation_enum_values": schema.ListAttribute{
							Description: "Allowed enum values for ENUM ODV types.",
							ElementType: types.StringType,
							Computed:    true,
						},
						"odv_validation_regex": schema.StringAttribute{
							Description: "Regex pattern for REGEX ODV types.",
							Computed:    true,
						},
						"depends_on": schema.ListAttribute{
							Description: "List of rule IDs this rule depends on.",
							ElementType: types.StringType,
							Computed:    true,
						},
					},
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"target_device_group": schema.StringAttribute{
				Description: "Device group ID(s) targeted by this benchmark (maps to target.deviceGroups). Required and immutable for this resource (replace on change).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enforcement_mode": schema.StringAttribute{
				Description: "Enforcement mode for the benchmark; allowed values: MONITOR or MONITOR_AND_ENFORCE. Required and immutable for this resource (replace on change).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tenant_id": schema.StringAttribute{
				Description: "Identifier for the tenant that owns the benchmark.",
				Computed:    true,
			},
			"deleted": schema.BoolAttribute{
				Description: "Whether the benchmark is marked deleted by the API.",
				Computed:    true,
			},
			"update_available": schema.BoolAttribute{
				Description: "Whether an update is available for the benchmark relative to current mSCP sources.",
				Computed:    true,
			},
			"last_updated_at": schema.StringAttribute{
				Description: "Timestamp (RFC3339) of the last update to the benchmark.",
				Computed:    true,
			},
		},
	}
}

// Configure sets up the API client for the resource from the provider configuration.
func (r *BenchmarkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ImportState handles the import of existing Benchmark resources.
func (r *BenchmarkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
