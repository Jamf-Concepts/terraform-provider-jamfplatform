// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package accountprivileges

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Validate checks every declared privilege string against the discovered
// tenant catalog and returns an error diagnostic (with a fuzzy "did you mean"
// suggestion) for any privilege Jamf Pro would silently drop. catalog must be
// non-nil; call DiscoveryFailureWarning instead when discovery failed. attrPath
// is the path of the privileges block; per-privilege errors are attached to the
// owning category attribute.
func Validate(ctx context.Context, catalog *Catalog, m *Model, attrPath path.Path) diag.Diagnostics {
	var diags diag.Diagnostics
	if catalog == nil || m == nil {
		return diags
	}
	for _, c := range Categories {
		sp := m.setPtr(c.WireKey)
		if sp.IsNull() || sp.IsUnknown() {
			continue
		}
		var vals []string
		for _, elem := range sp.Elements() {
			s, ok := elem.(types.String)
			if !ok || s.IsNull() || s.IsUnknown() {
				continue
			}
			vals = append(vals, s.ValueString())
		}
		catPath := attrPath.AtName(c.AttrName)
		for _, v := range vals {
			if catalog.Contains(v) {
				continue
			}
			detail := fmt.Sprintf(
				"The privilege %q is not grantable on this Jamf Pro tenant, so the server would silently ignore it (leaving a perpetual diff). It is not present in the tenant's Administrator privilege set.",
				v,
			)
			if sug := closestMatch(v, catalog.All()); sug != "" {
				detail += fmt.Sprintf(" Did you mean %q?", sug)
			}
			diags.AddAttributeError(catPath, "Unknown Jamf Pro privilege", detail)
		}
	}
	return diags
}

// DiscoveryFailureWarning returns a loud warning to emit when the privilege
// catalog could not be discovered, so the user knows declared privileges were
// NOT validated at plan time and a bad value would surface as a perpetual diff
// (not an apply error, because writes trust the planned value).
func DiscoveryFailureWarning(attrPath path.Path, err error) diag.Diagnostics {
	var diags diag.Diagnostics
	diags.AddAttributeWarning(
		attrPath,
		"Could not validate Jamf Pro privileges",
		fmt.Sprintf(
			"The provider could not discover this tenant's privilege catalog (%s), so the privileges below were NOT validated. Jamf Pro silently ignores unrecognised privileges; a typo will appear as a perpetual diff rather than an error. Verify each privilege string against the Jamf Pro admin UI.",
			err,
		),
	)
	return diags
}

// closestMatch returns the candidate with the smallest case-insensitive
// Levenshtein distance to s, provided it is reasonably close (within a third of
// the longer length). Returns "" when nothing is close enough.
func closestMatch(s string, candidates []string) string {
	best := ""
	bestDist := -1
	for _, c := range candidates {
		d := levenshtein(s, c)
		if bestDist == -1 || d < bestDist {
			bestDist = d
			best = c
		}
	}
	if best == "" {
		return ""
	}
	limit := max(len(best), len(s))
	if bestDist*3 > limit {
		return ""
	}
	return best
}

// levenshtein computes the case-insensitive edit distance between a and b.
func levenshtein(a, b string) int {
	a = toLowerASCII(a)
	b = toLowerASCII(b)
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min3(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func min3(a, b, c int) int {
	m := min(c, min(b, a))
	return m
}

func toLowerASCII(s string) string {
	bs := []byte(s)
	for i, c := range bs {
		if c >= 'A' && c <= 'Z' {
			bs[i] = c + ('a' - 'A')
		}
	}
	return string(bs)
}
