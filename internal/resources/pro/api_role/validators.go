// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_role

import (
	"context"
	"fmt"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// privilegeLister is the subset of *pro.Client the privilege preflight needs.
// Declaring it as an interface keeps the validator unit-testable without a live
// client.
type privilegeLister interface {
	ListApiRolePrivilegesV1(ctx context.Context) (*pro.ApiRolePrivileges, error)
}

// validatePrivileges is a plan-time preflight for an API role's `privileges`
// set. Jamf Pro rejects unknown privileges at apply time with a 400
// ("privilege(s) are not valid [...]"); this surfaces the same failure at plan
// time with a clear, per-privilege message. The valid privilege set varies by
// Jamf Pro version, so it is fetched live from /api/v1/api-role-privileges
// rather than baked into a static enum.
//
// Behaviour:
//   - nil lister, or null/unknown set: no-op (nothing to check, or the provider
//     is not yet configured).
//   - unknown / null individual elements: skipped — they cannot be validated at
//     plan time.
//   - a privilege present in the live list: OK.
//   - a privilege absent from the live list: an error diagnostic at attrPath.
//   - a fetch error: a WARNING (not an error). The preflight is best-effort and
//     must not block plans when the privileges API is unreachable; the server
//     still enforces on apply.
func validatePrivileges(ctx context.Context, lister privilegeLister, set types.Set, attrPath path.Path) diag.Diagnostics {
	var diags diag.Diagnostics
	if lister == nil || set.IsNull() || set.IsUnknown() {
		return diags
	}

	// Collect the requested privileges, skipping unknown/null elements.
	requested := make([]string, 0, len(set.Elements()))
	for _, elem := range set.Elements() {
		s, ok := elem.(types.String)
		if !ok || s.IsNull() || s.IsUnknown() {
			continue
		}
		if name := s.ValueString(); name != "" {
			requested = append(requested, name)
		}
	}
	if len(requested) == 0 {
		return diags
	}

	live, err := lister.ListApiRolePrivilegesV1(ctx)
	if err != nil {
		diags.AddAttributeWarning(
			attrPath,
			"Could not verify API role privileges",
			fmt.Sprintf("Skipping plan-time privilege validation: %s. The Jamf Pro server will still enforce valid privileges on apply.", err),
		)
		return diags
	}

	valid := make(map[string]struct{}, len(live.Privileges))
	for _, p := range live.Privileges {
		valid[p] = struct{}{}
	}

	for _, name := range requested {
		if _, ok := valid[name]; !ok {
			diags.AddAttributeError(
				attrPath,
				"Invalid API role privilege",
				fmt.Sprintf("Privilege %q is not a valid Jamf Pro privilege on this tenant. Jamf Pro would reject this on apply with \"privilege(s) are not valid\". Check the spelling (privileges are matched exactly and are case-sensitive) — the full list is available from the jamfplatform_pro_api_role_privileges data source.", name),
			)
		}
	}

	return diags
}
