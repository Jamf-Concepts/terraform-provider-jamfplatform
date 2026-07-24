// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package blueprint

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/blueprints"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestParseComponentConfiguration_Found(t *testing.T) {
	apiComponents := map[string]blueprints.Component{
		"com.jamf.ddm.disk-management": {
			Identifier:    "com.jamf.ddm.disk-management",
			Configuration: json.RawMessage(`{"externalStorage":"deny","networkStorage":"deny"}`),
		},
	}

	rawConfig, ok := parseComponentConfiguration(apiComponents, "com.jamf.ddm.disk-management")
	if !ok {
		t.Fatal("expected to find configuration")
	}
	var config map[string]any
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		t.Fatalf("failed to unmarshal configuration: %v", err)
	}
	if config["externalStorage"] != "deny" {
		t.Errorf("expected externalStorage 'deny', got %v", config["externalStorage"])
	}
}

func TestParseComponentConfiguration_NotFound(t *testing.T) {
	apiComponents := map[string]blueprints.Component{}

	_, ok := parseComponentConfiguration(apiComponents, "com.jamf.ddm.disk-management")
	if ok {
		t.Error("expected not found for missing component")
	}
}

func TestParseComponentConfiguration_NilConfiguration(t *testing.T) {
	apiComponents := map[string]blueprints.Component{
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
	apiComponents := map[string]blueprints.Component{
		"com.jamf.ddm.disk-management": {
			Identifier:    "com.jamf.ddm.disk-management",
			Configuration: json.RawMessage(`{invalid`),
		},
	}

	rawConfig, ok := parseComponentConfiguration(apiComponents, "com.jamf.ddm.disk-management")
	if !ok {
		t.Error("expected to find raw bytes even for invalid JSON — validation is caller's responsibility")
	}
	if string(rawConfig) != "{invalid" {
		t.Errorf("expected raw bytes returned as-is, got %q", string(rawConfig))
	}
}

func TestFlattenFlatLegacyPayloads_WithPayloads(t *testing.T) {
	apiComponents := map[string]blueprints.Component{
		"com.jamf.ddm-configuration-profile": {
			Identifier:    "com.jamf.ddm-configuration-profile",
			Configuration: json.RawMessage(`{"payloadDisplayName":"Test","payloadContent":[{"payloadType":"com.apple.wifi.managed","payloadIdentifier":"test-uuid","SSID_STR":"TestNetwork"}]}`),
		},
	}

	got := flattenFlatLegacyPayloads(types.DynamicNull(), apiComponents, map[string]struct{}{})

	if got.IsNull() {
		t.Fatal("expected non-null legacy payloads")
	}

	raw, err := helpers.TerraformDynamicToJSON(got)
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

func TestFlattenFlatLegacyPayloads_NoComponent(t *testing.T) {
	existingDyn, _ := helpers.JSONToTerraformDynamic([]any{map[string]any{"payload_type": "should be cleared"}})
	got := flattenFlatLegacyPayloads(existingDyn, map[string]blueprints.Component{}, map[string]struct{}{})

	if !got.IsNull() {
		t.Error("expected null legacy payloads when component is absent")
	}
}

func TestFlattenFlatLegacyPayloads_HandledAsRaw(t *testing.T) {
	apiComponents := map[string]blueprints.Component{
		"com.jamf.ddm-configuration-profile": {
			Identifier:    "com.jamf.ddm-configuration-profile",
			Configuration: json.RawMessage(`{"payloadContent":[{"payloadType":"com.apple.wifi.managed"}]}`),
		},
	}
	rawIdentifiers := map[string]struct{}{
		"com.jamf.ddm-configuration-profile": {},
	}

	existingDyn, _ := helpers.JSONToTerraformDynamic([]any{map[string]any{"payload_type": "existing"}})
	got := flattenFlatLegacyPayloads(existingDyn, apiComponents, rawIdentifiers)

	raw, err := helpers.TerraformDynamicToJSON(got)
	if err != nil {
		t.Fatalf("failed to convert: %v", err)
	}
	items := raw.([]any)
	if len(items) != 1 {
		t.Error("expected legacy payloads to remain unchanged when handled as raw")
	}
}

func TestFlattenFlatLegacyPayloads_NoPayloadContent(t *testing.T) {
	apiComponents := map[string]blueprints.Component{
		"com.jamf.ddm-configuration-profile": {
			Identifier:    "com.jamf.ddm-configuration-profile",
			Configuration: json.RawMessage(`{"payloadDisplayName":"Test"}`),
		},
	}

	got := flattenFlatLegacyPayloads(types.DynamicNull(), apiComponents, map[string]struct{}{})

	if !got.IsNull() {
		t.Error("expected null legacy payloads when payloadContent is absent")
	}
}

func TestUpdateModelFromAPIResponse_BasicFields(t *testing.T) {
	ctx := t.Context()
	model := &BlueprintResourceModel{}
	desc := "A test blueprint"
	created, _ := time.Parse(time.RFC3339, "2025-01-01T00:00:00Z")
	updated, _ := time.Parse(time.RFC3339, "2025-01-02T00:00:00Z")
	blueprint := &blueprints.BlueprintDetail{
		ID:          "bp-123",
		Name:        "Test Blueprint",
		Description: &desc,
		Created:     created,
		Updated:     updated,
		DeploymentState: &blueprints.DeploymentState{
			State: "DEPLOYED",
		},
		Scope: &blueprints.BlueprintScope{
			DeviceGroups: []string{"group-1", "group-2"},
		},
		Steps: []blueprints.BlueprintStep{},
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
	blueprint := &blueprints.BlueprintDetail{
		Description:     nil,
		DeploymentState: &blueprints.DeploymentState{State: "NOT_DEPLOYED"},
		Steps:           []blueprints.BlueprintStep{},
	}

	updateModelFromAPIResponse(ctx, model, blueprint)

	if !model.Description.IsNull() {
		t.Error("expected Description to remain null when API returns empty and model is null")
	}
}

func TestUpdateModelFromAPIResponse_NotDeployed(t *testing.T) {
	ctx := t.Context()
	model := &BlueprintResourceModel{}
	blueprint := &blueprints.BlueprintDetail{
		DeploymentState: &blueprints.DeploymentState{State: "NOT_DEPLOYED"},
		Steps:           []blueprints.BlueprintStep{},
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

	blueprint := &blueprints.BlueprintDetail{
		DeploymentState: &blueprints.DeploymentState{State: "DEPLOYED"},
		Steps: []blueprints.BlueprintStep{
			{
				Components: []blueprints.Component{
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
	blueprint := &blueprints.BlueprintDetail{
		DeploymentState: &blueprints.DeploymentState{State: "DEPLOYED"},
		Steps:           []blueprints.BlueprintStep{},
	}

	updateModelFromAPIResponse(ctx, model, blueprint)

	if model.Components != nil {
		t.Error("expected nil components when no steps")
	}
}

// TestUpdateLegacyPayloadsFromAPI_PreservesConfigShapeOnMatch pins the issue
// #282 fix: when the server response is semantically identical to the incoming
// (configuration-shaped) value, the reader keeps that value verbatim so the
// dynamic null-typing does not manufacture a diff.
func TestFlattenFlatLegacyPayloads_PreservesConfigShapeOnMatch(t *testing.T) {
	prior, err := helpers.JSONToTerraformDynamic([]any{
		map[string]any{
			"payload_type": "com.apple.notificationsettings",
			"settings": map[string]any{
				"NotificationSettings": []any{
					map[string]any{
						"BundleIdentifier":     "com.apple.tips",
						"AlertType":            float64(0),
						"BadgesEnabled":        nil,
						"NotificationsEnabled": false,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to build prior value: %v", err)
	}

	apiComponents := map[string]blueprints.Component{
		"com.jamf.ddm-configuration-profile": {
			Identifier:    "com.jamf.ddm-configuration-profile",
			Configuration: json.RawMessage(`{"payloadContent":[{"payloadType":"com.apple.notificationsettings","payloadIdentifier":"generated-uuid","NotificationSettings":[{"BundleIdentifier":"com.apple.tips","AlertType":0,"BadgesEnabled":null,"NotificationsEnabled":false}]}]}`),
		},
	}

	got := flattenFlatLegacyPayloads(prior, apiComponents, map[string]struct{}{})

	if !got.Equal(prior) {
		t.Errorf("expected config-shaped prior value to be preserved on semantic match, got %#v", got)
	}
}

// TestUpdateLegacyPayloadsFromAPI_OverwritesOnMismatch verifies that a genuine
// server-side difference is still surfaced rather than masked by the reconcile.
func TestFlattenFlatLegacyPayloads_OverwritesOnMismatch(t *testing.T) {
	prior, err := helpers.JSONToTerraformDynamic([]any{
		map[string]any{
			"payload_type": "com.apple.notificationsettings",
			"settings":     map[string]any{"BundleIdentifier": "com.apple.tips.old"},
		},
	})
	if err != nil {
		t.Fatalf("failed to build prior value: %v", err)
	}

	apiComponents := map[string]blueprints.Component{
		"com.jamf.ddm-configuration-profile": {
			Identifier:    "com.jamf.ddm-configuration-profile",
			Configuration: json.RawMessage(`{"payloadContent":[{"payloadType":"com.apple.notificationsettings","BundleIdentifier":"com.apple.tips.new"}]}`),
		},
	}

	got := flattenFlatLegacyPayloads(prior, apiComponents, map[string]struct{}{})

	if got.Equal(prior) {
		t.Fatal("expected differing server value to overwrite the prior value")
	}

	raw, err := helpers.TerraformDynamicToJSON(got)
	if err != nil {
		t.Fatalf("failed to convert result: %v", err)
	}
	settings := raw.([]any)[0].(map[string]any)["settings"].(map[string]any)
	if settings["BundleIdentifier"] != "com.apple.tips.new" {
		t.Errorf("expected overwritten value 'com.apple.tips.new', got %v", settings["BundleIdentifier"])
	}
}
