// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package scope

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/ldapgroups"
)

// ValidateDirectoryServiceUserGroupNames is a plan-time preflight for a scope
// `directory_service_user_group_names` set. The classic scope endpoints match
// these names against the tenant's configured directory service and reject
// unknown names at apply time with an opaque 409 ("Problem matching limitation
// user group"). This surfaces a hint at plan time with a clear, per-name message —
// call it from a scope-bearing resource's ModifyPlan, once per set (limitations and
// exclusions each have their own).
//
// It is ADVISORY (never blocks the plan): a no-match is a WARNING, not an error.
// Blocking would break the legitimate bootstrap case — creating the directory and a
// group-scoped resource in the same apply, where the directory does not exist yet at
// plan time. A genuinely wrong name still fails loudly at apply (the write retries
// the 409 for a bounded window, then surfaces it). See
// helpers.IsDirectoryGroupMatchConflict / the resources' retry-on-write.
//
// Behaviour:
//   - nil searcher, or null/unknown set: no-op (nothing to check, or the
//     provider is not yet configured).
//   - unknown / null individual elements (e.g. a name interpolated from another
//     resource's not-yet-known attribute): skipped — they cannot be validated
//     at plan time.
//   - a name resolving to >=1 exact directory group (on any server): OK.
//   - a name with no exact match: a WARNING (bootstrap-friendly; enforced at apply).
//   - a search transport error: a WARNING (the directory API is unreachable / not
//     yet configured; the server still enforces on apply).
func ValidateDirectoryServiceUserGroupNames(ctx context.Context, searcher ldapgroups.Searcher, set types.Set, attrPath path.Path) diag.Diagnostics {
	var diags diag.Diagnostics
	if searcher == nil || set.IsNull() || set.IsUnknown() {
		return diags
	}

	for _, elem := range set.Elements() {
		s, ok := elem.(types.String)
		if !ok || s.IsNull() || s.IsUnknown() {
			continue
		}
		name := s.ValueString()
		if name == "" {
			continue
		}

		matches, err := ldapgroups.ResolveByName(ctx, searcher, name)
		if err != nil {
			diags.AddAttributeWarning(
				attrPath,
				"Could not verify directory-service group",
				fmt.Sprintf("Skipping plan-time validation of directory-service user group %q: %s. The Jamf Pro server will still enforce this on apply.", name, err),
			)
			continue
		}
		if len(matches) == 0 {
			diags.AddAttributeWarning(
				attrPath,
				"Directory-service group not found yet",
				fmt.Sprintf("No directory-service user group named %q exists on any configured LDAP / cloud-identity-provider server *as of plan time*. This is expected when the directory is being created in the same apply (bootstrap); the write retries until the group resolves. If the name is simply wrong it will fail at apply with \"Problem matching limitation user group\" — the match is exact.", name),
			)
		}
	}

	return diags
}
