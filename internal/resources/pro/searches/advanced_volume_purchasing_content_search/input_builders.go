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

// buildAdvancedVolumePurchasingContentSearchInput converts a plan model into the SDK
// payload used for Create and Update. criteria and displayFields are emitted
// unconditionally as non-nil slices: the Pro /v1 advanced-search PUT is a full
// replace (an omitted or empty array clears the corresponding collection), so
// always sending the full representation makes the Terraform config
// authoritative and lets users clear criteria and display fields.
func buildAdvancedVolumePurchasingContentSearchInput(ctx context.Context, plan AdvancedVolumePurchasingContentSearchResourceModel) (*pro.AdvancedUserContentSearch, diag.Diagnostics) {
	var diags diag.Diagnostics

	displayFields, dfDiags := buildDisplayFields(ctx, plan.DisplayFields)
	diags.Append(dfDiags...)
	if diags.HasError() {
		return nil, diags
	}

	criteriaSlice := criteria.BuildSmartSearchCriteria(plan.Criteria)
	siteID := plan.SiteID.ValueString()

	search := &pro.AdvancedUserContentSearch{
		Name:          plan.Name.ValueString(),
		SiteID:        &siteID,
		Criteria:      &criteriaSlice,
		DisplayFields: &displayFields,
	}

	return search, diags
}

// buildDisplayFields converts the plan display_fields set into the SDK string
// slice. Always returns a non-nil slice (empty when null/unknown) so the
// `displayFields` array is always emitted; an empty array clears all columns
// on the full-replace PUT.
func buildDisplayFields(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	names := []string{}
	if !set.IsNull() && !set.IsUnknown() {
		extracted, d := helpers.SetToStringSlice(ctx, set)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		names = extracted
	}
	return names, diags
}
