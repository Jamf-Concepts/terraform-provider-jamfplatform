// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package benchmark

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// BenchmarkResource implements the Terraform resource for Jamf Compliance Benchmark.
type BenchmarkResource struct {
	client *jamfplatform.Client
}

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BenchmarkResource{}
var _ resource.ResourceWithImportState = &BenchmarkResource{}
var _ resource.ResourceWithIdentity = &BenchmarkResource{}

const (
	defaultCreateTimeout = 15 * time.Minute
	defaultReadTimeout   = 60 * time.Second
	defaultDeleteTimeout = 15 * time.Minute
)

// uuidRegex matches UUID strings used to validate device group IDs.
var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// NewBenchmarkResource returns a new instance of BenchmarkResource.
func NewBenchmarkResource() resource.Resource {
	return &BenchmarkResource{}
}

// Metadata sets the resource type name for the Terraform provider.
func (r *BenchmarkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cbengine_benchmark"
}

// IdentitySchema defines the unique identifier for benchmark resources.
func (r *BenchmarkResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "Benchmark ID used to uniquely reference Jamf Compliance Benchmarks.",
				RequiredForImport: true,
			},
		},
	}
}

// Schema returns the Terraform schema for the benchmark resource.
func (r *BenchmarkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:             0,
		MarkdownDescription: "Creates a Jamf Compliance Benchmark. Creation is asynchronous: the API accepts the request and deploys associated artifacts to the MDM. The provider will poll the benchmark sync state until it reaches `SYNCED` or a terminal failure. Requires **Compliance Benchmarks API** access.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier assigned by the API (maps to benchmarkId).",
				Computed:            true,
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "Benchmark title (max length 100). Required and replaces the resource when changed.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 100)},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional human-readable description of the benchmark (max length 1000). Replaces the resource when changed.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(0, 1000)},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source_baseline_id": schema.StringAttribute{
				MarkdownDescription: "mSCP baseline identifier used as the source for rules. Required on creation, but computed for imports. Use the `jamfplatform_cbengine_baselines` data source to look up available baselines.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"sources": schema.ListNestedAttribute{
				MarkdownDescription: "Set of mSCP sources (branch + revision) to include in the benchmark. Required; changing sources requires replace. Use the `jamfplatform_cbengine_rules` data source to look up available sources.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"branch": schema.StringAttribute{
							MarkdownDescription: "Source branch name.",
							Required:            true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"revision": schema.StringAttribute{
							MarkdownDescription: "Source revision identifier.",
							Required:            true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
					},
				},
			},
			"rules": schema.ListNestedAttribute{
				MarkdownDescription: "Set of rules to include in the benchmark. Each entry references a rule id and whether it is enabled; additional metadata (title, section, ODV hints) are computed from the API. Use the `jamfplatform_cbengine_rules` data source to look up available rules.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Rule identifier from the baseline.",
							Required:            true,
						},
						"enabled": schema.BoolAttribute{
							MarkdownDescription: "Whether the rule is enabled in this benchmark.",
							Required:            true,
						},
						"section_name": schema.StringAttribute{
							MarkdownDescription: "Section name of the rule from the baseline.",
							Computed:            true,
						},
						"title": schema.StringAttribute{
							MarkdownDescription: "Rule title resolved from the baseline.",
							Computed:            true,
						},
						"references": schema.ListAttribute{
							MarkdownDescription: "Reference URLs or identifiers for the rule.",
							ElementType:         types.StringType,
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Rule description from the baseline.",
							Computed:            true,
						},
						"supported_os": schema.ListNestedAttribute{
							MarkdownDescription: "Operating systems supported by the rule.",
							Computed:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"os_type": schema.StringAttribute{
										MarkdownDescription: "Operating system type (e.g. `MAC_OS`, `IOS`).",
										Computed:            true,
									},
									"os_version": schema.Int64Attribute{
										MarkdownDescription: "OS version integer.",
										Computed:            true,
									},
									"management_type": schema.StringAttribute{
										MarkdownDescription: "Management type for the OS.",
										Computed:            true,
									},
								},
							},
						},
						"os_specific_defaults": schema.MapNestedAttribute{
							MarkdownDescription: "OS-specific defaults for the rule.",
							Computed:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"title": schema.StringAttribute{
										MarkdownDescription: "OS-specific rule title.",
										Computed:            true,
									},
									"description": schema.StringAttribute{
										MarkdownDescription: "OS-specific rule description.",
										Computed:            true,
									},
									"odv_value": schema.StringAttribute{
										MarkdownDescription: "Recommended organization-defined value for this OS.",
										Computed:            true,
									},
									"odv_hint": schema.StringAttribute{
										MarkdownDescription: "Hint for the organization-defined value.",
										Computed:            true,
									},
								},
							},
						},
						"odv_value": schema.StringAttribute{
							MarkdownDescription: "Optional organization-defined value to apply for this rule (if applicable).",
							Optional:            true,
							Computed:            true,
						},
						"odv_hint": schema.StringAttribute{
							MarkdownDescription: "Hint for ODV usage.",
							Computed:            true,
						},
						"odv_placeholder": schema.StringAttribute{
							MarkdownDescription: "Placeholder for ODV input.",
							Computed:            true,
						},
						"odv_type": schema.StringAttribute{
							MarkdownDescription: "ODV type (`INTEGER`, `STRING`, `ENUM`, `REGEX`) when applicable.",
							Computed:            true,
						},
						"odv_validation_min": schema.Int64Attribute{
							MarkdownDescription: "Minimum validation for `INTEGER` ODV types.",
							Computed:            true,
						},
						"odv_validation_max": schema.Int64Attribute{
							MarkdownDescription: "Maximum validation for `INTEGER` ODV types.",
							Computed:            true,
						},
						"odv_validation_enum_values": schema.ListAttribute{
							MarkdownDescription: "Allowed enum values for `ENUM` ODV types.",
							ElementType:         types.StringType,
							Computed:            true,
						},
						"odv_validation_regex": schema.StringAttribute{
							MarkdownDescription: "Regex pattern for `REGEX` ODV types.",
							Computed:            true,
						},
						"depends_on": schema.ListAttribute{
							MarkdownDescription: "Rule IDs this rule depends on.",
							ElementType:         types.StringType,
							Computed:            true,
						},
						"reportable": schema.BoolAttribute{
							MarkdownDescription: "Whether the rule produces reportable compliance data.",
							Computed:            true,
						},
						"smart_card": schema.BoolAttribute{
							MarkdownDescription: "Whether the rule is related to smart card configuration.",
							Computed:            true,
						},
					},
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"target_device_group": schema.StringAttribute{
				MarkdownDescription: "Device group Platform ID targeted by this benchmark. Specified as a string in UUID format. The Platform ID can be sourced from the response body of the /api/v1/groups Jamf Pro API endpoint. Required and immutable for this resource (replace on change).",
				Required:            true,
				Validators: []validator.String{stringvalidator.RegexMatches(uuidRegex,
					"Device group ID must be a valid UUID")},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enforcement_mode": schema.StringAttribute{
				MarkdownDescription: "Enforcement mode for the benchmark; allowed values: MONITOR or MONITOR_AND_ENFORCE. Required and immutable for this resource (replace on change).",
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf("MONITOR", "MONITOR_AND_ENFORCE")},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "Identifier for the tenant that owns the benchmark.",
				Computed:            true,
			},
			"deleted": schema.BoolAttribute{
				MarkdownDescription: "Whether the benchmark is marked deleted by the API.",
				Computed:            true,
			},
			"update_available": schema.BoolAttribute{
				MarkdownDescription: "Whether an update is available for the benchmark relative to current mSCP sources.",
				Computed:            true,
			},
			"can_switch_to_enforce": schema.BoolAttribute{
				MarkdownDescription: "Whether the benchmark can be switched to MONITOR_AND_ENFORCE enforcement mode.",
				Computed:            true,
			},
			"last_updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp (RFC3339) of the last update to the benchmark.",
				Computed:            true,
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Delete: true,
			}),
		},
	}
}

// Configure sets up the API client for the resource from the provider configuration.
func (r *BenchmarkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*jamfplatform.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *jamfplatform.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// ImportState handles the import of existing Benchmark resources.
func (r *BenchmarkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
