// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package criteria

import (
	"cmp"
	"slices"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// CriterionModel is the Terraform model for a single classic <criterion>,
// matching the attribute map returned by CriterionAttributes. Shared by the
// advanced-search resources (and available to any future ProClassic resource
// exposing the classic criterion element).
type CriterionModel struct {
	Priority              types.Int64  `tfsdk:"priority"`
	Name                  types.String `tfsdk:"name"`
	SearchType            types.String `tfsdk:"search_type"`
	Value                 types.String `tfsdk:"value"`
	AndOr                 types.String `tfsdk:"and_or"`
	HasOpeningParenthesis types.Bool   `tfsdk:"has_opening_parenthesis"`
	HasClosingParenthesis types.Bool   `tfsdk:"has_closing_parenthesis"`
}

// BuildCriterionSlice maps plan criterion models to SDK criteria. Priority is
// filled from the element index when omitted; and_or defaults to "and";
// parentheses default to false. Output is sorted by priority so the wire order
// matches the user's list order.
//
// The result is always non-nil (an empty slice for empty input) so callers can
// wrap it in an always-emitted <criteria> element. The classic API treats an
// omitted wrapper as "leave unchanged" but an empty wrapper as "clear all" —
// so emitting the empty wrapper is what lets a resource remove every criterion.
func BuildCriterionSlice(models []CriterionModel) []proclassic.Criterion {
	out := make([]proclassic.Criterion, 0, len(models))
	for idx, c := range models {
		priority := idx
		if !c.Priority.IsNull() && !c.Priority.IsUnknown() {
			priority = int(c.Priority.ValueInt64())
		}
		andOr := "and"
		if !c.AndOr.IsNull() && !c.AndOr.IsUnknown() && c.AndOr.ValueString() != "" {
			andOr = c.AndOr.ValueString()
		}
		opening := false
		if !c.HasOpeningParenthesis.IsNull() && !c.HasOpeningParenthesis.IsUnknown() {
			opening = c.HasOpeningParenthesis.ValueBool()
		}
		closing := false
		if !c.HasClosingParenthesis.IsNull() && !c.HasClosingParenthesis.IsUnknown() {
			closing = c.HasClosingParenthesis.ValueBool()
		}

		name := c.Name.ValueString()
		searchType := c.SearchType.ValueString()
		value := c.Value.ValueString()

		out = append(out, proclassic.Criterion{
			Name:         &name,
			Priority:     &priority,
			AndOr:        &andOr,
			SearchType:   &searchType,
			Value:        &value,
			OpeningParen: &opening,
			ClosingParen: &closing,
		})
	}
	slices.SortStableFunc(out, func(a, b proclassic.Criterion) int {
		if a.Priority == nil || b.Priority == nil {
			return 0
		}
		return cmp.Compare(*a.Priority, *b.Priority)
	})
	return out
}

// FlattenCriterionSlice maps an SDK criteria slice back to plan models. Server
// is authoritative for every field, so values are copied directly (no Reconcile
// helpers — those return null when the API value is empty and prior state was
// null, which diverges between the import path and the post-apply refresh path).
// Returns nil for an empty or absent slice (a null list in state).
func FlattenCriterionSlice(src *[]proclassic.Criterion) []CriterionModel {
	if src == nil || len(*src) == 0 {
		return nil
	}
	in := *src
	out := make([]CriterionModel, len(in))
	for i, c := range in {
		priority := types.Int64Null()
		if c.Priority != nil {
			priority = types.Int64Value(int64(*c.Priority))
		}
		out[i] = CriterionModel{
			Priority:              priority,
			Name:                  helpers.StringPointerValueOrNull(c.Name),
			SearchType:            helpers.StringPointerValueOrNull(c.SearchType),
			Value:                 helpers.StringPointerValueOrNull(c.Value),
			AndOr:                 helpers.StringPointerValueOrNull(c.AndOr),
			HasOpeningParenthesis: helpers.BoolPointerValueOrNull(c.OpeningParen),
			HasClosingParenthesis: helpers.BoolPointerValueOrNull(c.ClosingParen),
		}
	}
	return out
}
