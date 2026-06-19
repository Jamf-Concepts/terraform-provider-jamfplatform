// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package advanced_computer_search

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/criteria"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

func TestBuildInput_AlwaysEmitsCriteriaAndDisplayWrappers(t *testing.T) {
	// No criteria, null display_fields: both wrappers must still be emitted
	// (non-nil, empty) so the classic PUT clears them rather than leaving the
	// server values untouched.
	plan := AdvancedComputerSearchResourceModel{
		Name:          types.StringValue("empty search"),
		SiteID:        types.StringValue("-1"),
		Criteria:      nil,
		DisplayFields: types.SetNull(types.StringType),
	}

	out, diags := buildAdvancedComputerSearchInput(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	if out.Criteria == nil || out.Criteria.Criterion == nil {
		t.Fatal("criteria wrapper must always be emitted (non-nil)")
	}
	if len(*out.Criteria.Criterion) != 0 {
		t.Errorf("expected empty criterion slice, got %d", len(*out.Criteria.Criterion))
	}
	if out.DisplayFields == nil || out.DisplayFields.DisplayField == nil {
		t.Fatal("display_fields wrapper must always be emitted (non-nil)")
	}
	if len(*out.DisplayFields.DisplayField) != 0 {
		t.Errorf("expected empty display_field slice, got %d", len(*out.DisplayFields.DisplayField))
	}
	// view_as and sort_1/2/3 are not modelled, so they must not be sent — the
	// server keeps its defaults.
	if out.ViewAs != nil {
		t.Errorf("view_as must not be sent, got %v", *out.ViewAs)
	}
	if out.Sort1 != nil || out.Sort2 != nil || out.Sort3 != nil {
		t.Errorf("sort fields must not be sent")
	}
	if out.Site == nil || out.Site.ID == nil || *out.Site.ID != -1 {
		t.Errorf("site sentinel -1 expected, got %v", out.Site)
	}
}

func TestBuildInput_PopulatedDisplayFieldsAndCriteria(t *testing.T) {
	df, d := types.SetValueFrom(context.Background(), types.StringType, []string{"Computer Name", "Serial Number"})
	if d.HasError() {
		t.Fatalf("set value: %v", d)
	}
	plan := AdvancedComputerSearchResourceModel{
		Name:   types.StringValue("lab macs"),
		SiteID: types.StringValue("-1"),
		Criteria: []criteria.CriterionModel{
			{Name: types.StringValue("Computer Name"), SearchType: types.StringValue("like"), Value: types.StringValue("lab")},
		},
		DisplayFields: df,
	}
	out, diags := buildAdvancedComputerSearchInput(context.Background(), plan)
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
	if got := scope.BuildSiteObject(types.StringValue("-1")); got == nil || got.ID == nil || *got.ID != -1 {
		t.Errorf("expected site id -1, got %v", got)
	}
	if got := scope.BuildSiteObject(types.StringNull()); got != nil {
		t.Errorf("null site_id should produce nil SiteObject, got %v", got)
	}
}
