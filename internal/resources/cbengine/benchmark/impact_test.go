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

func TestBenchmarkImpactScopePluralTargets(t *testing.T) {
	got := benchmarkImpactScope(context.Background(), &BenchmarkResourceModel{
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

func TestBenchmarkImpactScopeNilModel(t *testing.T) {
	got := benchmarkImpactScope(context.Background(), nil)
	if got.DeviceType != impact.DeviceTypeAny {
		t.Fatalf("even an empty benchmark scope spans both estates, got %v", got.DeviceType)
	}
	if !got.Empty() {
		t.Fatalf("a nil model must yield an empty scope, got %+v", got)
	}
}

func TestBenchmarkImpactScopeUnknownPluralIsPending(t *testing.T) {
	// An unknown set references groups this same plan creates, so the audience
	// cannot be totalled and the path is recorded as pending instead.
	got := benchmarkImpactScope(context.Background(), &BenchmarkResourceModel{
		TargetDeviceGroups: types.SetUnknown(types.StringType),
	})
	if len(got.PlatformGroupIDs) != 0 {
		t.Fatalf("an unknown target must not be counted, got %v", got.PlatformGroupIDs)
	}
	if len(got.PendingPaths) != 1 || got.PendingPaths[0] != "target_device_groups" {
		t.Fatalf("the unknown target must be pending, got %v", got.PendingPaths)
	}
}
