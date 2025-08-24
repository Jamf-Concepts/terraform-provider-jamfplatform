// Copyright 2025 Jamf Software LLC.

package benchmark

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
)

// NewBenchmarkDataSource returns a new instance of benchmarkDataSource.
func NewBenchmarkDataSource() datasource.DataSource {
	return &benchmarkDataSource{}
}

// Configure sets up the API client for the data source from the provider configuration.
func (d *benchmarkDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	d.client = apiClient
}

// Metadata sets the data source type name for the Terraform provider.
func (d *benchmarkDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cbengine_benchmark"
}

// Schema sets the Terraform schema for the data source.
func (d *benchmarkDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns a benchmark by ID or title.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The benchmark ID to fetch. Optional if title is set.",
				Optional:    true,
			},
			"title": schema.StringAttribute{
				Description: "The benchmark title to fetch. Optional if id is set.",
				Optional:    true,
			},
			"benchmark_id": schema.StringAttribute{
				Description: "Benchmark ID.",
				Computed:    true,
			},
			"tenant_id": schema.StringAttribute{
				Description: "Tenant ID.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description.",
				Computed:    true,
			},
			"sources": schema.ListNestedAttribute{
				Description: "Sources.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"branch": schema.StringAttribute{
							Description: "Branch.",
							Computed:    true,
						},
						"revision": schema.StringAttribute{
							Description: "Revision.",
							Computed:    true,
						},
					},
				},
			},
			"rules": schema.ListNestedAttribute{
				Description: "Rules.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Rule ID.",
							Computed:    true,
						},
						"section_name": schema.StringAttribute{
							Description: "Section name for the rule.",
							Computed:    true,
						},
						"enabled": schema.BoolAttribute{
							Description: "Whether the rule is enabled.",
							Computed:    true,
						},
						"title": schema.StringAttribute{
							Description: "Title of the rule.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Description of the rule.",
							Computed:    true,
						},
						"references": schema.ListAttribute{
							Description: "References for the rule.",
							ElementType: types.StringType,
							Computed:    true,
						},
						"odv_value": schema.StringAttribute{
							Description: "ODV value.",
							Computed:    true,
						},
						"odv_hint": schema.StringAttribute{
							Description: "ODV hint.",
							Computed:    true,
						},
						"odv_placeholder": schema.StringAttribute{
							Description: "ODV placeholder.",
							Computed:    true,
						},
						"odv_type": schema.StringAttribute{
							Description: "ODV type.",
							Computed:    true,
						},
						"odv_validation_min": schema.Int64Attribute{
							Description: "Minimum value.",
							Computed:    true,
						},
						"odv_validation_max": schema.Int64Attribute{
							Description: "Maximum value.",
							Computed:    true,
						},
						"odv_validation_enum_values": schema.ListAttribute{
							Description: "Allowed enum values.",
							ElementType: types.StringType,
							Computed:    true,
						},
						"odv_validation_regex": schema.StringAttribute{
							Description: "Regex pattern.",
							Computed:    true,
						},
						"supported_os": schema.ListNestedAttribute{
							Description: "Supported operating systems.",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"os_type": schema.StringAttribute{
										Description: "OS type (e.g. MAC_OS, IOS).",
										Computed:    true,
									},
									"os_version": schema.Int64Attribute{
										Description: "OS version.",
										Computed:    true,
									},
									"management_type": schema.StringAttribute{
										Description: "Management type (e.g. MANAGED, BYOD).",
										Computed:    true,
									},
								},
							},
						},
						"os_specific_defaults": schema.MapNestedAttribute{
							Description: "OS-specific rule defaults.",
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
										Description: "Recommended ODV value.",
										Computed:    true,
									},
									"odv_hint": schema.StringAttribute{
										Description: "Recommended ODV hint.",
										Computed:    true,
									},
								},
							},
						},
						"depends_on": schema.ListAttribute{
							Description: "IDs of rules this rule depends on.",
							ElementType: types.StringType,
							Computed:    true,
						},
					},
				},
			},
			"target_device_group": schema.StringAttribute{
				Description: "Device group for the target configuration.",
				Computed:    true,
			},
			"enforcement_mode": schema.StringAttribute{
				Description: "Enforcement mode.",
				Computed:    true,
			},
			"deleted": schema.BoolAttribute{
				Description: "Deleted flag.",
				Computed:    true,
			},
			"update_available": schema.BoolAttribute{
				Description: "Update available flag.",
				Computed:    true,
			},
			"last_updated_at": schema.StringAttribute{
				Description: "Last updated at (RFC3339).",
				Computed:    true,
			},
		},
	}
}

// Read fetches a benchmark by ID or title and populates the Terraform state.
func (d *benchmarkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config benchmarkDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider client was not configured. Please ensure provider block is set up correctly.",
		)
		return
	}

	var bench *client.CBEngineBenchmarkResponse
	var err error
	if !config.ID.IsNull() && config.ID.ValueString() != "" {
		bench, err = d.client.GetCBEngineBenchmarkByID(ctx, config.ID.ValueString())
	} else if !config.Title.IsNull() && config.Title.ValueString() != "" {
		bench, err = d.client.GetCBEngineBenchmarkByTitle(ctx, config.Title.ValueString())
	} else {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"Either 'id' or 'title' must be set to look up a benchmark.",
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get benchmark",
			err.Error(),
		)
		return
	}

	sources := make([]sourceModel, 0, len(bench.Sources))
	for _, s := range bench.Sources {
		sources = append(sources, sourceModel{
			Branch:   types.StringValue(s.Branch),
			Revision: types.StringValue(s.Revision),
		})
	}

	rules := make([]ruleModel, 0, len(bench.Rules))
	for _, r := range bench.Rules {
		var references types.List
		if len(r.References) == 0 {
			references = types.ListNull(types.StringType)
		} else {
			vals := make([]attr.Value, len(r.References))
			for j, ref := range r.References {
				vals[j] = types.StringValue(ref)
			}
			references, _ = types.ListValue(types.StringType, vals)
		}

		var odvValue, odvHint, odvPlaceholder, odvType, odvValidationRegex types.String
		var odvValidationMin, odvValidationMax types.Int64
		var odvValidationEnumValues types.List
		if r.ODV != nil {
			odvValue = types.StringValue(r.ODV.Value)
			odvHint = types.StringValue(r.ODV.Hint)
			odvPlaceholder = types.StringValue(r.ODV.Placeholder)
			odvType = types.StringValue(r.ODV.Type)
			if r.ODV.Validation != nil {
				if r.ODV.Validation.Min != nil {
					odvValidationMin = types.Int64Value(int64(*r.ODV.Validation.Min))
				} else {
					odvValidationMin = types.Int64Null()
				}
				if r.ODV.Validation.Max != nil {
					odvValidationMax = types.Int64Value(int64(*r.ODV.Validation.Max))
				} else {
					odvValidationMax = types.Int64Null()
				}
				enumValues := make([]attr.Value, len(r.ODV.Validation.EnumValues))
				for k, v := range r.ODV.Validation.EnumValues {
					enumValues[k] = types.StringValue(v)
				}
				if len(enumValues) == 0 {
					odvValidationEnumValues = types.ListNull(types.StringType)
				} else {
					odvValidationEnumValues, _ = types.ListValue(types.StringType, enumValues)
				}
				odvValidationRegex = types.StringValue(r.ODV.Validation.Regex)
			} else {
				odvValidationMin = types.Int64Null()
				odvValidationMax = types.Int64Null()
				odvValidationEnumValues = types.ListNull(types.StringType)
				odvValidationRegex = types.StringNull()
			}
		} else {
			odvValue = types.StringNull()
			odvHint = types.StringNull()
			odvPlaceholder = types.StringNull()
			odvType = types.StringNull()
			odvValidationMin = types.Int64Null()
			odvValidationMax = types.Int64Null()
			odvValidationEnumValues = types.ListNull(types.StringType)
			odvValidationRegex = types.StringNull()
		}

		var supportedOS types.List
		osInfoObjType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"os_type":         types.StringType,
				"os_version":      types.Int64Type,
				"management_type": types.StringType,
			},
		}
		if len(r.SupportedOS) == 0 {
			supportedOS = types.ListNull(osInfoObjType)
		} else {
			osVals := make([]attr.Value, len(r.SupportedOS))
			for j, os := range r.SupportedOS {
				osVals[j], _ = types.ObjectValue(
					map[string]attr.Type{
						"os_type":         types.StringType,
						"os_version":      types.Int64Type,
						"management_type": types.StringType,
					},
					map[string]attr.Value{
						"os_type":         types.StringValue(os.OSType),
						"os_version":      types.Int64Value(int64(os.OSVersion)),
						"management_type": types.StringValue(os.ManagementType),
					},
				)
			}
			supportedOS, _ = types.ListValue(osInfoObjType, osVals)
		}

		osSpecObjType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"title":       types.StringType,
				"description": types.StringType,
				"odv_value":   types.StringType,
				"odv_hint":    types.StringType,
			},
		}
		var osSpecificDefaults types.Map
		if len(r.OSSpecificDefaults) == 0 {
			osSpecificDefaults = types.MapNull(osSpecObjType)
		} else {
			vals := make(map[string]attr.Value, len(r.OSSpecificDefaults))
			for k, v := range r.OSSpecificDefaults {
				var odvValue, odvHint types.String
				if v.ODV != nil {
					odvValue = types.StringValue(v.ODV.Value)
					odvHint = types.StringValue(v.ODV.Hint)
				} else {
					odvValue = types.StringNull()
					odvHint = types.StringNull()
				}
				vals[k], _ = types.ObjectValue(
					map[string]attr.Type{
						"title":       types.StringType,
						"description": types.StringType,
						"odv_value":   types.StringType,
						"odv_hint":    types.StringType,
					},
					map[string]attr.Value{
						"title":       types.StringValue(v.Title),
						"description": types.StringValue(v.Description),
						"odv_value":   odvValue,
						"odv_hint":    odvHint,
					},
				)
			}
			osSpecificDefaults, _ = types.MapValue(osSpecObjType, vals)
		}

		var dependsOn types.List
		if r.RuleRelation == nil || len(r.RuleRelation.DependsOn) == 0 {
			dependsOn = types.ListNull(types.StringType)
		} else {
			vals := make([]attr.Value, len(r.RuleRelation.DependsOn))
			for j, dep := range r.RuleRelation.DependsOn {
				vals[j] = types.StringValue(dep)
			}
			dependsOn, _ = types.ListValue(types.StringType, vals)
		}

		rules = append(rules, ruleModel{
			ID:                      types.StringValue(r.ID),
			SectionName:             types.StringValue(r.SectionName),
			Enabled:                 types.BoolValue(r.Enabled),
			Title:                   types.StringValue(r.Title),
			Description:             types.StringValue(r.Description),
			References:              references,
			ODVValue:                odvValue,
			ODVHint:                 odvHint,
			ODVPlaceholder:          odvPlaceholder,
			ODVType:                 odvType,
			ODVValidationMin:        odvValidationMin,
			ODVValidationMax:        odvValidationMax,
			ODVValidationEnumValues: odvValidationEnumValues,
			ODVValidationRegex:      odvValidationRegex,
			SupportedOS:             supportedOS,
			OSSpecificDefaults:      osSpecificDefaults,
			DependsOn:               dependsOn,
		})
	}

	var targetDeviceGroup types.String
	if len(bench.Target.DeviceGroups) > 0 {
		targetDeviceGroup = types.StringValue(bench.Target.DeviceGroups[0])
	} else {
		targetDeviceGroup = types.StringNull()
	}

	state := benchmarkDataSourceModel{
		ID:                config.ID,
		BenchmarkID:       types.StringValue(bench.BenchmarkID),
		TenantID:          types.StringValue(bench.TenantID),
		Title:             types.StringValue(bench.Title),
		Description:       types.StringValue(bench.Description),
		Sources:           sources,
		Rules:             rules,
		TargetDeviceGroup: targetDeviceGroup,
		EnforcementMode:   types.StringValue(bench.EnforcementMode),
		Deleted:           types.BoolValue(bench.Deleted),
		UpdateAvailable:   types.BoolValue(bench.UpdateAvailable),
		LastUpdatedAt:     types.StringValue(bench.LastUpdatedAt.Format("2006-01-02T15:04:05Z07:00")),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
