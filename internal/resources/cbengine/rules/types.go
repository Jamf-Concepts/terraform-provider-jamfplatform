// Copyright 2025 Jamf Software LLC.

package rules

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// RulesDataSource implements the Terraform data source for mSCP rules.
type RulesDataSource struct {
	client *client.Client
}

// RulesDataSourceModel represents the Terraform data source model for mSCP rules.
type RulesDataSourceModel struct {
	BaselineID types.String  `tfsdk:"baseline_id"`
	Sources    []SourceModel `tfsdk:"sources"`
	Rules      []RuleModel   `tfsdk:"rules"`
}

// SourceModel represents a source branch and revision for a rule.
type SourceModel struct {
	Branch   types.String `tfsdk:"branch"`
	Revision types.String `tfsdk:"revision"`
}

// RuleModel represents a rule in the data source, including ODV and computed fields.
type RuleModel struct {
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
	SupportedOS             []OSInfoModel  `tfsdk:"supported_os"`
	OSSpecificDefaults      types.Map      `tfsdk:"os_specific_defaults"`
	DependsOn               []types.String `tfsdk:"depends_on"`
}

// OSInfoModel represents supported OS information for a rule.
type OSInfoModel struct {
	OSType         types.String `tfsdk:"os_type"`
	OSVersion      types.Int64  `tfsdk:"os_version"`
	ManagementType types.String `tfsdk:"management_type"`
}
