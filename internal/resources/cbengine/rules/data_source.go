// Copyright 2025 Jamf Software LLC.

package rules

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &RulesDataSource{}

// NewRulesDataSource returns a new instance of RulesDataSource.
func NewRulesDataSource() datasource.DataSource {
	return &RulesDataSource{}
}

// Metadata sets the data source type name for the Terraform provider.
func (d *RulesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cbengine_rules"
}

// Schema sets the Terraform schema for the data source.
func (d *RulesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns list of rules for a given mSCP baseline.",
		Attributes: map[string]schema.Attribute{
			"baseline_id": schema.StringAttribute{
				Description: "The baseline ID to fetch rules for.",
				Required:    true,
			},
			"sources": schema.ListNestedAttribute{
				Description: "List of sources for the rules baseline.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"branch": schema.StringAttribute{
							Description: "Source branch.",
							Computed:    true,
						},
						"revision": schema.StringAttribute{
							Description: "Source revision.",
							Computed:    true,
						},
					},
				},
			},
			"rules": schema.ListNestedAttribute{
				Description: "List of rules for the baseline.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Unique identifier for the rule.",
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
							Description: "ODV validation minimum value.",
							Computed:    true,
						},
						"odv_validation_max": schema.Int64Attribute{
							Description: "ODV validation maximum value.",
							Computed:    true,
						},
						"odv_validation_enum_values": schema.ListAttribute{
							Description: "ODV validation allowed enum values.",
							ElementType: types.StringType,
							Computed:    true,
						},
						"odv_validation_regex": schema.StringAttribute{
							Description: "ODV validation regex pattern.",
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
		},
	}
}

// Configure sets up the API client for the data source from the provider configuration.
func (d *RulesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

// Read implements datasource.DataSource for RulesDataSource. It fetches the list of rules from the API and sets the state.
func (d *RulesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RulesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

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

	rulesResp, err := d.client.GetCBEngineRules(ctx, data.BaselineID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get rules",
			err.Error(),
		)
		return
	}

	var sources []SourceModel
	for _, s := range rulesResp.Sources {
		sources = append(sources, SourceModel{
			Branch:   types.StringValue(s.Branch),
			Revision: types.StringValue(s.Revision),
		})
	}

	var rules []RuleModel
	for _, r := range rulesResp.Rules {
		var references []types.String
		for _, ref := range r.References {
			references = append(references, types.StringValue(ref))
		}

		var odvValue, odvHint, odvPlaceholder, odvType types.String
		var odvValidationMin, odvValidationMax types.Int64
		var odvValidationEnumValues []types.String
		var odvValidationRegex types.String
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
				for _, v := range r.ODV.Validation.EnumValues {
					odvValidationEnumValues = append(odvValidationEnumValues, types.StringValue(v))
				}
				odvValidationRegex = types.StringValue(r.ODV.Validation.Regex)
			}
		}

		var supportedOS []OSInfoModel
		for _, os := range r.SupportedOS {
			supportedOS = append(supportedOS, OSInfoModel{
				OSType:         types.StringValue(os.OSType),
				OSVersion:      types.Int64Value(int64(os.OSVersion)),
				ManagementType: types.StringValue(os.ManagementType),
			})
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

		var dependsOn []types.String
		if r.RuleRelation != nil {
			for _, dep := range r.RuleRelation.DependsOn {
				dependsOn = append(dependsOn, types.StringValue(dep))
			}
		}

		rules = append(rules, RuleModel{
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

	data = RulesDataSourceModel{
		BaselineID: data.BaselineID,
		Sources:    sources,
		Rules:      rules,
	}

	tflog.Trace(ctx, "read a data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
