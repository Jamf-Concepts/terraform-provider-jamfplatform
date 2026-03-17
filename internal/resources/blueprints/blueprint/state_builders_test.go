// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package blueprint

import (
	"encoding/json"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestParseComponentConfiguration_Found(t *testing.T) {
	apiComponents := map[string]jamfplatform.BlueprintComponentV1{
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
	apiComponents := map[string]jamfplatform.BlueprintComponentV1{}

	_, ok := parseComponentConfiguration(apiComponents, "com.jamf.ddm.disk-management")
	if ok {
		t.Error("expected not found for missing component")
	}
}

func TestParseComponentConfiguration_NilConfiguration(t *testing.T) {
	apiComponents := map[string]jamfplatform.BlueprintComponentV1{
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
	apiComponents := map[string]jamfplatform.BlueprintComponentV1{
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
	ctx := t.Context()
	apiComponents := map[string]jamfplatform.BlueprintComponentV1{
		"com.jamf.ddm-configuration-profile": {
			Identifier:    "com.jamf.ddm-configuration-profile",
			Configuration: json.RawMessage(`{"payloadDisplayName":"Test","payloadContent":[{"payloadType":"com.apple.wifi.managed","payloadIdentifier":"test-uuid","SSID_STR":"TestNetwork"}]}`),
		},
	}
	rawIdentifiers := map[string]struct{}{}

	model := &BlueprintResourceModel{}
	updateLegacyPayloadsFromAPI(ctx, model, apiComponents, rawIdentifiers)

	if model.LegacyPayloads.IsNull() {
		t.Fatal("expected non-null legacy payloads")
	}

	raw, err := helpers.TerraformDynamicToJSON(model.LegacyPayloads)
	if err != nil {
		t.Fatalf("failed to convert dynamic to JSON: %v", err)
	}
	items, ok := raw.([]any)
	if !ok {
		t.Fatalf("expected list, got %T", raw)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 payload, got %d", len(items))
	}

	payload, ok := items[0].(map[string]any)
	if !ok {
		t.Fatal("expected payload to be a map")
	}
	if payload["payload_type"] != "com.apple.wifi.managed" {
		t.Errorf("expected payload_type 'com.apple.wifi.managed', got %v", payload["payload_type"])
	}

	settings, ok := payload["settings"].(map[string]any)
	if !ok {
		t.Fatal("expected settings to be a map")
	}
	if settings["SSID_STR"] != "TestNetwork" {
		t.Errorf("expected SSID_STR 'TestNetwork', got %v", settings["SSID_STR"])
	}
	if _, exists := settings["payloadType"]; exists {
		t.Error("payloadType should not be in settings")
	}
}

func TestUpdateLegacyPayloadsFromAPI_NoComponent(t *testing.T) {
	ctx := t.Context()
	apiComponents := map[string]jamfplatform.BlueprintComponentV1{}
	rawIdentifiers := map[string]struct{}{}

	existingDyn, _ := helpers.JSONToTerraformDynamic([]any{map[string]any{"payload_type": "should be cleared"}})
	model := &BlueprintResourceModel{
		LegacyPayloads: existingDyn,
	}
	updateLegacyPayloadsFromAPI(ctx, model, apiComponents, rawIdentifiers)

	if !model.LegacyPayloads.IsNull() {
		t.Error("expected null legacy payloads when component is absent")
	}
}

func TestUpdateLegacyPayloadsFromAPI_HandledAsRaw(t *testing.T) {
	ctx := t.Context()
	apiComponents := map[string]jamfplatform.BlueprintComponentV1{
		"com.jamf.ddm-configuration-profile": {
			Identifier:    "com.jamf.ddm-configuration-profile",
			Configuration: json.RawMessage(`{"payloadContent":[{"payloadType":"com.apple.wifi.managed"}]}`),
		},
	}
	rawIdentifiers := map[string]struct{}{
		"com.jamf.ddm-configuration-profile": {},
	}

	existingDyn, _ := helpers.JSONToTerraformDynamic([]any{map[string]any{"payload_type": "existing"}})
	model := &BlueprintResourceModel{LegacyPayloads: existingDyn}
	updateLegacyPayloadsFromAPI(ctx, model, apiComponents, rawIdentifiers)

	raw, err := helpers.TerraformDynamicToJSON(model.LegacyPayloads)
	if err != nil {
		t.Fatalf("failed to convert: %v", err)
	}
	items := raw.([]any)
	if len(items) != 1 {
		t.Error("expected legacy payloads to remain unchanged when handled as raw")
	}
}

func TestUpdateLegacyPayloadsFromAPI_NoPayloadContent(t *testing.T) {
	ctx := t.Context()
	apiComponents := map[string]jamfplatform.BlueprintComponentV1{
		"com.jamf.ddm-configuration-profile": {
			Identifier:    "com.jamf.ddm-configuration-profile",
			Configuration: json.RawMessage(`{"payloadDisplayName":"Test"}`),
		},
	}
	rawIdentifiers := map[string]struct{}{}

	model := &BlueprintResourceModel{}
	updateLegacyPayloadsFromAPI(ctx, model, apiComponents, rawIdentifiers)

	if !model.LegacyPayloads.IsNull() {
		t.Error("expected null legacy payloads when payloadContent is absent")
	}
}

func TestUpdateModelFromAPIResponse_BasicFields(t *testing.T) {
	ctx := t.Context()
	model := &BlueprintResourceModel{}
	blueprint := &jamfplatform.BlueprintDetailV1{
		ID:          "bp-123",
		Name:        "Test Blueprint",
		Description: "A test blueprint",
		Created:     "2025-01-01T00:00:00Z",
		Updated:     "2025-01-02T00:00:00Z",
		DeploymentState: jamfplatform.BlueprintDeploymentStateV1{
			State: "DEPLOYED",
		},
		Scope: jamfplatform.BlueprintUpdateScopeV1{
			DeviceGroups: []string{"group-1", "group-2"},
		},
		Steps: []jamfplatform.BlueprintStepV1{},
	}

	updateModelFromAPIResponse(ctx, model, blueprint)

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
	ctx := t.Context()
	model := &BlueprintResourceModel{Description: types.StringNull()}
	blueprint := &jamfplatform.BlueprintDetailV1{
		Description:     "",
		DeploymentState: jamfplatform.BlueprintDeploymentStateV1{State: "NOT_DEPLOYED"},
		Steps:           []jamfplatform.BlueprintStepV1{},
	}

	updateModelFromAPIResponse(ctx, model, blueprint)

	if !model.Description.IsNull() {
		t.Error("expected Description to remain null when API returns empty and model is null")
	}
}

func TestUpdateModelFromAPIResponse_NotDeployed(t *testing.T) {
	ctx := t.Context()
	model := &BlueprintResourceModel{}
	blueprint := &jamfplatform.BlueprintDetailV1{
		DeploymentState: jamfplatform.BlueprintDeploymentStateV1{State: "NOT_DEPLOYED"},
		Steps:           []jamfplatform.BlueprintStepV1{},
	}

	updateModelFromAPIResponse(ctx, model, blueprint)

	if model.Deployed.ValueBool() {
		t.Error("expected Deployed to be false for NOT_DEPLOYED state")
	}
}

func TestUpdateModelFromAPIResponse_RawComponents(t *testing.T) {
	ctx := t.Context()
	model := &BlueprintResourceModel{
		Components: []ComponentModel{
			{Identifier: types.StringValue("com.jamf.ddm.disk-management")},
		},
	}

	blueprint := &jamfplatform.BlueprintDetailV1{
		DeploymentState: jamfplatform.BlueprintDeploymentStateV1{State: "DEPLOYED"},
		Steps: []jamfplatform.BlueprintStepV1{
			{
				Components: []jamfplatform.BlueprintComponentV1{
					{
						Identifier:    "com.jamf.ddm.disk-management",
						Configuration: json.RawMessage(`{"externalStorage":"deny"}`),
					},
				},
			},
		},
	}

	updateModelFromAPIResponse(ctx, model, blueprint)

	if len(model.Components) != 1 {
		t.Fatalf("expected 1 raw component, got %d", len(model.Components))
	}
	if model.Components[0].Identifier.ValueString() != "com.jamf.ddm.disk-management" {
		t.Errorf("expected identifier 'com.jamf.ddm.disk-management', got %q", model.Components[0].Identifier.ValueString())
	}
}

func TestUpdateModelFromAPIResponse_NoSteps(t *testing.T) {
	ctx := t.Context()
	model := &BlueprintResourceModel{}
	blueprint := &jamfplatform.BlueprintDetailV1{
		DeploymentState: jamfplatform.BlueprintDeploymentStateV1{State: "DEPLOYED"},
		Steps:           []jamfplatform.BlueprintStepV1{},
	}

	updateModelFromAPIResponse(ctx, model, blueprint)

	if model.Components != nil {
		t.Error("expected nil components when no steps")
	}
}
