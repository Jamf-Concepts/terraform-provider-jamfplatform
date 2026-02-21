// Copyright 2026 Jamf Software LLC.

//go:build acceptance

package client_test

import (
	"context"
	"testing"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// Data source acceptance tests verify that all list/read client methods work against a live tenant.

// Blueprint data sources

func TestAcceptance_DataSource_GetBlueprints(t *testing.T) {
	c := testhelpers.NewAcceptanceClient(t)

	blueprints, err := c.GetBlueprintsV1(context.Background(), nil, "")
	if err != nil {
		t.Fatalf("GetBlueprintsV1 failed: %v", err)
	}

	t.Logf("Found %d blueprints", len(blueprints))
}

func TestAcceptance_DataSource_GetBlueprintsWithSort(t *testing.T) {
	c := testhelpers.NewAcceptanceClient(t)

	blueprints, err := c.GetBlueprintsV1(context.Background(), []string{"name:asc"}, "")
	if err != nil {
		t.Fatalf("GetBlueprintsV1 with sort failed: %v", err)
	}

	t.Logf("Found %d blueprints (sorted by name asc)", len(blueprints))
}

func TestAcceptance_DataSource_GetBlueprintsWithSearch(t *testing.T) {
	c := testhelpers.NewAcceptanceClient(t)

	blueprints, err := c.GetBlueprintsV1(context.Background(), nil, "tf-acc")
	if err != nil {
		t.Fatalf("GetBlueprintsV1 with search failed: %v", err)
	}

	t.Logf("Found %d blueprints matching 'tf-acc'", len(blueprints))
}

func TestAcceptance_DataSource_GetBlueprintComponents(t *testing.T) {
	c := testhelpers.NewAcceptanceClient(t)

	components, err := c.GetBlueprintComponentsV1(context.Background())
	if err != nil {
		t.Fatalf("GetBlueprintComponentsV1 failed: %v", err)
	}

	if len(components) == 0 {
		t.Log("No blueprint components found — expected at least some available component types")
		return
	}

	t.Logf("Found %d blueprint components:", len(components))
	for _, comp := range components {
		t.Logf("  Component: %s (%s)", comp.Name, comp.Identifier)
	}
}

func TestAcceptance_DataSource_GetBlueprintComponentByID(t *testing.T) {
	c := testhelpers.NewAcceptanceClient(t)
	ctx := context.Background()

	components, err := c.GetBlueprintComponentsV1(ctx)
	if err != nil {
		t.Fatalf("GetBlueprintComponentsV1 failed: %v", err)
	}
	if len(components) == 0 {
		t.Skip("No blueprint components available to read by ID")
	}

	comp, err := c.GetBlueprintComponentByIDV1(ctx, components[0].Identifier)
	if err != nil {
		t.Fatalf("GetBlueprintComponentByIDV1 failed for %q: %v", components[0].Identifier, err)
	}

	if comp.Identifier != components[0].Identifier {
		t.Errorf("expected identifier %q, got %q", components[0].Identifier, comp.Identifier)
	}

	t.Logf("Read component: %s (%s)", comp.Name, comp.Identifier)
}

// Device Group data sources

func TestAcceptance_DataSource_GetDeviceGroups(t *testing.T) {
	c := testhelpers.NewAcceptanceClient(t)

	groups, err := c.GetDeviceGroupsV1(context.Background(), nil, "")
	if err != nil {
		t.Fatalf("GetDeviceGroupsV1 failed: %v", err)
	}

	t.Logf("Found %d device groups", len(groups))
}

func TestAcceptance_DataSource_GetDeviceGroupsWithSort(t *testing.T) {
	c := testhelpers.NewAcceptanceClient(t)

	groups, err := c.GetDeviceGroupsV1(context.Background(), []string{"name:asc"}, "")
	if err != nil {
		t.Fatalf("GetDeviceGroupsV1 with sort failed: %v", err)
	}

	t.Logf("Found %d device groups (sorted)", len(groups))
}

func TestAcceptance_DataSource_GetDeviceGroupByID(t *testing.T) {
	groupID := testhelpers.RequireSmartGroupFixture(t)
	c := testhelpers.NewAcceptanceClient(t)

	group, err := c.GetDeviceGroupByIDV1(context.Background(), groupID)
	if err != nil {
		t.Fatalf("GetDeviceGroupByIDV1 failed: %v", err)
	}

	if group.ID != groupID {
		t.Errorf("expected group ID %q, got %q", groupID, group.ID)
	}
	if group.GroupType != "SMART" {
		t.Errorf("expected group type 'SMART', got %q", group.GroupType)
	}

	t.Logf("Read device group: %s (%s), members: %d", group.Name, group.ID, group.MemberCount)
}

func TestAcceptance_DataSource_GetDeviceGroupMembers(t *testing.T) {
	groupID := testhelpers.RequireSmartGroupFixture(t)
	c := testhelpers.NewAcceptanceClient(t)

	members, err := c.GetDeviceGroupMembersV1(context.Background(), groupID)
	if err != nil {
		t.Fatalf("GetDeviceGroupMembersV1 failed: %v", err)
	}

	t.Logf("Found %d members in fixture smart group", len(members))
}

// Device data sources

func TestAcceptance_DataSource_GetDevices(t *testing.T) {
	c := testhelpers.NewAcceptanceClient(t)

	devices, err := c.GetDevicesV1(context.Background(), nil, "")
	if err != nil {
		t.Fatalf("GetDevicesV1 failed: %v", err)
	}

	t.Logf("Found %d devices", len(devices))

	if len(devices) > 0 {
		d := devices[0]
		t.Logf("  First device: %s (%s) — %s %s", d.Name, d.ID, d.Model, d.OperatingSystemVersion)
	}
}

func TestAcceptance_DataSource_GetDevicesWithFilter(t *testing.T) {
	c := testhelpers.NewAcceptanceClient(t)

	devices, err := c.GetDevicesV1(context.Background(), nil, "")
	if err != nil {
		t.Fatalf("GetDevicesV1 failed: %v", err)
	}

	t.Logf("Found %d devices (filtered)", len(devices))
}

func TestAcceptance_DataSource_GetDeviceByID(t *testing.T) {
	c := testhelpers.NewAcceptanceClient(t)
	ctx := context.Background()

	devices, err := c.GetDevicesV1(ctx, nil, "")
	if err != nil {
		t.Fatalf("GetDevicesV1 failed: %v", err)
	}
	if len(devices) == 0 {
		t.Skip("No devices available to read by ID")
	}

	device, err := c.GetDeviceByIDV1(ctx, devices[0].ID)
	if err != nil {
		t.Fatalf("GetDeviceByIDV1 failed: %v", err)
	}

	if device.ID != devices[0].ID {
		t.Errorf("expected device ID %q, got %q", devices[0].ID, device.ID)
	}

	t.Logf("Read device: %s (%s), managed: %v, supervised: %v", device.Name, device.ID, device.Managed, device.Supervised)

	if device.Hardware != nil {
		t.Logf("  Hardware: %s %s, serial: %s", device.Hardware.Make, device.Hardware.Model, device.Hardware.SerialNumber)
	}
	if device.OperatingSystem != nil {
		t.Logf("  OS: %s %s (build %s)", device.OperatingSystem.Name, device.OperatingSystem.Version, device.OperatingSystem.Build)
	}
}

func TestAcceptance_DataSource_GetDeviceGroupsForDevice(t *testing.T) {
	c := testhelpers.NewAcceptanceClient(t)
	ctx := context.Background()

	devices, err := c.GetDevicesV1(ctx, nil, "")
	if err != nil {
		t.Fatalf("GetDevicesV1 failed: %v", err)
	}
	if len(devices) == 0 {
		t.Skip("No devices available to check group membership")
	}

	groups, err := c.GetDeviceGroupsForDeviceV1(ctx, devices[0].ID)
	if err != nil {
		t.Fatalf("GetDeviceGroupsForDeviceV1 failed: %v", err)
	}

	t.Logf("Device %s belongs to %d groups", devices[0].ID, len(groups))
	for _, g := range groups {
		t.Logf("  Group: %s (%s)", g.GroupName, g.GroupID)
	}
}

// CBEngine data sources

func TestAcceptance_DataSource_GetBaselines(t *testing.T) {
	c := testhelpers.NewAcceptanceClient(t)

	baselines, err := c.GetCBEngineBaselinesV1(context.Background())
	if err != nil {
		t.Fatalf("GetCBEngineBaselinesV1 failed: %v", err)
	}

	if len(baselines.Baselines) == 0 {
		t.Log("No baselines found — CB Engine may not be enabled on this tenant")
		return
	}

	t.Logf("Found %d baselines:", len(baselines.Baselines))
	for _, b := range baselines.Baselines {
		t.Logf("  %s (%s) — %d rules", b.Title, b.BaselineID, b.RuleCount)
	}
}

func TestAcceptance_DataSource_GetRulesForBaseline(t *testing.T) {
	c := testhelpers.NewAcceptanceClient(t)
	ctx := context.Background()

	baselines, err := c.GetCBEngineBaselinesV1(ctx)
	if err != nil {
		t.Fatalf("GetCBEngineBaselinesV1 failed: %v", err)
	}
	if len(baselines.Baselines) == 0 {
		t.Skip("No baselines available — cannot fetch rules")
	}

	baseline := baselines.Baselines[0]
	rules, err := c.GetCBEngineRulesV1(ctx, baseline.BaselineID)
	if err != nil {
		t.Fatalf("GetCBEngineRulesV1 failed for baseline %q: %v", baseline.BaselineID, err)
	}

	t.Logf("Found %d rules for baseline %q:", len(rules.Rules), baseline.Title)
	t.Logf("  Sources: %d", len(rules.Sources))

	rulesWithODV := 0
	for _, r := range rules.Rules {
		if r.ODV != nil {
			rulesWithODV++
		}
	}
	t.Logf("  Rules with ODV: %d / %d", rulesWithODV, len(rules.Rules))
}

func TestAcceptance_DataSource_GetBenchmarks(t *testing.T) {
	c := testhelpers.NewAcceptanceClient(t)

	benchmarks, err := c.GetCBEngineBenchmarksV2(context.Background())
	if err != nil {
		t.Fatalf("GetCBEngineBenchmarksV2 failed: %v", err)
	}

	t.Logf("Found %d benchmarks", len(benchmarks.Benchmarks))
	for _, b := range benchmarks.Benchmarks {
		t.Logf("  %s (%s) — sync: %s", b.Title, b.ID, b.SyncState)
	}
}
