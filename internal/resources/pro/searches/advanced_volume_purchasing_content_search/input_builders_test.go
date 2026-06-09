// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package advanced_volume_purchasing_content_search

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/criteria"
)

func TestBuildInput_AlwaysEmitsCriteriaAndDisplayFields(t *testing.T) {
	// No criteria, null display_fields: both arrays must still be emitted
	// (non-nil, empty) so the full-replace PUT clears them rather than leaving
	// the server values untouched.
	plan := AdvancedVolumePurchasingContentSearchResourceModel{
		Name:          types.StringValue("empty search"),
		SiteID:        types.StringValue("-1"),
		Criteria:      types.ListNull(criteria.CriterionObjectType()),
		DisplayFields: types.SetNull(types.StringType),
	}

	out, diags := buildAdvancedVolumePurchasingContentSearchInput(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	if out.Criteria == nil {
		t.Fatal("criteria array must always be emitted (non-nil)")
	}
	if len(*out.Criteria) != 0 {
		t.Errorf("expected empty criterion slice, got %d", len(*out.Criteria))
	}
	if out.DisplayFields == nil {
		t.Fatal("display_fields array must always be emitted (non-nil)")
	}
	if len(*out.DisplayFields) != 0 {
		t.Errorf("expected empty display field slice, got %d", len(*out.DisplayFields))
	}
	if out.SiteID == nil || *out.SiteID != "-1" {
		t.Errorf("site sentinel -1 expected, got %v", out.SiteID)
	}
	if out.Name != "empty search" {
		t.Errorf("name mismatch: %q", out.Name)
	}
}

func TestBuildInput_PopulatedDisplayFieldsAndCriteria(t *testing.T) {
	df, d := types.SetValueFrom(context.Background(), types.StringType, []string{"Name", "Cost"})
	if d.HasError() {
		t.Fatalf("set value: %v", d)
	}
	critList, cd := criteria.CriteriaListValue(context.Background(), []criteria.CriterionModel{
		{Name: types.StringValue("Name"), SearchType: types.StringValue("is"), Value: types.StringValue("Office")},
	})
	if cd.HasError() {
		t.Fatalf("criteria list: %v", cd)
	}
	plan := AdvancedVolumePurchasingContentSearchResourceModel{
		Name:          types.StringValue("unmanaged"),
		SiteID:        types.StringValue("-1"),
		Criteria:      critList,
		DisplayFields: df,
	}
	out, diags := buildAdvancedVolumePurchasingContentSearchInput(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	if len(*out.Criteria) != 1 {
		t.Errorf("expected 1 criterion, got %d", len(*out.Criteria))
	}
	// priority filled from index when omitted.
	if (*out.Criteria)[0].Priority == nil || *(*out.Criteria)[0].Priority != 0 {
		t.Errorf("priority should default to index 0, got %v", (*out.Criteria)[0].Priority)
	}
	if len(*out.DisplayFields) != 2 {
		t.Errorf("expected 2 display fields, got %d", len(*out.DisplayFields))
	}
}
