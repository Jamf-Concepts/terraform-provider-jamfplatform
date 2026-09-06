// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package accountprivileges

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Model is the Terraform model for the privileges single-nested block. Each
// field is a Set of privilege strings for one category. A nil/null Set means
// the category is unmanaged by the configuration: it stays null on read
// (IntersectIntoState) and on write it is carried from the live server grid
// rather than omitted (MergeGrid), because the classic endpoints replace the
// whole grid on any sent <privileges> element.
type Model struct {
	JamfProServerObjects  types.Set `tfsdk:"jamf_pro_server_objects"`
	JamfProServerSettings types.Set `tfsdk:"jamf_pro_server_settings"`
	JamfProServerActions  types.Set `tfsdk:"jamf_pro_server_actions"`
	CasperAdmin           types.Set `tfsdk:"casper_admin"`
	CasperRemote          types.Set `tfsdk:"casper_remote"`
	CasperImaging         types.Set `tfsdk:"casper_imaging"`
	Recon                 types.Set `tfsdk:"recon"`
}

// setPtr returns a pointer to the Set field for the given wire key, so the
// generic helpers below can iterate categories without a switch per call site.
func (m *Model) setPtr(wireKey string) *types.Set {
	switch wireKey {
	case "jss_objects":
		return &m.JamfProServerObjects
	case "jss_settings":
		return &m.JamfProServerSettings
	case "jss_actions":
		return &m.JamfProServerActions
	case "casper_admin":
		return &m.CasperAdmin
	case "casper_remote":
		return &m.CasperRemote
	case "casper_imaging":
		return &m.CasperImaging
	case "recon":
		return &m.Recon
	default:
		return nil
	}
}

// AttrTypes returns the attribute types for the privileges object, keyed by
// Terraform attribute name.
func AttrTypes() map[string]attr.Type {
	out := make(map[string]attr.Type, len(Categories))
	for _, c := range Categories {
		out[c.AttrName] = types.SetType{ElemType: types.StringType}
	}
	return out
}

// declaredStrings returns the string elements of a category Set, or nil if the
// Set is null/unknown.
func declaredStrings(ctx context.Context, s types.Set) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if s.IsNull() || s.IsUnknown() {
		return nil, diags
	}
	var out []string
	diags = s.ElementsAs(ctx, &out, false)
	return out, diags
}

// ToMap converts the declared (non-null) categories of a Model into a map keyed
// by wire key. Null/unknown categories are omitted from the map. The result is
// NOT a wire-ready grid: the classic /accounts endpoints replace the whole
// privilege grid on any sent <privileges> element, so a partial map sent as-is
// empties every category it does not name. Callers building a write go through
// MergeGrid, which fills the undeclared categories from the live server grid.
func (m *Model) ToMap(ctx context.Context) (map[string][]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := make(map[string][]string)
	for _, c := range Categories {
		sp := m.setPtr(c.WireKey)
		if sp.IsNull() || sp.IsUnknown() {
			continue
		}
		vals, d := declaredStrings(ctx, *sp)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		out[c.WireKey] = vals
	}
	return out, diags
}

// MergeGrid builds the wire-ready privilege grid for a write: the live server
// grid with every category the configuration declares replaced by the declared
// value. It emulates, client-side, the per-category retention the classic API
// does not provide. Wire-probed 2026-09-06 on Jamf Pro 11.31.1, through the SDK,
// on both PUT /accounts/groupid/{id} and PUT /accounts/userid/{id}: a sent
// <privileges> element replaces the entire grid, so every category absent from
// the body is emptied on the server (jss_settings kept only the server-injected
// "Read License Information"; jss_actions was cleared) while omitting the
// element altogether leaves the grid untouched (issue #385). Read hides that
// loss, because IntersectIntoState treats a null category as unmanaged and never
// refreshes it, which made the wipe invisible to a plan.
//
// The merge is category-granular, the same shape as the shared scope helper's
// per-category ownership (STYLE_GUIDE §Scope helper): a declared category,
// including a declared empty set, wins over the server value and an empty set
// is kept as a present key so it marshals as an empty element and clears the
// category; an undeclared category is carried from server verbatim, including
// any dependency privileges the server injected, which the server would
// re-inject anyway and which Read's intersect already keeps out of state. A nil
// server grid (Create, before anything exists) yields just the declared
// categories. server is never mutated.
func MergeGrid(ctx context.Context, declared *Model, server map[string][]string) (map[string][]string, diag.Diagnostics) {
	out := make(map[string][]string, len(Categories))
	for k, v := range server {
		out[k] = append([]string(nil), v...)
	}
	if declared == nil {
		return out, nil
	}
	declaredMap, diags := declared.ToMap(ctx)
	if diags.HasError() {
		return nil, diags
	}
	for k, v := range declaredMap {
		if v == nil {
			v = []string{}
		}
		out[k] = v
	}
	return out, diags
}

// IsEmpty reports whether every category is null/unknown (no privileges
// declared at all).
func (m *Model) IsEmpty() bool {
	for _, c := range Categories {
		sp := m.setPtr(c.WireKey)
		if !sp.IsNull() && !sp.IsUnknown() {
			return false
		}
	}
	return true
}

// NewStringSet builds a types.Set of strings from a slice (sorted and
// de-duplicated for stable output). A nil slice yields an empty (non-null) set.
// Deduping is required because the classic /accounts endpoint can echo the same
// privilege string more than once within a category (wire-probed 2026-06-12:
// the Administrator grid returns Create/Read/Update Cloud Distribution Point and
// Read/Update Computer Check-In twice each); types.SetValue rejects duplicate
// elements with a hard "Duplicate Set Element" error, which previously aborted
// import, terraform query hydration, and the account_privileges data source
// (issue #290).
func NewStringSet(vals []string) (types.Set, diag.Diagnostics) {
	sorted := append([]string(nil), vals...)
	sort.Strings(sorted)
	elems := make([]attr.Value, 0, len(sorted))
	for i, v := range sorted {
		if i > 0 && v == sorted[i-1] {
			continue
		}
		elems = append(elems, types.StringValue(v))
	}
	return types.SetValue(types.StringType, elems)
}
