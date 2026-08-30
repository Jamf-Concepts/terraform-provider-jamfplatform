// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_app

import (
	"context"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/securitycloud"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestAssignAppResourceModelCollapsesEmptyCollections pins the reconciliation that
// stops an application with no host names from diffing against its own refresh
// forever. The endpoint returns `[]` rather than omitting an empty collection, and
// the schema refuses an explicit empty collection, so `[]` can only have come from an
// absent attribute and has to read back as null.
func TestAssignAppResourceModelCollapsesEmptyCollections(t *testing.T) {
	state := ZtnaAppResourceModel{}
	app := &securitycloud.App{
		ID:           "00000000-0000-0000-0000-000000000001",
		CategoryName: "Uncategorized",
		Hostnames:    []string{},
		BareIps:      []string{},
		Assignments: &securitycloud.Assignments{
			Inclusions: securitycloud.AssignmentsInclusions{AllUsers: true, Groups: &[]string{}},
		},
		Routing: &securitycloud.Routing{Type: securitycloud.RoutingTypeDirect},
	}

	diags := assignAppResourceModel(context.Background(), &state, app)
	if diags.HasError() {
		t.Fatalf("assigning state: %s", diags)
	}

	for name, set := range map[string]types.Set{
		"hostnames":              state.Hostnames,
		"direct_ips_and_subnets": state.DirectIPsAndSubnets,
		"device_group_ids":       state.DeviceGroupIDs,
	} {
		if !set.IsNull() {
			t.Errorf("%s must collapse an empty response to null, got %s", name, set)
		}
	}
	if !state.RoutingOverrides.IsNull() {
		t.Errorf("routing_overrides must collapse an empty response to null, got %s", state.RoutingOverrides)
	}
}

// TestAssignAppResourceModelDerivesAppType pins the derived attribute for both forms.
func TestAssignAppResourceModelDerivesAppType(t *testing.T) {
	predefined := "2aaa401c-232e-4db1-8384-6a94d9fc264e"
	name := "Internal CRM"

	cases := []struct {
		label   string
		app     *securitycloud.App
		wantTyp string
		wantNil bool
	}{
		{"predefined", &securitycloud.App{PredefinedAppID: &predefined}, appTypePredefined, true},
		{"custom", &securitycloud.App{Name: &name}, appTypeCustom, false},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			tc.app.Routing = &securitycloud.Routing{Type: securitycloud.RoutingTypeDirect}
			state := ZtnaAppResourceModel{}
			if diags := assignAppResourceModel(context.Background(), &state, tc.app); diags.HasError() {
				t.Fatalf("assigning state: %s", diags)
			}
			if state.AppType.ValueString() != tc.wantTyp {
				t.Errorf("app_type = %q, want %q", state.AppType.ValueString(), tc.wantTyp)
			}
			if state.Name.IsNull() != tc.wantNil {
				t.Errorf("name null = %v, want %v", state.Name.IsNull(), tc.wantNil)
			}
		})
	}
}

// TestAssignAppResourceModelGatesSecurityOnTheTarget is the reconciliation that keeps
// state honest about what the configuration asked for. The server always returns all
// three cards with its own defaults, so copying the response across would put values
// in state the operator never wrote and the next plan would show them as removals.
func TestAssignAppResourceModelGatesSecurityOnTheTarget(t *testing.T) {
	response := &securitycloud.AppSecurity{
		DeviceManagementBasedAccess: &securitycloud.DeviceManagementBasedAccess{Enabled: true, NotificationsEnabled: true},
		RiskControls: &securitycloud.RiskControls{
			Enabled:              true,
			LevelThreshold:       securitycloud.RiskControlsLevelThresholdHigh,
			NotificationsEnabled: true,
		},
		DohIntegration: &securitycloud.DohIntegration{Blocking: true, NotificationsEnabled: true},
	}

	t.Run("no security block declared", func(t *testing.T) {
		state := ZtnaAppResourceModel{}
		app := &securitycloud.App{Routing: &securitycloud.Routing{Type: securitycloud.RoutingTypeDirect}, Security: response}
		if diags := assignAppResourceModel(context.Background(), &state, app); diags.HasError() {
			t.Fatalf("assigning state: %s", diags)
		}
		if state.Security != nil {
			t.Errorf("security must stay absent when the configuration never declared it, got %+v", state.Security)
		}
	})

	t.Run("one card declared", func(t *testing.T) {
		state := ZtnaAppResourceModel{
			Security: &SecurityModel{JamfTrust: &SecurityControlModel{}},
		}
		app := &securitycloud.App{Routing: &securitycloud.Routing{Type: securitycloud.RoutingTypeDirect}, Security: response}
		if diags := assignAppResourceModel(context.Background(), &state, app); diags.HasError() {
			t.Fatalf("assigning state: %s", diags)
		}
		if state.Security == nil || state.Security.JamfTrust == nil {
			t.Fatal("the declared card must be populated")
		}
		if !state.Security.JamfTrust.Enabled.ValueBool() {
			t.Error("jamf_trust.enabled must come from dohIntegration.blocking")
		}
		if state.Security.ManagedDevice != nil || state.Security.DeviceRisk != nil {
			t.Error("an undeclared card must stay absent even though the server reports one")
		}
	})
}

// TestRoutingFromWireMapsLabels pins the read side of both enum translations,
// including that an absent resolution mode stays absent rather than becoming a
// label.
func TestRoutingFromWireMapsLabels(t *testing.T) {
	gateway := "a7d2"
	ipv4 := securitycloud.RoutingDnsIpResolutionTypeIPv4

	viaZTNA := routingFromWire(&securitycloud.Routing{
		Type:                securitycloud.RoutingTypeCustom,
		GatewayID:           &gateway,
		DnsIpResolutionType: &ipv4,
	})
	if viaZTNA.Mode.ValueString() != routingModeLabels[securitycloud.RoutingTypeCustom] {
		t.Errorf("mode = %q", viaZTNA.Mode.ValueString())
	}
	if viaZTNA.RoutingMode.ValueString() != "Legacy" {
		t.Errorf("routing_mode = %q, want Legacy", viaZTNA.RoutingMode.ValueString())
	}

	direct := routingFromWire(&securitycloud.Routing{Type: securitycloud.RoutingTypeDirect})
	if !direct.GatewayID.IsNull() || !direct.RoutingMode.IsNull() {
		t.Errorf("direct routing must read back with both members null, got gateway=%s resolution=%s",
			direct.GatewayID, direct.RoutingMode)
	}
	if routingFromWire(nil) != nil {
		t.Error("routingFromWire(nil) must stay nil")
	}
}

// TestAssignAppDataSourceModelKeepsEmptyCollections pins the difference between the
// two read paths. A data source reports what the server holds with no configuration
// to reconcile against, so an empty collection stays empty rather than collapsing,
// and all three security cards are always populated.
func TestAssignAppDataSourceModelKeepsEmptyCollections(t *testing.T) {
	var ds ZtnaAppDataSourceModel
	app := &securitycloud.App{
		ID:           "00000000-0000-0000-0000-000000000001",
		CategoryName: "Uncategorized",
		Hostnames:    []string{},
		BareIps:      []string{},
		Assignments: &securitycloud.Assignments{
			Inclusions: securitycloud.AssignmentsInclusions{AllUsers: true, Groups: &[]string{}},
		},
		Routing: &securitycloud.Routing{Type: securitycloud.RoutingTypeDirect},
		Security: &securitycloud.AppSecurity{
			DeviceManagementBasedAccess: &securitycloud.DeviceManagementBasedAccess{},
			RiskControls:                &securitycloud.RiskControls{LevelThreshold: securitycloud.RiskControlsLevelThresholdLow},
			DohIntegration:              &securitycloud.DohIntegration{},
		},
	}

	if diags := assignAppDataSourceModel(context.Background(), &ds, app); diags.HasError() {
		t.Fatalf("assigning data source state: %s", diags)
	}

	for name, list := range map[string]types.List{
		"hostnames":              ds.Hostnames,
		"direct_ips_and_subnets": ds.DirectIPsAndSubnets,
		"device_group_ids":       ds.DeviceGroupIDs,
		"routing_overrides":      ds.RoutingOverrides,
	} {
		if list.IsNull() {
			t.Errorf("%s must stay an empty list on the data source, got null", name)
		}
		if len(list.Elements()) != 0 {
			t.Errorf("%s = %s, want empty", name, list)
		}
	}
	if ds.Security.IsNull() {
		t.Fatal("the data source must always report the security block")
	}
}

// TestAssignAppResourceModelTolerantOfNoAssignments pins that a response missing the
// assignments object does not panic or produce a half-filled model. The spec marks it
// required, but every optional pointer on a generated type is one a server change
// could start omitting.
func TestAssignAppResourceModelTolerantOfNoAssignments(t *testing.T) {
	state := ZtnaAppResourceModel{}
	app := &securitycloud.App{
		ID:      "00000000-0000-0000-0000-000000000001",
		Routing: &securitycloud.Routing{Type: securitycloud.RoutingTypeDirect},
	}
	if diags := assignAppResourceModel(context.Background(), &state, app); diags.HasError() {
		t.Fatalf("assigning state: %s", diags)
	}
	if state.AllDeviceGroups.ValueBool() {
		t.Error("all_device_groups must default to false when the response carries no assignments")
	}
	if !state.DeviceGroupIDs.IsNull() {
		t.Error("device_group_ids must be null when the response carries no assignments")
	}
}

// TestBuildAppsResultModel pins that the plural result carries the same values the
// singular data source would, since it is built from the same assignment.
func TestBuildAppsResultModel(t *testing.T) {
	name := "Internal CRM"
	app := securitycloud.App{
		ID:           "00000000-0000-0000-0000-000000000002",
		Name:         &name,
		CategoryName: "Technology",
		Hostnames:    []string{"crm.example.com"},
		Assignments: &securitycloud.Assignments{
			Inclusions: securitycloud.AssignmentsInclusions{AllUsers: false, Groups: &[]string{"group-a"}},
		},
		Routing: &securitycloud.Routing{Type: securitycloud.RoutingTypeDirect},
	}

	result, diags := buildAppsResultModel(context.Background(), app)
	if diags.HasError() {
		t.Fatalf("building result: %s", diags)
	}
	if result.Name.ValueString() != name {
		t.Errorf("name = %q", result.Name.ValueString())
	}
	if result.AppType.ValueString() != appTypeCustom {
		t.Errorf("app_type = %q, want %q", result.AppType.ValueString(), appTypeCustom)
	}
	if len(result.DeviceGroupIDs.Elements()) != 1 {
		t.Errorf("device_group_ids = %s", result.DeviceGroupIDs)
	}
}

// TestDisplayNameFor pins the list-result label, which has to fall back twice: a
// predefined application has no name, and nothing guarantees a definition ID either.
func TestDisplayNameFor(t *testing.T) {
	name := "Internal CRM"
	predefined := "2aaa401c"
	empty := ""

	cases := []struct {
		label string
		app   securitycloud.App
		want  string
	}{
		{"named", securitycloud.App{ID: "id-1", Name: &name}, name},
		{"predefined", securitycloud.App{ID: "id-2", PredefinedAppID: &predefined}, predefined},
		{"empty name falls through", securitycloud.App{ID: "id-3", Name: &empty, PredefinedAppID: &predefined}, predefined},
		{"neither", securitycloud.App{ID: "id-4"}, "id-4"},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			if got := displayNameFor(tc.app); got != tc.want {
				t.Errorf("displayNameFor = %q, want %q", got, tc.want)
			}
		})
	}
}
