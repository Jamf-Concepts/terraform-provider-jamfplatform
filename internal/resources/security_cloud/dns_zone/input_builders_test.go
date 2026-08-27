// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package dns_zone

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildZoneWriteInput_FullZone(t *testing.T) {
	plan := planModel(t, "Test Zone", []string{"testdomain.com", "*.testdomain.com"}, []NameServerModel{
		{IP: types.StringValue("203.0.113.53"), GatewayID: types.StringValue("a7d2")},
		{IP: types.StringValue("198.51.100.53"), GatewayID: types.StringValue("1cbb")},
	})

	got, diags := buildZoneWriteInput(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}

	if got.Name != "Test Zone" {
		t.Errorf("Name = %q, want %q", got.Name, "Test Zone")
	}
	if len(got.Domains) != 2 {
		t.Errorf("Domains = %v, want 2 entries", got.Domains)
	}
	if len(got.NameServers) != 2 {
		t.Fatalf("NameServers = %+v, want 2 entries", got.NameServers)
	}
	for _, ns := range got.NameServers {
		if ns.IP == "" || ns.GatewayID == "" {
			t.Errorf("name server %+v lost a field in translation", ns)
		}
	}
}

// TestBuildZonePatchInput_SendsEveryField pins the full-object update. Omitting a
// field would preserve it server-side, but all three are Required in the schema
// so the plan always carries a complete desired state — a patch that dropped one
// would silently stop applying a change the user made.
func TestBuildZonePatchInput_SendsEveryField(t *testing.T) {
	plan := planModel(t, "Renamed Zone", []string{"testdomain.com"}, []NameServerModel{
		{IP: types.StringValue("203.0.113.53"), GatewayID: types.StringValue("a7d2")},
	})

	got, diags := buildZonePatchInput(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}

	if got.Name == nil || *got.Name != "Renamed Zone" {
		t.Errorf("Name = %v, want a pointer to %q", got.Name, "Renamed Zone")
	}
	if got.Domains == nil {
		t.Fatal("Domains must be sent on every update, got nil")
	}
	if len(*got.Domains) != 1 {
		t.Errorf("Domains = %v, want 1 entry", *got.Domains)
	}
	if got.NameServers == nil {
		t.Fatal("NameServers must be sent on every update, got nil")
	}
	if len(*got.NameServers) != 1 {
		t.Errorf("NameServers = %+v, want 1 entry", *got.NameServers)
	}
}

// planModel assembles a resource model the way the framework would.
func planModel(t *testing.T, name string, domains []string, nameServers []NameServerModel) DNSZoneResourceModel {
	t.Helper()

	domainValues := make([]attr.Value, 0, len(domains))
	for _, d := range domains {
		domainValues = append(domainValues, types.StringValue(d))
	}
	domainSet, diags := types.SetValue(types.StringType, domainValues)
	if diags.HasError() {
		t.Fatalf("building domains set: %v", diags)
	}

	nsValues := make([]attr.Value, 0, len(nameServers))
	for _, ns := range nameServers {
		obj, objDiags := types.ObjectValue(nameServerAttributeTypes, map[string]attr.Value{
			"ip_address": ns.IP,
			"gateway_id": ns.GatewayID,
		})
		if objDiags.HasError() {
			t.Fatalf("building name server object: %v", objDiags)
		}
		nsValues = append(nsValues, obj)
	}
	nsSet, nsDiags := types.SetValue(nameServerObjectType, nsValues)
	if nsDiags.HasError() {
		t.Fatalf("building name servers set: %v", nsDiags)
	}

	return DNSZoneResourceModel{
		ID:          types.StringValue("f5734162-26d4-4d0f-a3ae-62f952b3688f"),
		Name:        types.StringValue(name),
		Domains:     domainSet,
		NameServers: nsSet,
	}
}
