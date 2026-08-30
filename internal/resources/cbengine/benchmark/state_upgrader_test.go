// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package benchmark

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// State upgraders are unit-tested rather than acceptance-tested: the acceptance
// harness always writes state with the current schema version, so the v0→v1 path
// is unreachable from it.

func TestUpgradeTargetDeviceGroups_PromotesTheSingular(t *testing.T) {
	// The whole point of the upgrader. Without it the singular value is dropped,
	// target_device_groups reads null, and the user's first plan after editing
	// their config to the plural attribute sees null -> ["dg-1"] on an attribute
	// carrying RequiresReplace — destroying and recreating a live benchmark.
	got := upgradeTargetDeviceGroups(benchmarkResourceModelV0{
		TargetDeviceGroup:  types.StringValue("dg-1"),
		TargetDeviceGroups: types.SetNull(types.StringType),
	})
	if got.IsNull() {
		t.Fatal("the singular value must survive the upgrade, not be dropped")
	}
	if elems := setStrings(got); len(elems) != 1 || elems[0] != "dg-1" {
		t.Errorf("expected [dg-1], got %v", elems)
	}
}

func TestUpgradeTargetDeviceGroups_KeepsThePlural(t *testing.T) {
	got := upgradeTargetDeviceGroups(benchmarkResourceModelV0{
		TargetDeviceGroup:  types.StringNull(),
		TargetDeviceGroups: types.SetValueMust(types.StringType, nil),
	})
	if got.IsNull() {
		t.Fatal("an explicitly set plural must be preserved as-is")
	}
}

func TestUpgradeTargetDeviceGroups_PluralWinsOverSingular(t *testing.T) {
	// v0 validation made the two mutually exclusive, so this state should not
	// exist — but if it does, the preferred attribute is the one to trust rather
	// than silently merging a value the user did not intend to keep.
	got := upgradeTargetDeviceGroups(benchmarkResourceModelV0{
		TargetDeviceGroup:  types.StringValue("singular"),
		TargetDeviceGroups: types.SetValueMust(types.StringType, nil),
	})
	if elems := setStrings(got); len(elems) != 0 {
		t.Errorf("the plural attribute should win, got %v", elems)
	}
}

func TestUpgradeTargetDeviceGroups_NeitherSetStaysNull(t *testing.T) {
	got := upgradeTargetDeviceGroups(benchmarkResourceModelV0{
		TargetDeviceGroup:  types.StringNull(),
		TargetDeviceGroups: types.SetNull(types.StringType),
	})
	if !got.IsNull() {
		t.Errorf("nothing to migrate should stay null, got %v", got)
	}
}

func TestUpgradeTargetDeviceGroups_EmptySingularIsNotPromoted(t *testing.T) {
	got := upgradeTargetDeviceGroups(benchmarkResourceModelV0{
		TargetDeviceGroup:  types.StringValue(""),
		TargetDeviceGroups: types.SetNull(types.StringType),
	})
	if !got.IsNull() {
		t.Errorf("an empty singular is not a device group, got %v", got)
	}
}

// TestUpgradeState_DeclaresTheV0Path guards the wiring: a missing entry for
// version 0 makes every v0 state unreadable at v1 with a framework error, which
// is worse than the forced replacement the upgrader exists to prevent.
func TestUpgradeState_DeclaresTheV0Path(t *testing.T) {
	r := NewBenchmarkResource().(*BenchmarkResource)
	upgraders := r.UpgradeState(context.Background())
	u, ok := upgraders[0]
	if !ok {
		t.Fatalf("no upgrader registered for schema version 0, got versions %v", upgraders)
	}
	if u.PriorSchema == nil {
		t.Fatal("the v0 upgrader needs a PriorSchema to decode state that still carries target_device_group")
	}
	if _, ok := u.PriorSchema.Attributes["target_device_group"]; !ok {
		t.Error("the v0 PriorSchema must still declare target_device_group, or v0 state cannot be decoded")
	}
	if u.PriorSchema.Version != 0 {
		t.Errorf("PriorSchema version should be 0, got %d", u.PriorSchema.Version)
	}
}
