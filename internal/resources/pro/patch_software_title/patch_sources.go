// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_software_title

import (
	"context"
	"fmt"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// resolveSourceID resolves a patch source's numeric id from the name the v3
// configuration reports in patchSourceName.
//
// The configurations endpoint names a title's patch source but never numbers
// it, while source_id is what defines a title on the classic create call and
// therefore what the schema exposes. The number is only ever needed where
// there is no prior state to carry it — an import, a data source read, or list
// hydration — so this walks the two classic source catalogues (internal, then
// external) rather than adding a lookup to the steady-state read path. Both
// catalogues are small: a tenant has one internal source and a handful of
// external ones at most.
//
// A name matching in both catalogues is an error rather than a silent
// first-match, since the two id spaces are separate and guessing would put a
// wrong number in state for a RequiresReplace attribute.
func resolveSourceID(ctx context.Context, c *proclassic.Client, name string) (types.Int64, error) {
	if c == nil || name == "" {
		return types.Int64Null(), fmt.Errorf("cannot resolve a patch source without a name")
	}

	var matches []int

	internal, err := c.ListPatchInternalSources(ctx)
	if err != nil {
		return types.Int64Null(), fmt.Errorf("listing internal patch sources while resolving %q: %w", name, err)
	}
	if internal != nil {
		matches = append(matches, matchSourceIDs(internal.PatchInternalSources, name)...)
	}

	external, err := c.ListPatchExternalSources(ctx)
	if err != nil {
		return types.Int64Null(), fmt.Errorf("listing external patch sources while resolving %q: %w", name, err)
	}
	if external != nil {
		matches = append(matches, matchSourceIDs(external.PatchExternalSources, name)...)
	}

	switch len(matches) {
	case 0:
		return types.Int64Null(), fmt.Errorf("no patch source named %q in this tenant's internal or external patch sources", name)
	case 1:
		return types.Int64Value(int64(matches[0])), nil
	default:
		return types.Int64Null(), fmt.Errorf("patch source name %q is not unique across this tenant's internal and external patch sources (ids %v); the provider cannot determine which source_id the title uses", name, matches)
	}
}

// matchSourceIDs returns the ids of every catalogue entry whose name matches
// exactly.
func matchSourceIDs(sources []proclassic.IDName, name string) []int {
	var out []int
	for i := range sources {
		if sources[i].Name == nil || *sources[i].Name != name {
			continue
		}
		if sources[i].ID == nil {
			continue
		}
		out = append(out, *sources[i].ID)
	}
	return out
}
