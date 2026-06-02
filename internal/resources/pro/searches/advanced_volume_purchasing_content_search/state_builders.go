// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package advanced_volume_purchasing_content_search

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/criteria"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// assignAdvancedVolumePurchasingContentSearchResourceModel populates a resource model from
// an AdvancedSearch response. Server is authoritative for every field. Always
// sourced from a GET after Create/Update — the Pro PUT response echoes the
// submitted body (including invalid display fields the server silently drops),
// so the canonical representation must come from a fresh GET.
func assignAdvancedVolumePurchasingContentSearchResourceModel(ctx context.Context, state *AdvancedVolumePurchasingContentSearchResourceModel, search *pro.AdvancedUserContentSearch) diag.Diagnostics {
	var diags diag.Diagnostics
	if search == nil {
		return diags
	}

	state.ID = helpers.StringPointerValueOrNull(search.ID)
	state.Name = types.StringValue(search.Name)
	state.SiteID = helpers.StringPointerValueOrNull(search.SiteID)
	state.Criteria = criteria.FlattenSmartSearchCriteria(search.Criteria)

	displayFields, dfDiags := flattenDisplayFields(ctx, search.DisplayFields)
	diags.Append(dfDiags...)
	if diags.HasError() {
		return diags
	}
	state.DisplayFields = displayFields

	return diags
}

// assignAdvancedVolumePurchasingContentSearchDataSourceModel populates a data source model
// from an AdvancedSearch response. Symmetric with the resource builder.
func assignAdvancedVolumePurchasingContentSearchDataSourceModel(ctx context.Context, state *AdvancedVolumePurchasingContentSearchDataSourceModel, search *pro.AdvancedUserContentSearch) diag.Diagnostics {
	var diags diag.Diagnostics
	if search == nil {
		return diags
	}

	state.ID = helpers.StringPointerValueOrNull(search.ID)
	state.Name = types.StringValue(search.Name)
	state.SiteID = helpers.StringPointerValueOrNull(search.SiteID)
	state.Criteria = criteria.FlattenSmartSearchCriteria(search.Criteria)

	displayFields, dfDiags := flattenDisplayFields(ctx, search.DisplayFields)
	diags.Append(dfDiags...)
	if diags.HasError() {
		return diags
	}
	state.DisplayFields = displayFields

	return diags
}

// flattenDisplayFields converts the SDK display-fields slice into a Set of column
// names. Returns a null set for an empty or absent slice. Modelled as a Set
// because Jamf Pro returns the columns in its own canonical order, not the order
// they were submitted.
func flattenDisplayFields(ctx context.Context, src *[]string) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	if src == nil || len(*src) == 0 {
		return types.SetNull(types.StringType), diags
	}
	set, d := types.SetValueFrom(ctx, types.StringType, *src)
	diags.Append(d...)
	return set, diags
}
