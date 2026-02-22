// Copyright 2026 Jamf Software LLC.

package blueprint

import (
	"encoding/json"
	"testing"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestParseComponentConfiguration_Found(t *testing.T) {
	apiComponents := map[string]client.BlueprintComponentV1{
		"com.jamf.ddm.disk-management": {
			Identifier:    "com.jamf.ddm.disk-management",
			Configuration: json.RawMessage(`{"externalStorage":"deny","networkStorage":"deny"}`),
		},
	}

	config, ok := parseComponentConfiguration(apiComponents, "com.jamf.ddm.disk-management")
	if !ok {
		t.Fatal("expected to find configuration")
	}
	if config["externalStorage"] != "deny" {
		t.Errorf("expected externalStorage 'deny', got %v", config["externalStorage"])
	}
}

func TestParseComponentConfiguration_NotFound(t *testing.T) {
	apiComponents := map[string]client.BlueprintComponentV1{}

	_, ok := parseComponentConfiguration(apiComponents, "com.jamf.ddm.disk-management")
	if ok {
		t.Error("expected not found for missing component")
	}
}

func TestParseComponentConfiguration_NilConfiguration(t *testing.T) {
	apiComponents := map[string]client.BlueprintComponentV1{
		"com.jamf.ddm.disk-management": {
			Identifier:    "com.jamf.ddm.disk-management",
			Configuration: nil,
		},
	}

	_, ok := parseComponentConfiguration(apiComponents, "com.jamf.ddm.disk-management")
	if ok {
		t.Error("expected not found for nil configuration")
	}
}

func TestParseComponentConfiguration_InvalidJSON(t *testing.T) {
	apiComponents := map[string]client.BlueprintComponentV1{
		"com.jamf.ddm.disk-management": {
			Identifier:    "com.jamf.ddm.disk-management",
			Configuration: json.RawMessage(`{invalid`),
		},
	}

	_, ok := parseComponentConfiguration(apiComponents, "com.jamf.ddm.disk-management")
	if ok {
		t.Error("expected not found for invalid JSON")
	}
}

func TestUpdateLegacyPayloadsFromAPI_WithPayloads(t *testing.T) {
	apiComponents := map[string]client.BlueprintComponentV1{
		"com.jamf.ddm-configuration-profile": {
			Identifier:    "com.jamf.ddm-configuration-profile",
			Configuration: json.RawMessage(`{"payloadDisplayName":"Test","payloadContent":[{"payloadType":"com.apple.wifi.managed"}]}`),
		},
	}
	rawIdentifiers := map[string]struct{}{}

	model := &BlueprintResourceModel{}
	updateLegacyPayloadsFromAPI(model, apiComponents, rawIdentifiers)

	if model.LegacyPayloads.IsNull() {
		t.Fatal("expected non-null legacy payloads")
	}

	var payloads []any
	if err := json.Unmarshal([]byte(model.LegacyPayloads.ValueString()), &payloads); err != nil {
		t.Fatalf("expected valid JSON array, got error: %v", err)
	}
	if len(payloads) != 1 {
		t.Errorf("expected 1 payload, got %d", len(payloads))
	}
}

func TestUpdateLegacyPayloadsFromAPI_NoComponent(t *testing.T) {
	apiComponents := map[string]client.BlueprintComponentV1{}
	rawIdentifiers := map[string]struct{}{}

	model := &BlueprintResourceModel{LegacyPayloads: types.StringValue("should be cleared")}
	updateLegacyPayloadsFromAPI(model, apiComponents, rawIdentifiers)

	if !model.LegacyPayloads.IsNull() {
		t.Error("expected null legacy payloads when component is absent")
	}
}

func TestUpdateLegacyPayloadsFromAPI_HandledAsRaw(t *testing.T) {
	apiComponents := map[string]client.BlueprintComponentV1{
		"com.jamf.ddm-configuration-profile": {
			Identifier:    "com.jamf.ddm-configuration-profile",
			Configuration: json.RawMessage(`{"payloadContent":[{"payloadType":"com.apple.wifi.managed"}]}`),
		},
	}
	rawIdentifiers := map[string]struct{}{
		"com.jamf.ddm-configuration-profile": {},
	}

	model := &BlueprintResourceModel{LegacyPayloads: types.StringValue("existing")}
	updateLegacyPayloadsFromAPI(model, apiComponents, rawIdentifiers)

	if model.LegacyPayloads.ValueString() != "existing" {
		t.Error("expected legacy payloads to remain unchanged when handled as raw")
	}
}

func TestUpdateLegacyPayloadsFromAPI_NoPayloadContent(t *testing.T) {
	apiComponents := map[string]client.BlueprintComponentV1{
		"com.jamf.ddm-configuration-profile": {
			Identifier:    "com.jamf.ddm-configuration-profile",
			Configuration: json.RawMessage(`{"payloadDisplayName":"Test"}`),
		},
	}
	rawIdentifiers := map[string]struct{}{}

	model := &BlueprintResourceModel{}
	updateLegacyPayloadsFromAPI(model, apiComponents, rawIdentifiers)

	if !model.LegacyPayloads.IsNull() {
		t.Error("expected null legacy payloads when payloadContent is absent")
	}
}

func TestUpdateModelFromAPIResponse_BasicFields(t *testing.T) {
	model := &BlueprintResourceModel{}
	blueprint := &client.BlueprintDetailV1{
		ID:          "bp-123",
		Name:        "Test Blueprint",
		Description: "A test blueprint",
		Created:     "2025-01-01T00:00:00Z",
		Updated:     "2025-01-02T00:00:00Z",
		DeploymentState: client.BlueprintDeploymentStateV1{
			State: "DEPLOYED",
		},
		Scope: client.BlueprintUpdateScopeV1{
			DeviceGroups: []string{"group-1", "group-2"},
		},
		Steps: []client.BlueprintStepV1{},
	}

	updateModelFromAPIResponse(model, blueprint)

	if model.ID.ValueString() != "bp-123" {
		t.Errorf("expected ID 'bp-123', got %q", model.ID.ValueString())
	}
	if model.Name.ValueString() != "Test Blueprint" {
		t.Errorf("expected Name 'Test Blueprint', got %q", model.Name.ValueString())
	}
	if model.Description.ValueString() != "A test blueprint" {
		t.Errorf("expected Description 'A test blueprint', got %q", model.Description.ValueString())
	}
	if model.DeploymentState.ValueString() != "DEPLOYED" {
		t.Errorf("expected DeploymentState 'DEPLOYED', got %q", model.DeploymentState.ValueString())
	}
	if !model.Deployed.ValueBool() {
		t.Error("expected Deployed to be true for DEPLOYED state")
	}
	if model.Created.ValueString() != "2025-01-01T00:00:00Z" {
		t.Errorf("expected Created timestamp, got %q", model.Created.ValueString())
	}
}

func TestUpdateModelFromAPIResponse_EmptyDescription(t *testing.T) {
	model := &BlueprintResourceModel{Description: types.StringNull()}
	blueprint := &client.BlueprintDetailV1{
		Description:     "",
		DeploymentState: client.BlueprintDeploymentStateV1{State: "NOT_DEPLOYED"},
		Steps:           []client.BlueprintStepV1{},
	}

	updateModelFromAPIResponse(model, blueprint)

	if !model.Description.IsNull() {
		t.Error("expected Description to remain null when API returns empty and model is null")
	}
}

func TestUpdateModelFromAPIResponse_NotDeployed(t *testing.T) {
	model := &BlueprintResourceModel{}
	blueprint := &client.BlueprintDetailV1{
		DeploymentState: client.BlueprintDeploymentStateV1{State: "NOT_DEPLOYED"},
		Steps:           []client.BlueprintStepV1{},
	}

	updateModelFromAPIResponse(model, blueprint)

	if model.Deployed.ValueBool() {
		t.Error("expected Deployed to be false for NOT_DEPLOYED state")
	}
}

func TestUpdateModelFromAPIResponse_RawComponents(t *testing.T) {
	model := &BlueprintResourceModel{
		Components: []ComponentModel{
			{Identifier: types.StringValue("com.jamf.ddm.disk-management")},
		},
	}

	blueprint := &client.BlueprintDetailV1{
		DeploymentState: client.BlueprintDeploymentStateV1{State: "DEPLOYED"},
		Steps: []client.BlueprintStepV1{
			{
				Components: []client.BlueprintComponentV1{
					{
						Identifier:    "com.jamf.ddm.disk-management",
						Configuration: json.RawMessage(`{"externalStorage":"deny"}`),
					},
				},
			},
		},
	}

	updateModelFromAPIResponse(model, blueprint)

	if len(model.Components) != 1 {
		t.Fatalf("expected 1 raw component, got %d", len(model.Components))
	}
	if model.Components[0].Identifier.ValueString() != "com.jamf.ddm.disk-management" {
		t.Errorf("expected identifier 'com.jamf.ddm.disk-management', got %q", model.Components[0].Identifier.ValueString())
	}
}

func TestUpdateModelFromAPIResponse_NoSteps(t *testing.T) {
	model := &BlueprintResourceModel{}
	blueprint := &client.BlueprintDetailV1{
		DeploymentState: client.BlueprintDeploymentStateV1{State: "DEPLOYED"},
		Steps:           []client.BlueprintStepV1{},
	}

	updateModelFromAPIResponse(model, blueprint)

	if model.Components != nil {
		t.Error("expected nil components when no steps")
	}
}
