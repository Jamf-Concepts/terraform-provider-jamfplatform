// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package advanced_user_search

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/criteria"
)

func TestBuildInput_AlwaysEmitsCriteriaAndDisplayWrappers(t *testing.T) {
	plan := AdvancedUserSearchResourceModel{
		Name:          types.StringValue("empty user search"),
		SiteID:        types.StringValue("-1"),
		Criteria:      nil,
		DisplayFields: types.SetNull(types.StringType),
	}
	out, diags := buildAdvancedUserSearchInput(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	if out.Criteria == nil || out.Criteria.Criterion == nil || len(*out.Criteria.Criterion) != 0 {
		t.Errorf("criteria wrapper must be emitted empty, got %v", out.Criteria)
	}
	if out.DisplayFields == nil || out.DisplayFields.DisplayField == nil || len(*out.DisplayFields.DisplayField) != 0 {
		t.Errorf("display_fields wrapper must be emitted empty, got %v", out.DisplayFields)
	}
	if out.Site == nil || out.Site.ID == nil || *out.Site.ID != -1 {
		t.Errorf("site sentinel -1 expected, got %v", out.Site)
	}
}

func TestBuildInput_PopulatedCriteriaAndDisplay(t *testing.T) {
	df, d := types.SetValueFrom(context.Background(), types.StringType, []string{"Full Name", "Email Address"})
	if d.HasError() {
		t.Fatalf("set value: %v", d)
	}
	plan := AdvancedUserSearchResourceModel{
		Name:   types.StringValue("vips"),
		SiteID: types.StringValue("-1"),
		Criteria: []criteria.CriterionModel{
			{Name: types.StringValue("Full Name"), SearchType: types.StringValue("like"), Value: types.StringValue("a")},
		},
		DisplayFields: df,
	}
	out, diags := buildAdvancedUserSearchInput(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	if len(*out.Criteria.Criterion) != 1 {
		t.Errorf("expected 1 criterion, got %d", len(*out.Criteria.Criterion))
	}
	if len(*out.DisplayFields.DisplayField) != 2 {
		t.Errorf("expected 2 display fields, got %d", len(*out.DisplayFields.DisplayField))
	}
}

func TestBuildSiteObject_NoneSentinel(t *testing.T) {
	if got := buildSiteObject(types.StringValue("-1")); got == nil || got.ID == nil || *got.ID != -1 {
		t.Errorf("expected site id -1, got %v", got)
	}
	if got := buildSiteObject(types.StringNull()); got != nil {
		t.Errorf("null site_id should produce nil SiteObject, got %v", got)
	}
}
