// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_policy

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/impact"
)

func impactSet(t *testing.T, vals ...string) types.Set {
	t.Helper()
	out, diags := types.SetValueFrom(context.Background(), types.StringType, vals)
	if diags.HasError() {
		t.Fatalf("building set: %v", diags)
	}
	return out
}

func impactEffects(s impact.Scope) map[string]impact.Effect {
	out := map[string]impact.Effect{}
	for _, u := range s.Unresolvable {
		out[u.Path] = u.Effect
	}
	return out
}

func TestPatchPolicyImpactScopeClassifiesItsLimitedScope(t *testing.T) {
	got := patchPolicyImpactScope(context.Background(), &PatchPolicyResourceModel{
		Scope: &PatchPolicyScopeModel{
			Targets: &PatchPolicyScopeTargetsModel{
				ComputerGroupIDs: impactSet(t, "12"),
				BuildingIDs:      impactSet(t, "321"),
			},
			Limitations: &PatchPolicyScopeLimitationsModel{
				NetworkSegmentIDs: impactSet(t, "7"),
			},
			Exclusions: &PatchPolicyScopeExclusionsModel{
				ComputerGroupIDs: impactSet(t, "13"),
				IbeaconIDs:       impactSet(t, "4"),
			},
		},
	})

	if got.DeviceType != impact.DeviceTypeComputer {
		t.Fatalf("a patch policy is computer-scoped, got %v", got.DeviceType)
	}
	if len(got.ProGroups) != 1 || got.ProGroups[0].DeviceType != impact.DeviceTypeComputer {
		t.Fatalf("target groups wrong: %+v", got.ProGroups)
	}
	if len(got.ExcludedProGroups) != 1 {
		t.Fatalf("excluded groups must be carried as data: %+v", got.ExcludedProGroups)
	}

	eff := impactEffects(got)
	if eff["limitations.network_segment_ids"] != impact.Narrows {
		t.Fatalf("a limitation narrows, got %v", eff["limitations.network_segment_ids"])
	}
	if eff["exclusions.ibeacon_ids"] != impact.Narrows {
		t.Fatalf("an iBeacon exclusion narrows, got %v", eff["exclusions.ibeacon_ids"])
	}
	// Buildings are carried as data, so they resolve to real devices rather than
	// being reported as an unquantified caveat.
	if len(got.BuildingIDs) != 1 || got.BuildingIDs[0] != "321" {
		t.Fatalf("buildings must be carried as data: %v", got.BuildingIDs)
	}
	if _, reported := eff["targets.building_ids"]; reported {
		t.Fatalf("a building carried as data must not also be a caveat: %+v", got.Unresolvable)
	}
	// A patch policy has no user-based categories at all, so none may be reported.
	for p := range eff {
		if p == "targets.user_ids" || p == "targets.user_group_ids" || p == "targets.all_jss_users" {
			t.Fatalf("a patch policy has no user scope; %s must not be reported", p)
		}
	}
}

func TestPatchPolicyImpactScopeNilScope(t *testing.T) {
	if got := patchPolicyImpactScope(context.Background(), &PatchPolicyResourceModel{}); !got.Empty() {
		t.Fatalf("an absent scope block yields nothing, got %+v", got)
	}
}
