// Copyright 2025 Jamf Software LLC.

package rules

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
)

// rulesDataSource implements the Terraform data source for mSCP rules.
type rulesDataSource struct {
	client *client.Client
}

// NewRulesDataSource returns a new instance of the rules data source.
func NewRulesDataSource() datasource.DataSource {
	return &rulesDataSource{}
}

// Configure sets up the API client for the data source from the provider configuration.
func (d *rulesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *rulesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cbengine_rules"
}

// Schema sets the Terraform schema for the data source.
func (d *rulesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
		},
	}
}

// rulesDataSourceModel represents the Terraform data source model for mSCP rules.
type rulesDataSourceModel struct {
	BaselineID types.String  `tfsdk:"baseline_id"`
	Sources    []sourceModel `tfsdk:"sources"`
	Rules      []ruleModel   `tfsdk:"rules"`
}

// sourceModel represents a source branch and revision for a rule.
type sourceModel struct {
	Branch   types.String `tfsdk:"branch"`
	Revision types.String `tfsdk:"revision"`
}

// ruleModel represents a rule in the data source, including ODV and computed fields.
type ruleModel struct {
	ID                 types.String                       `tfsdk:"id"`
	SectionName        types.String                       `tfsdk:"section_name"`
	Enabled            types.Bool                         `tfsdk:"enabled"`
	Title              types.String                       `tfsdk:"title"`
	Description        types.String                       `tfsdk:"description"`
	References         []types.String                     `tfsdk:"references"`
	ODV                *odvModel                          `tfsdk:"odv"`
	SupportedOS        []osInfoModel                      `tfsdk:"supported_os"`
	OSSpecificDefaults map[string]osSpecificRuleInfoModel `tfsdk:"os_specific_defaults"`
	RuleRelation       *ruleRelationModel                 `tfsdk:"rule_relation"`
}

// odvModel represents the Organization Defined Value (ODV) for a rule, including its value, hint, placeholder, type, and validation.
type odvModel struct {
	Value       types.String                `tfsdk:"value"`
	Hint        types.String                `tfsdk:"hint"`
	Placeholder types.String                `tfsdk:"placeholder"`
	Type        types.String                `tfsdk:"type"`
	Validation  *validationConstraintsModel `tfsdk:"validation"`
}

// validationConstraintsModel represents validation constraints for an ODV field.
type validationConstraintsModel struct {
	Min        types.Int64    `tfsdk:"min"`
	Max        types.Int64    `tfsdk:"max"`
	EnumValues []types.String `tfsdk:"enum_values"`
	Regex      types.String   `tfsdk:"regex"`
}

// osInfoModel represents supported OS information for a rule.
type osInfoModel struct {
	OSType         types.String `tfsdk:"os_type"`
	OSVersion      types.Int64  `tfsdk:"os_version"`
	ManagementType types.String `tfsdk:"management_type"`
}

// osSpecificRuleInfoModel represents OS-specific rule information, including ODV recommendations.
type osSpecificRuleInfoModel struct {
	Title       types.String            `tfsdk:"title"`
	Description types.String            `tfsdk:"description"`
	ODV         *odvRecommendationModel `tfsdk:"odv"`
}

// odvRecommendationModel represents a recommended ODV value and hint for a specific OS.
type odvRecommendationModel struct {
	Value types.String `tfsdk:"value"`
	Hint  types.String `tfsdk:"hint"`
}

// ruleRelationModel represents rule dependencies for a rule.
type ruleRelationModel struct {
	DependsOn []types.String `tfsdk:"depends_on"`
}

// Read implements datasource.DataSource for rulesDataSource. It fetches the list of rules from the API and sets the state.
func (d *rulesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config rulesDataSourceModel
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

	rulesResp, err := d.client.GetCBEngineRules(ctx, config.BaselineID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get rules",
			err.Error(),
		)
		return
	}

	var sources []sourceModel
	for _, s := range rulesResp.Sources {
		sources = append(sources, sourceModel{
			Branch:   types.StringValue(s.Branch),
			Revision: types.StringValue(s.Revision),
		})
	}

	var rules []ruleModel
	for _, r := range rulesResp.Rules {
		var references []types.String
		for _, ref := range r.References {
			references = append(references, types.StringValue(ref))
		}

		var odv *odvModel
		if r.ODV != nil {
			var validation *validationConstraintsModel
			if r.ODV.Validation != nil {
				var enumValues []types.String
				for _, v := range r.ODV.Validation.EnumValues {
					enumValues = append(enumValues, types.StringValue(v))
				}
				validation = &validationConstraintsModel{
					Min:        intToInt64Ptr(r.ODV.Validation.Min),
					Max:        intToInt64Ptr(r.ODV.Validation.Max),
					EnumValues: enumValues,
					Regex:      types.StringValue(r.ODV.Validation.Regex),
				}
			}
			odv = &odvModel{
				Value:       types.StringValue(r.ODV.Value),
				Hint:        types.StringValue(r.ODV.Hint),
				Placeholder: types.StringValue(r.ODV.Placeholder),
				Type:        types.StringValue(r.ODV.Type),
				Validation:  validation,
			}
		}

		var supportedOS []osInfoModel
		for _, os := range r.SupportedOS {
			supportedOS = append(supportedOS, osInfoModel{
				OSType:         types.StringValue(os.OSType),
				OSVersion:      types.Int64Value(int64(os.OSVersion)),
				ManagementType: types.StringValue(os.ManagementType),
			})
		}

		osSpecificDefaults := make(map[string]osSpecificRuleInfoModel)
		for k, v := range r.OSSpecificDefaults {
			var odvRec *odvRecommendationModel
			if v.ODV != nil {
				odvRec = &odvRecommendationModel{
					Value: types.StringValue(v.ODV.Value),
					Hint:  types.StringValue(v.ODV.Hint),
				}
			}
			osSpecificDefaults[k] = osSpecificRuleInfoModel{
				Title:       types.StringValue(v.Title),
				Description: types.StringValue(v.Description),
				ODV:         odvRec,
			}
		}

		var ruleRelation *ruleRelationModel
		if r.RuleRelation != nil {
			var dependsOn []types.String
			for _, dep := range r.RuleRelation.DependsOn {
				dependsOn = append(dependsOn, types.StringValue(dep))
			}
			ruleRelation = &ruleRelationModel{
				DependsOn: dependsOn,
			}
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

	state := rulesDataSourceModel{
		BaselineID: config.BaselineID,
		Sources:    sources,
		Rules:      rules,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Helper for *int to types.Int64
func intToInt64Ptr(i *int) types.Int64 {
	if i == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*i))
}
