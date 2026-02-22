// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package blueprint

import (
	"encoding/json"
	"testing"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func TestCollectLegacyPayloadsString_ValidJSON(t *testing.T) {
	r := &BlueprintResource{}
	var components []client.BlueprintComponentV1
	var diags diag.Diagnostics

	r.collectLegacyPayloadsString(&components, &diags, `[{"payloadType":"com.apple.wifi.managed","payloadIdentifier":"com.example.wifi"}]`, "My Blueprint")

	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if len(components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(components))
	}
	if components[0].Identifier != "com.jamf.ddm-configuration-profile" {
		t.Errorf("expected identifier 'com.jamf.ddm-configuration-profile', got %q", components[0].Identifier)
	}

	var config map[string]any
	if err := json.Unmarshal(components[0].Configuration, &config); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}
	if config["payloadDisplayName"] != "My Blueprint" {
		t.Errorf("expected payloadDisplayName 'My Blueprint', got %v", config["payloadDisplayName"])
	}
}

func TestCollectLegacyPayloadsString_InvalidJSON(t *testing.T) {
	r := &BlueprintResource{}
	var components []client.BlueprintComponentV1
	var diags diag.Diagnostics

	r.collectLegacyPayloadsString(&components, &diags, `not-valid-json`, "Blueprint")

	if !diags.HasError() {
		t.Error("expected error for invalid JSON")
	}
	if len(components) != 0 {
		t.Errorf("expected 0 components on error, got %d", len(components))
	}
}

func TestCollectLegacyPayloadsString_EmptyArray(t *testing.T) {
	r := &BlueprintResource{}
	var components []client.BlueprintComponentV1
	var diags diag.Diagnostics

	r.collectLegacyPayloadsString(&components, &diags, `[]`, "Empty Blueprint")

	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if len(components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(components))
	}

	var config map[string]any
	if err := json.Unmarshal(components[0].Configuration, &config); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}
	payloadContent, ok := config["payloadContent"].([]any)
	if !ok {
		t.Fatal("expected payloadContent to be an array")
	}
	if len(payloadContent) != 0 {
		t.Errorf("expected empty payload content, got %d items", len(payloadContent))
	}
}
