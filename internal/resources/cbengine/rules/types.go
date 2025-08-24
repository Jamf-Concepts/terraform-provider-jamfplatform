package rules

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// rulesDataSource implements the Terraform data source for mSCP rules.
type rulesDataSource struct {
	client *client.Client
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
	ID                      types.String   `tfsdk:"id"`
	SectionName             types.String   `tfsdk:"section_name"`
	Enabled                 types.Bool     `tfsdk:"enabled"`
	Title                   types.String   `tfsdk:"title"`
	Description             types.String   `tfsdk:"description"`
	References              []types.String `tfsdk:"references"`
	ODVValue                types.String   `tfsdk:"odv_value"`
	ODVHint                 types.String   `tfsdk:"odv_hint"`
	ODVPlaceholder          types.String   `tfsdk:"odv_placeholder"`
	ODVType                 types.String   `tfsdk:"odv_type"`
	ODVValidationMin        types.Int64    `tfsdk:"odv_validation_min"`
	ODVValidationMax        types.Int64    `tfsdk:"odv_validation_max"`
	ODVValidationEnumValues []types.String `tfsdk:"odv_validation_enum_values"`
	ODVValidationRegex      types.String   `tfsdk:"odv_validation_regex"`
	SupportedOS             []osInfoModel  `tfsdk:"supported_os"`
	OSSpecificDefaults      types.Map      `tfsdk:"os_specific_defaults"`
	DependsOn               []types.String `tfsdk:"depends_on"`
}

// osInfoModel represents supported OS information for a rule.
type osInfoModel struct {
	OSType         types.String `tfsdk:"os_type"`
	OSVersion      types.Int64  `tfsdk:"os_version"`
	ManagementType types.String `tfsdk:"management_type"`
}
