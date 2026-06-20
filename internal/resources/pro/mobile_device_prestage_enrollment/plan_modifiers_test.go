// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_prestage_enrollment

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// --- listUseStateForUnknown -------------------------------------------------

func newAttrList(t *testing.T, in []string) types.List {
	t.Helper()
	elems := make([]attr.Value, 0, len(in))
	for _, s := range in {
		elems = append(elems, types.StringValue(s))
	}
	l, diags := types.ListValue(types.StringType, elems)
	if diags.HasError() {
		t.Fatalf("list construction: %v", diags)
	}
	return l
}

func TestListUseStateForUnknown_CopiesStateWhenPlanUnknown(t *testing.T) {
	state := newAttrList(t, []string{"cert-A"})
	req := planmodifier.ListRequest{
		StateValue:  state,
		PlanValue:   types.ListUnknown(types.StringType),
		ConfigValue: types.ListNull(types.StringType), // null ≠ unknown → copy fires
	}
	resp := &planmodifier.ListResponse{PlanValue: req.PlanValue}
	listUseStateForUnknown().PlanModifyList(context.Background(), req, resp)

	if resp.PlanValue.IsUnknown() {
		t.Fatalf("plan should have been replaced with prior state, still Unknown")
	}
	if !resp.PlanValue.Equal(state) {
		t.Errorf("plan should equal prior state, got %v", resp.PlanValue)
	}
}

func TestListUseStateForUnknown_NoCopyWhenStateNull(t *testing.T) {
	req := planmodifier.ListRequest{
		StateValue:  types.ListNull(types.StringType),
		PlanValue:   types.ListUnknown(types.StringType),
		ConfigValue: types.ListNull(types.StringType),
	}
	resp := &planmodifier.ListResponse{PlanValue: req.PlanValue}
	listUseStateForUnknown().PlanModifyList(context.Background(), req, resp)

	if !resp.PlanValue.IsUnknown() {
		t.Errorf("null prior state must leave plan Unknown, got %v", resp.PlanValue)
	}
}

func TestListUseStateForUnknown_NoCopyWhenPlanKnown(t *testing.T) {
	state := newAttrList(t, []string{"cert-A"})
	plan := newAttrList(t, []string{"cert-B"})
	req := planmodifier.ListRequest{
		StateValue:  state,
		PlanValue:   plan,
		ConfigValue: plan,
	}
	resp := &planmodifier.ListResponse{PlanValue: req.PlanValue}
	listUseStateForUnknown().PlanModifyList(context.Background(), req, resp)

	if !resp.PlanValue.Equal(plan) {
		t.Errorf("known plan must be left untouched, got %v", resp.PlanValue)
	}
}
