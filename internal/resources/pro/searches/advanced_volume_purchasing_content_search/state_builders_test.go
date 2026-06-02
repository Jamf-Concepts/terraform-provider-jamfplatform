// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package advanced_volume_purchasing_content_search

import (
	"context"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenDisplayFields_NilAndEmptyToNullSet(t *testing.T) {
	got, diags := flattenDisplayFields(context.Background(), nil)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if !got.IsNull() {
		t.Errorf("nil slice should flatten to null set")
	}

	empty := []string{}
	got, _ = flattenDisplayFields(context.Background(), &empty)
	if !got.IsNull() {
		t.Errorf("empty slice should flatten to null set")
	}
}

func TestFlattenDisplayFields_Populated(t *testing.T) {
	src := []string{"Name", "Cost"}
	got, diags := flattenDisplayFields(context.Background(), &src)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if got.IsNull() {
		t.Fatal("expected non-null set")
	}
	if n := len(got.Elements()); n != 2 {
		t.Errorf("expected 2 display fields, got %d", n)
	}
}

func TestAssignResourceModel_CopiesFieldsNoSiteName(t *testing.T) {
	id := "478"
	site := "-1"
	pri := 0
	search := &pro.AdvancedUserContentSearch{
		ID:     &id,
		Name:   "unmanaged",
		SiteID: &site,
		Criteria: &[]pro.SmartSearchCriterion{
			{Name: "Name", SearchType: "is", Value: "Office", AndOr: "and", Priority: &pri},
		},
		DisplayFields: &[]string{"Name"},
	}

	state := &AdvancedVolumePurchasingContentSearchResourceModel{SiteID: types.StringValue("-1")}
	diags := assignAdvancedVolumePurchasingContentSearchResourceModel(context.Background(), state, search)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if state.ID.ValueString() != "478" {
		t.Errorf("id mismatch: %q", state.ID.ValueString())
	}
	if state.Name.ValueString() != "unmanaged" {
		t.Errorf("name mismatch: %q", state.Name.ValueString())
	}
	if state.SiteID.ValueString() != "-1" {
		t.Errorf("site_id mismatch: %q", state.SiteID.ValueString())
	}
	if len(state.Criteria) != 1 {
		t.Errorf("expected 1 criterion, got %d", len(state.Criteria))
	}
	if state.DisplayFields.IsNull() || len(state.DisplayFields.Elements()) != 1 {
		t.Errorf("expected 1 display field, got %v", state.DisplayFields)
	}
}
