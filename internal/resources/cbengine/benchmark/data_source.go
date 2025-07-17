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
	clientSet, ok := req.ProviderData.(*client.ClientSet)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected ProviderData type",
			"Expected *client.ClientSet, got something else.",
		)
		return
	}
	if clientSet.CBEngine == nil {
		resp.Diagnostics.AddError(
			"CBEngine API client not configured",
			"The provider's cbengine block is missing or incomplete. Please provide valid credentials.",
		)
		return
	}
	d.client = clientSet.CBEngine
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
						"odv": schema.SingleNestedAttribute{
							Description: "Organization defined value for the rule.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"value": schema.StringAttribute{
									Description: "ODV value.",
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
											Description: "Minimum value.",
											Computed:    true,
										},
										"max": schema.Int64Attribute{
											Description: "Maximum value.",
											Computed:    true,
										},
										"enum_values": schema.ListAttribute{
											Description: "Allowed enum values.",
											ElementType: types.StringType,
											Computed:    true,
										},
										"regex": schema.StringAttribute{
											Description: "Regex pattern.",
											Computed:    true,
										},
									},
								},
							},
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
			},
			"target": schema.SingleNestedAttribute{
				Description: "Target.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"device_groups": schema.ListAttribute{
						Description: "Device groups.",
						ElementType: types.StringType,
						Computed:    true,
					},
				},
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

		var odv types.Object
		if r.ODV != nil {
			validationObjType := map[string]attr.Type{
				"min":         types.Int64Type,
				"max":         types.Int64Type,
				"enum_values": types.ListType{ElemType: types.StringType},
				"regex":       types.StringType,
			}
			var validation types.Object
			if r.ODV.Validation != nil {
				enumValues := make([]attr.Value, len(r.ODV.Validation.EnumValues))
				for k, v := range r.ODV.Validation.EnumValues {
					enumValues[k] = types.StringValue(v)
				}
				var enumValuesList types.List
				if len(enumValues) == 0 {
					enumValuesList = types.ListNull(types.StringType)
				} else {
					enumValuesList, _ = types.ListValue(types.StringType, enumValues)
				}

				var minVal, maxVal types.Int64
				if r.ODV.Validation.Min != nil {
					minVal = types.Int64Value(int64(*r.ODV.Validation.Min))
				} else {
					minVal = types.Int64Null()
				}
				if r.ODV.Validation.Max != nil {
					maxVal = types.Int64Value(int64(*r.ODV.Validation.Max))
				} else {
					maxVal = types.Int64Null()
				}

				validation, _ = types.ObjectValue(
					validationObjType,
					map[string]attr.Value{
						"min":         minVal,
						"max":         maxVal,
						"enum_values": enumValuesList,
						"regex":       types.StringValue(r.ODV.Validation.Regex),
					},
				)
			} else {
				validation = types.ObjectNull(validationObjType)
			}

			odv, _ = types.ObjectValue(
				map[string]attr.Type{
					"value":       types.StringType,
					"hint":        types.StringType,
					"placeholder": types.StringType,
					"type":        types.StringType,
					"validation": types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"min":         types.Int64Type,
							"max":         types.Int64Type,
							"enum_values": types.ListType{ElemType: types.StringType},
							"regex":       types.StringType,
						},
					},
				},
				map[string]attr.Value{
					"value":       types.StringValue(r.ODV.Value),
					"hint":        types.StringValue(r.ODV.Hint),
					"placeholder": types.StringValue(r.ODV.Placeholder),
					"type":        types.StringValue(r.ODV.Type),
					"validation":  validation,
				},
			)
		} else {
			odv = types.ObjectNull(map[string]attr.Type{
				"value":       types.StringType,
				"hint":        types.StringType,
				"placeholder": types.StringType,
				"type":        types.StringType,
				"validation": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"min":         types.Int64Type,
						"max":         types.Int64Type,
						"enum_values": types.ListType{ElemType: types.StringType},
						"regex":       types.StringType,
					},
				},
			})
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
				"odv": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"value": types.StringType,
						"hint":  types.StringType,
					},
				},
			},
		}
		var osSpecificDefaults types.Map
		if len(r.OSSpecificDefaults) == 0 {
			osSpecificDefaults = types.MapNull(osSpecObjType)
		} else {
			vals := make(map[string]attr.Value, len(r.OSSpecificDefaults))
			for k, v := range r.OSSpecificDefaults {
				var odvVal attr.Value = types.ObjectNull(map[string]attr.Type{"value": types.StringType, "hint": types.StringType})
				if v.ODV != nil {
					odvVal, _ = types.ObjectValue(
						map[string]attr.Type{
							"value": types.StringType,
							"hint":  types.StringType,
						},
						map[string]attr.Value{
							"value": types.StringValue(v.ODV.Value),
							"hint":  types.StringValue(v.ODV.Hint),
						},
					)
				}
				vals[k], _ = types.ObjectValue(
					map[string]attr.Type{
						"title":       types.StringType,
						"description": types.StringType,
						"odv": types.ObjectType{
							AttrTypes: map[string]attr.Type{
								"value": types.StringType,
								"hint":  types.StringType,
							},
						},
					},
					map[string]attr.Value{
						"title":       types.StringValue(v.Title),
						"description": types.StringValue(v.Description),
						"odv":         odvVal,
					},
				)
			}
			osSpecificDefaults, _ = types.MapValue(osSpecObjType, vals)
		}

		ruleRelObjType := map[string]attr.Type{
			"depends_on": types.ListType{ElemType: types.StringType},
		}
		var ruleRelation types.Object
		if r.RuleRelation == nil || len(r.RuleRelation.DependsOn) == 0 {
			ruleRelation = types.ObjectNull(ruleRelObjType)
		} else {
			dependsOnVals := make([]attr.Value, len(r.RuleRelation.DependsOn))
			for j, dep := range r.RuleRelation.DependsOn {
				dependsOnVals[j] = types.StringValue(dep)
			}
			dependsOnList, _ := types.ListValue(types.StringType, dependsOnVals)
			ruleRelation, _ = types.ObjectValue(
				ruleRelObjType,
				map[string]attr.Value{"depends_on": dependsOnList},
			)
		}

		rules = append(rules, ruleModel{
			ID:                 types.StringValue(r.ID),
			SectionName:        types.StringValue(r.SectionName),
			Enabled:            types.BoolValue(r.Enabled),
			Title:              types.StringValue(r.Title),
			Description:        types.StringValue(r.Description),
			References:         references,
			ODV:                odv,
			SupportedOS:        supportedOS,
			OSSpecificDefaults: osSpecificDefaults,
			RuleRelation:       ruleRelation,
		})
	}

	var target *targetModel
	if len(bench.Target.DeviceGroups) > 0 {
		groups := make([]types.String, 0, len(bench.Target.DeviceGroups))
		for _, g := range bench.Target.DeviceGroups {
			groups = append(groups, types.StringValue(g))
		}
		target = &targetModel{DeviceGroups: groups}
	}

	state := benchmarkDataSourceModel{
		ID:              config.ID,
		BenchmarkID:     types.StringValue(bench.BenchmarkID),
		TenantID:        types.StringValue(bench.TenantID),
		Title:           types.StringValue(bench.Title),
		Description:     types.StringValue(bench.Description),
		Sources:         sources,
		Rules:           rules,
		Target:          target,
		EnforcementMode: types.StringValue(bench.EnforcementMode),
		Deleted:         types.BoolValue(bench.Deleted),
		UpdateAvailable: types.BoolValue(bench.UpdateAvailable),
		LastUpdatedAt:   types.StringValue(bench.LastUpdatedAt.Format("2006-01-02T15:04:05Z07:00")),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
