// Copyright 2025 Jamf Software LLC.

package baselines

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// BaselinesDataSource implements the Terraform data source for mSCP baselines.
type BaselinesDataSource struct {
	client *client.Client
}

// BaselinesDataSourceModel represents the Terraform data source model for mSCP baselines.
type BaselinesDataSourceModel struct {
	Baselines []BaselineModel `tfsdk:"baselines"`
}

// BaselineModel represents a single mSCP baseline in the data source.
type BaselineModel struct {
	ID          types.String `tfsdk:"id"`
	BaselineID  types.String `tfsdk:"baseline_id"`
	Title       types.String `tfsdk:"title"`
	Description types.String `tfsdk:"description"`
	RuleCount   types.Int64  `tfsdk:"rule_count"`
}
