// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_grouped_gateway

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const probeTenant = "928260f5-01f3-4881-bd2e-f28faa0dbab2"

// TestBuildGroupedGatewayCreateInput_PreservesMemberOrder pins the ordering
// contract at the boundary. Membership order is the priority order for the
// first-available strategy, so a builder that sorted or reordered would change
// what the group does.
func TestBuildGroupedGatewayCreateInput_PreservesMemberOrder(t *testing.T) {
	plan := groupedGatewayPlan(t, "First available", "30 minutes", "df90", "9702", "c6f0")

	got, diags := buildGroupedGatewayCreateInput(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}

	want := []string{"df90", "9702", "c6f0"}
	if len(got.GatewayIds) != len(want) {
		t.Fatalf("gatewayIds = %v, want %v", got.GatewayIds, want)
	}
	for i := range want {
		if got.GatewayIds[i] != want[i] {
			t.Fatalf("gatewayIds = %v, want %v — order must survive the boundary", got.GatewayIds, want)
		}
	}
}

// TestBuildGroupedGatewayCreateInput_AlwaysSendsGatewayStability pins the odd
// requirement: the field is required on create for every strategy, even the two
// that ignore it, and the Go zero value is rejected.
func TestBuildGroupedGatewayCreateInput_AlwaysSendsGatewayStability(t *testing.T) {
	for label, wire := range map[string]string{"First available": "ACTIVE_STANDBY", "Random": "RANDOM", "Nearest": "NEAREST"} {
		plan := groupedGatewayPlan(t, label, "5 minutes", "df90", "9702")

		got, diags := buildGroupedGatewayCreateInput(context.Background(), plan)
		if diags.HasError() {
			t.Fatalf("%s: diagnostics: %v", label, diags)
		}
		if got.RecoveryDelayInSec != 300 {
			t.Errorf("%s: recoveryDelayInSec = %d, want 300 — required regardless of strategy", label, got.RecoveryDelayInSec)
		}
		if got.RoutingStrategy != wire {
			t.Errorf("routingStrategy = %q, want the stored value %q for label %q", got.RoutingStrategy, wire, label)
		}
	}
}

// TestBuildGroupedGatewayPatchInput_SendsEveryField pins the full-object update.
// Every field is Required in the schema, so the plan always describes a complete
// desired state and a patch that dropped one would silently stop applying a change
// the user made.
func TestBuildGroupedGatewayPatchInput_SendsEveryField(t *testing.T) {
	plan := groupedGatewayPlan(t, "Nearest", "1 hour", "9702", "df90")

	got, diags := buildGroupedGatewayPatchInput(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}

	if got.Name == nil || *got.Name != "tf-group" {
		t.Errorf("name = %v", got.Name)
	}
	if got.RoutingStrategy == nil || *got.RoutingStrategy != "NEAREST" {
		t.Errorf("routingStrategy = %v, want the stored value for \"Nearest\"", got.RoutingStrategy)
	}
	if got.RecoveryDelayInSec == nil || *got.RecoveryDelayInSec != 3600 {
		t.Errorf("recoveryDelayInSec = %v, want the stored value for \"1 hour\"", got.RecoveryDelayInSec)
	}
	if got.GatewayIds == nil || len(*got.GatewayIds) != 2 {
		t.Fatalf("gatewayIds = %v, want 2 entries", got.GatewayIds)
	}
	if (*got.GatewayIds)[0] != "9702" {
		t.Errorf("gatewayIds = %v, want the reordered list applied verbatim", *got.GatewayIds)
	}
	if got.TenantIds == nil || len(*got.TenantIds) != 1 {
		t.Errorf("tenantIds = %v, want 1 entry", got.TenantIds)
	}
}

// groupedGatewayPlan builds a resource model with the given membership order.
func groupedGatewayPlan(t *testing.T, strategy, stability string, gatewayIDs ...string) GroupedGatewayResourceModel {
	t.Helper()

	elems := make([]attr.Value, 0, len(gatewayIDs))
	for _, id := range gatewayIDs {
		elems = append(elems, types.StringValue(id))
	}
	list, diags := types.ListValue(types.StringType, elems)
	if diags.HasError() {
		t.Fatalf("building gateway_ids list: %v", diags)
	}
	tenants, tenantDiags := types.SetValue(types.StringType, []attr.Value{types.StringValue(probeTenant)})
	if tenantDiags.HasError() {
		t.Fatalf("building tenant_ids set: %v", tenantDiags)
	}

	return GroupedGatewayResourceModel{
		ID:                       types.StringValue("b6ed74d2-165c-4503-8572-07eb8ef44195"),
		Name:                     types.StringValue("tf-group"),
		GatewayIDs:               list,
		RoutingStrategy:          types.StringValue(strategy),
		RequiredGatewayStability: types.StringValue(stability),
		TenantIDs:                tenants,
	}
}
