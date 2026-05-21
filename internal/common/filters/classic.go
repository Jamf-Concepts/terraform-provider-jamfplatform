// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package filters

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// classicFilterBlockDescription is the user-facing markdown description for the
// classic filter block. Classic APIs do not accept query parameters on their
// List endpoints, so this filter is applied client-side after the full list is
// fetched. It is intentionally narrow: a single case-insensitive substring match
// against the resource's display name. For exact-name lookup, use the singular
// "by name" data source instead.
const classicFilterBlockDescription = "Optional client-side filter for classic-API list operations. Applied after the full list is fetched (classic endpoints do not accept query parameters). Use the singular `by name` data source for exact-name lookup."

const classicNameSubstringDescription = "Case-insensitive substring matched against the resource's display name. Empty or omitted returns every item."

// ClassicFilterModel is the parsed Terraform value for classic filter blocks.
// All classic list resources and plural data sources share this one shape —
// classic has no RSQL or query parameters, so a single substring is the only
// dimension we filter on.
type ClassicFilterModel struct {
	NameSubstring types.String `tfsdk:"name_substring"`
}

// ClassicFilterAttribute returns the schema block for a plural data source's
// optional client-side filter. By convention callers MUST mount the returned
// value under attribute key `"filter"` so every classic data source presents
// the same `filter { name_substring = "…" }` shape to users.
func ClassicFilterAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: classicFilterBlockDescription,
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"name_substring": schema.StringAttribute{
				MarkdownDescription: classicNameSubstringDescription,
				Optional:            true,
			},
		},
	}
}

// ClassicListFilterAttribute is the list-resource counterpart to
// ClassicFilterAttribute. Same shape, list-schema types, same mount-at-`filter`
// convention.
func ClassicListFilterAttribute() listschema.SingleNestedAttribute {
	return listschema.SingleNestedAttribute{
		MarkdownDescription: classicFilterBlockDescription,
		Optional:            true,
		Attributes: map[string]listschema.Attribute{
			"name_substring": listschema.StringAttribute{
				MarkdownDescription: classicNameSubstringDescription,
				Optional:            true,
			},
		},
	}
}

// ApplyClassicFilter applies a parsed ClassicFilterModel to a slice and returns
// the filtered result. Null, unknown, or empty NameSubstring returns the input
// slice **as-is** (same underlying array, `result == items` reference-equal).
// Otherwise items survive the filter when the lowered name extracted by the
// supplied accessor contains the lowered substring, and the return value is a
// **fresh non-nil** slice (possibly length zero) backed by its own array.
//
// Generic over T so every classic list resource passes its own SDK item type
// and a one-line name accessor — no per-resource filter logic duplication.
//
// The input slice is never mutated regardless of which branch is taken.
func ApplyClassicFilter[T any](items []T, f ClassicFilterModel, name func(T) string) []T {
	needle, ok := configuredFilterValue(f.NameSubstring)
	if !ok {
		return items
	}
	needleLower := strings.ToLower(needle)
	out := make([]T, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(name(item)), needleLower) {
			out = append(out, item)
		}
	}
	return out
}
