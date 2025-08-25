// Copyright 2025 Jamf Software LLC.

package benchmark

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// BenchmarkResource implements the Terraform resource for Jamf Compliance Benchmark.
type BenchmarkResource struct {
	client *client.Client
}

// SourceModel represents a source branch and revision for a benchmark.
type SourceModel struct {
	Branch   types.String `tfsdk:"branch"`
	Revision types.String `tfsdk:"revision"`
}

// BenchmarkResourceModel represents the Terraform resource model for a Jamf Compliance Benchmark.
type BenchmarkResourceModel struct {
	ID                types.String  `tfsdk:"id"`
	Title             types.String  `tfsdk:"title"`
	Description       types.String  `tfsdk:"description"`
	SourceBaselineID  types.String  `tfsdk:"source_baseline_id"`
	Sources           []SourceModel `tfsdk:"sources"`
	Rules             []RuleModel   `tfsdk:"rules"`
	TargetDeviceGroup types.String  `tfsdk:"target_device_group"`
	EnforcementMode   types.String  `tfsdk:"enforcement_mode"`
	TenantID          types.String  `tfsdk:"tenant_id"`
	Deleted           types.Bool    `tfsdk:"deleted"`
	UpdateAvailable   types.Bool    `tfsdk:"update_available"`
	LastUpdatedAt     types.String  `tfsdk:"last_updated_at"`
}

// BenchmarkDataSource implements the Terraform data source for Jamf Compliance Benchmarks.
type BenchmarkDataSource struct {
	client *client.Client
}

// BenchmarkDataSourceModel represents the Terraform data source model for a Jamf Compliance Benchmark.
type BenchmarkDataSourceModel struct {
	ID                types.String  `tfsdk:"id"`
	Title             types.String  `tfsdk:"title"`
	BenchmarkID       types.String  `tfsdk:"benchmark_id"`
	TenantID          types.String  `tfsdk:"tenant_id"`
	Description       types.String  `tfsdk:"description"`
	Sources           []SourceModel `tfsdk:"sources"`
	Rules             []RuleModel   `tfsdk:"rules"`
	TargetDeviceGroup types.String  `tfsdk:"target_device_group"`
	EnforcementMode   types.String  `tfsdk:"enforcement_mode"`
	Deleted           types.Bool    `tfsdk:"deleted"`
	UpdateAvailable   types.Bool    `tfsdk:"update_available"`
	LastUpdatedAt     types.String  `tfsdk:"last_updated_at"`
}

// RuleModel represents a rule in the benchmark, including ODV and computed fields.
type RuleModel struct {
	ID                      types.String `tfsdk:"id"`
	SectionName             types.String `tfsdk:"section_name"`
	Enabled                 types.Bool   `tfsdk:"enabled"`
	Title                   types.String `tfsdk:"title"`
	Description             types.String `tfsdk:"description"`
	References              types.List   `tfsdk:"references"`
	ODVValue                types.String `tfsdk:"odv_value"`
	ODVHint                 types.String `tfsdk:"odv_hint"`
	ODVPlaceholder          types.String `tfsdk:"odv_placeholder"`
	ODVType                 types.String `tfsdk:"odv_type"`
	ODVValidationMin        types.Int64  `tfsdk:"odv_validation_min"`
	ODVValidationMax        types.Int64  `tfsdk:"odv_validation_max"`
	ODVValidationEnumValues types.List   `tfsdk:"odv_validation_enum_values"`
	ODVValidationRegex      types.String `tfsdk:"odv_validation_regex"`
	SupportedOS             types.List   `tfsdk:"supported_os"`
	OSSpecificDefaults      types.Map    `tfsdk:"os_specific_defaults"`
	DependsOn               types.List   `tfsdk:"depends_on"`
}
