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

// sourceModel represents a source branch and revision for a benchmark.
type sourceModel struct {
	Branch   types.String `tfsdk:"branch"`
	Revision types.String `tfsdk:"revision"`
}

// targetModel represents the device groups targeted by the benchmark.
type targetModel struct {
	DeviceGroups []types.String `tfsdk:"device_groups"`
}

// benchmarkResourceModel represents the Terraform resource model for a Jamf Compliance Benchmark.
type benchmarkResourceModel struct {
	ID               types.String  `tfsdk:"id"`
	Title            types.String  `tfsdk:"title"`
	Description      types.String  `tfsdk:"description"`
	SourceBaselineID types.String  `tfsdk:"source_baseline_id"`
	Sources          []sourceModel `tfsdk:"sources"`
	Rules            []ruleModel   `tfsdk:"rules"`
	Target           *targetModel  `tfsdk:"target"`
	EnforcementMode  types.String  `tfsdk:"enforcement_mode"`
	TenantID         types.String  `tfsdk:"tenant_id"`
	Deleted          types.Bool    `tfsdk:"deleted"`
	UpdateAvailable  types.Bool    `tfsdk:"update_available"`
	LastUpdatedAt    types.String  `tfsdk:"last_updated_at"`
}

// benchmarkDataSource implements the Terraform data source for Jamf Compliance Benchmarks.
type benchmarkDataSource struct {
	client *client.Client
}

// benchmarkDataSourceModel represents the Terraform data source model for a Jamf Compliance Benchmark.
type benchmarkDataSourceModel struct {
	ID              types.String  `tfsdk:"id"`
	Title           types.String  `tfsdk:"title"`
	BenchmarkID     types.String  `tfsdk:"benchmark_id"`
	TenantID        types.String  `tfsdk:"tenant_id"`
	Description     types.String  `tfsdk:"description"`
	Sources         []sourceModel `tfsdk:"sources"`
	Rules           []ruleModel   `tfsdk:"rules"`
	Target          *targetModel  `tfsdk:"target"`
	EnforcementMode types.String  `tfsdk:"enforcement_mode"`
	Deleted         types.Bool    `tfsdk:"deleted"`
	UpdateAvailable types.Bool    `tfsdk:"update_available"`
	LastUpdatedAt   types.String  `tfsdk:"last_updated_at"`
}

// ruleModel represents a rule in the benchmark, including ODV and computed fields.
type ruleModel struct {
	ID                 types.String `tfsdk:"id"`
	SectionName        types.String `tfsdk:"section_name"`
	Enabled            types.Bool   `tfsdk:"enabled"`
	Title              types.String `tfsdk:"title"`
	Description        types.String `tfsdk:"description"`
	References         types.List   `tfsdk:"references"`
	ODV                types.Object `tfsdk:"odv"`
	SupportedOS        types.List   `tfsdk:"supported_os"`
	OSSpecificDefaults types.Map    `tfsdk:"os_specific_defaults"`
	RuleRelation       types.Object `tfsdk:"rule_relation"`
}
