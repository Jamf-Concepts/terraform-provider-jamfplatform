// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package content_categories

import (
	datasourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ContentCategoriesDataSourceModel represents the Terraform data source model for
// the Jamf-curated content category catalogue.
type ContentCategoriesDataSourceModel struct {
	ID                types.String                 `tfsdk:"id"`
	ContentCategories []ContentCategoryResultModel `tfsdk:"content_categories"`
	Timeouts          datasourceTimeouts.Value     `tfsdk:"timeouts"`
}

// ContentCategoryResultModel represents a single content category in the results.
type ContentCategoryResultModel struct {
	ID          types.String `tfsdk:"id"`
	DisplayName types.String `tfsdk:"display_name"`
	Name        types.String `tfsdk:"name"`
}
