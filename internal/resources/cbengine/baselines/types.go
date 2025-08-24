package baselines

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// baselinesDataSource implements the Terraform data source for mSCP baselines.
type baselinesDataSource struct {
	client *client.Client
}

// baselinesDataSourceModel represents the Terraform data source model for mSCP baselines.
type baselinesDataSourceModel struct {
	Baselines []baselineModel `tfsdk:"baselines"`
}

// baselineModel represents a single mSCP baseline in the data source.
type baselineModel struct {
	ID          types.String `tfsdk:"id"`
	BaselineID  types.String `tfsdk:"baseline_id"`
	Title       types.String `tfsdk:"title"`
	Description types.String `tfsdk:"description"`
	RuleCount   types.Int64  `tfsdk:"rule_count"`
}
