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
// the category is unmanaged by the configuration (omit-on-write, stays null on
// read).
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
// by wire key. Null/unknown categories are omitted entirely so the caller can
// honour omit-on-write semantics (the classic PUT merges, retaining categories
// that are not sent).
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

// newStringSet builds a types.Set of strings from a slice (sorted for stable
// output). A nil slice yields an empty (non-null) set.
func newStringSet(vals []string) (types.Set, diag.Diagnostics) {
	sorted := append([]string(nil), vals...)
	sort.Strings(sorted)
	elems := make([]attr.Value, len(sorted))
	for i, v := range sorted {
		elems[i] = types.StringValue(v)
	}
	return types.SetValue(types.StringType, elems)
}
