// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ebook

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

// refsByEstate indexes resolved group references so a test can assert which
// estate each id was tagged with.
func refsByEstate(s impact.Scope) map[string]impact.DeviceType {
	out := map[string]impact.DeviceType{}
	for _, r := range s.ProGroups {
		out[r.ID] = r.DeviceType
	}
	return out
}

func TestEbookImpactScopeTagsEachEstateSeparately(t *testing.T) {
	// The reason group references carry an estate: an ebook names computer groups
	// and mobile device groups in one block, and numeric ids repeat across the two.
	// Tagging by attribute is the only way id 1 under computer_group_ids and id 1
	// under mobile_device_group_ids resolve to different groups.
	got := ebookImpactScope(context.Background(), &EbookResourceModel{
		Scope: &EbookScopeModel{
			Targets: &EbookScopeTargetsModel{
				ComputerGroupIDs:     impactSet(t, "12"),
				MobileDeviceGroupIDs: impactSet(t, "66"),
				ComputerIDs:          nullSet(),
				MobileDeviceIDs:      nullSet(),
				BuildingIDs:          nullSet(),
				DepartmentIDs:        nullSet(),
				UserIDs:              nullSet(),
				UserGroupIDs:         nullSet(),
				ClassIDs:             nullSet(),
			},
		},
	})
	if got.DeviceType != impact.DeviceTypeAny {
		t.Fatalf("an ebook spans both estates, got %v", got.DeviceType)
	}
	byEstate := refsByEstate(got)
	if byEstate["12"] != impact.DeviceTypeComputer {
		t.Fatalf("computer_group_ids must be tagged as computers, got %v", byEstate["12"])
	}
	if byEstate["66"] != impact.DeviceTypeMobile {
		t.Fatalf("mobile_device_group_ids must be tagged as mobile devices, got %v", byEstate["66"])
	}
}

func TestEbookImpactScopeClassTargetsBroaden(t *testing.T) {
	got := ebookImpactScope(context.Background(), &EbookResourceModel{
		Scope: &EbookScopeModel{
			Targets: &EbookScopeTargetsModel{
				ComputerGroupIDs:     nullSet(),
				MobileDeviceGroupIDs: nullSet(),
				ComputerIDs:          nullSet(),
				MobileDeviceIDs:      nullSet(),
				BuildingIDs:          nullSet(),
				DepartmentIDs:        nullSet(),
				UserIDs:              nullSet(),
				UserGroupIDs:         nullSet(),
				ClassIDs:             impactSet(t, "3"),
			},
		},
	})
	var found bool
	for _, u := range got.Unresolvable {
		if u.Path == "targets.class_ids" {
			found = true
			if u.Effect != impact.Broadens {
				t.Fatalf("classes can only add devices, got %v", u.Effect)
			}
		}
	}
	if !found {
		t.Fatalf("class targets must be recorded: %+v", got.Unresolvable)
	}
}

func TestEbookImpactScopeExclusionsCoverBothEstates(t *testing.T) {
	got := ebookImpactScope(context.Background(), &EbookResourceModel{
		Scope: &EbookScopeModel{
			Exclusions: &EbookScopeExclusionsModel{
				ComputerGroupIDs:                 impactSet(t, "13"),
				MobileDeviceGroupIDs:             impactSet(t, "67"),
				ComputerIDs:                      nullSet(),
				MobileDeviceIDs:                  nullSet(),
				BuildingIDs:                      nullSet(),
				DepartmentIDs:                    nullSet(),
				UserIDs:                          nullSet(),
				UserGroupIDs:                     nullSet(),
				NetworkSegmentIDs:                nullSet(),
				DirectoryServiceOrLocalUserNames: nullSet(),
				DirectoryServiceUserGroupNames:   nullSet(),
			},
		},
	})
	if len(got.ExcludedProGroups) != 2 {
		t.Fatalf("both estates' exclusions must be carried as data: %+v", got.ExcludedProGroups)
	}
	seen := map[impact.DeviceType]bool{}
	for _, r := range got.ExcludedProGroups {
		seen[r.DeviceType] = true
	}
	if !seen[impact.DeviceTypeComputer] || !seen[impact.DeviceTypeMobile] {
		t.Fatalf("exclusions must be tagged per estate: %+v", got.ExcludedProGroups)
	}
}

func TestEbookImpactScopeNilScope(t *testing.T) {
	if got := ebookImpactScope(context.Background(), &EbookResourceModel{}); !got.Empty() {
		t.Fatalf("an absent scope block yields nothing, got %+v", got)
	}
	if got := ebookImpactScope(context.Background(), nil); !got.Empty() {
		t.Fatalf("a nil model yields nothing, got %+v", got)
	}
}
