// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package category

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// assignCategoryResourceModel populates a resource model from a Category response.
func assignCategoryResourceModel(state *CategoryResourceModel, c *pro.Category) {
	state.ID = helpers.StringPointerValueOrNull(c.ID)
	state.Name = types.StringValue(c.Name)
	state.Priority = types.Int64Value(int64(c.Priority))
}

// assignCategoryDataSourceModel populates a data source model from a Category response.
func assignCategoryDataSourceModel(state *CategoryDataSourceModel, c *pro.Category) {
	state.ID = helpers.StringPointerValueOrNull(c.ID)
	state.Name = types.StringValue(c.Name)
	state.Priority = types.Int64Value(int64(c.Priority))
}
