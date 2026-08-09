// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package restricted_software

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

func TestRestrictedSoftwareImpactScope(t *testing.T) {
	got := restrictedSoftwareImpactScope(context.Background(), &RestrictedSoftwareResourceModel{
		Scope: &RestrictedSoftwareScopeModel{
			Targets: &RestrictedSoftwareScopeTargetsModel{
				AllComputers:     types.BoolValue(true),
				ComputerGroupIDs: impactSet(t, "12"),
			},
			Exclusions: &RestrictedSoftwareScopeExclusionsModel{
				ComputerGroupIDs:                 impactSet(t, "13"),
				DirectoryServiceOrLocalUserNames: impactSet(t, "someone"),
			},
		},
	})

	if !got.All {
		t.Fatal("all_computers must set the tenant-wide flag")
	}
	if len(got.ProGroups) != 1 || got.ProGroups[0].DeviceType != impact.DeviceTypeComputer {
		t.Fatalf("target groups wrong: %+v", got.ProGroups)
	}
	if len(got.ExcludedProGroups) != 1 {
		t.Fatalf("excluded groups must be carried as data: %+v", got.ExcludedProGroups)
	}

	// This resource has no limitations block at all, so nothing from that section
	// may be reported.
	for _, u := range got.Unresolvable {
		if len(u.Path) > 11 && u.Path[:11] == "limitations" {
			t.Fatalf("restricted software has no limitations block; %s must not be reported", u.Path)
		}
	}
	var sawUserNames bool
	for _, u := range got.Unresolvable {
		if u.Path == "exclusions.directory_service_or_local_user_names" {
			sawUserNames = true
			if u.Effect != impact.Narrows {
				t.Fatalf("an excluded user name narrows, got %v", u.Effect)
			}
		}
	}
	if !sawUserNames {
		t.Fatalf("the excluded user names must be reported: %+v", got.Unresolvable)
	}
}

func TestRestrictedSoftwareImpactScopeNilScope(t *testing.T) {
	if got := restrictedSoftwareImpactScope(context.Background(), &RestrictedSoftwareResourceModel{}); !got.Empty() {
		t.Fatalf("an absent scope block yields nothing, got %+v", got)
	}
}
