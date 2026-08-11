// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package benchmark

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/impact"
)

func benchImpactSet(t *testing.T, vals ...string) types.Set {
	t.Helper()
	out, diags := types.SetValueFrom(context.Background(), types.StringType, vals)
	if diags.HasError() {
		t.Fatalf("building set: %v", diags)
	}
	return out
}

func benchImpactNullSet() types.Set { return types.SetNull(types.StringType) }

func TestBenchmarkImpactScopePluralTargets(t *testing.T) {
	got := benchmarkImpactScope(context.Background(), &BenchmarkResourceModel{
		TargetDeviceGroup:  types.StringNull(),
		TargetDeviceGroups: benchImpactSet(t, "uuid-1", "uuid-2"),
	})
	if got.DeviceType != impact.DeviceTypeAny {
		t.Fatalf("a benchmark can reach either estate, got %v", got.DeviceType)
	}
	if len(got.PlatformGroupIDs) != 2 {
		t.Fatalf("plural targets must all be counted, got %v", got.PlatformGroupIDs)
	}
	if len(got.PendingPaths) != 0 {
		t.Fatalf("nothing here is unknown, got pending %v", got.PendingPaths)
	}
}

func TestBenchmarkImpactScopeDeprecatedSingularIsCounted(t *testing.T) {
	// A configuration still written against the deprecated singular attribute has
	// an audience worth reporting, so its value folds into the same category.
	got := benchmarkImpactScope(context.Background(), &BenchmarkResourceModel{
		TargetDeviceGroup:  types.StringValue("uuid-3"),
		TargetDeviceGroups: benchImpactNullSet(),
	})
	if len(got.PlatformGroupIDs) != 1 || got.PlatformGroupIDs[0] != "uuid-3" {
		t.Fatalf("the singular target must be counted, got %v", got.PlatformGroupIDs)
	}
	if len(got.PendingPaths) != 0 {
		t.Fatalf("a known singular target is not pending, got %v", got.PendingPaths)
	}
}

func TestBenchmarkImpactScopeUnknownSingularIsPending(t *testing.T) {
	// An unknown singular value references a group this same plan creates, so the
	// figure cannot be completed and the path must be recorded as pending.
	got := benchmarkImpactScope(context.Background(), &BenchmarkResourceModel{
		TargetDeviceGroup:  types.StringUnknown(),
		TargetDeviceGroups: benchImpactNullSet(),
	})
	if len(got.PlatformGroupIDs) != 0 {
		t.Fatalf("an unknown target must not be counted, got %v", got.PlatformGroupIDs)
	}
	if len(got.PendingPaths) != 1 || got.PendingPaths[0] != "target_device_group" {
		t.Fatalf("the unknown singular target must be pending, got %v", got.PendingPaths)
	}
}

func TestBenchmarkImpactScopeBothAttributesCombine(t *testing.T) {
	got := benchmarkImpactScope(context.Background(), &BenchmarkResourceModel{
		TargetDeviceGroup:  types.StringValue("uuid-3"),
		TargetDeviceGroups: benchImpactSet(t, "uuid-1"),
	})
	if len(got.PlatformGroupIDs) != 2 {
		t.Fatalf("both attributes name groups, so both must be counted, got %v", got.PlatformGroupIDs)
	}
	seen := map[string]bool{}
	for _, id := range got.PlatformGroupIDs {
		seen[id] = true
	}
	if !seen["uuid-1"] || !seen["uuid-3"] {
		t.Fatalf("both the plural and the singular target must appear, got %v", got.PlatformGroupIDs)
	}
}

func TestBenchmarkImpactScopeNilModel(t *testing.T) {
	got := benchmarkImpactScope(context.Background(), nil)
	if got.DeviceType != impact.DeviceTypeAny {
		t.Fatalf("even an empty benchmark scope spans both estates, got %v", got.DeviceType)
	}
	if !got.Empty() {
		t.Fatalf("a nil model must yield an empty scope, got %+v", got)
	}
}
