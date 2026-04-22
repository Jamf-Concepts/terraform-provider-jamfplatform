// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCustomDeclarations_GetIdentifier(t *testing.T) {
	c := &CustomDeclarationsComponent{}
	if c.GetIdentifier() != "com.jamf.ddm.custom-declarations" {
		t.Errorf("expected 'com.jamf.ddm.custom-declarations', got %q", c.GetIdentifier())
	}
}

func TestCustomDeclarations_ToRawConfiguration_WithDeclarations(t *testing.T) {
	c := &CustomDeclarationsComponent{
		Declarations: []CustomDeclarationModel{
			{
				ChannelType: types.StringValue("SYSTEM"),
				Kind:        types.StringValue("CONFIGURATION"),
				Payload:     types.StringValue(`{"key":"value"}`),
				Type:        types.StringValue("com.apple.configuration.test"),
			},
		},
	}

	rawCfg, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var config map[string]any
	if err := json.Unmarshal(rawCfg, &config); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	declarations, ok := config["declarations"].([]any)
	if !ok {
		t.Fatal("expected declarations to be a slice")
	}
	if len(declarations) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(declarations))
	}

	decl, ok := declarations[0].(map[string]any)
	if !ok {
		t.Fatal("expected declaration to be a map")
	}
	if decl["channelType"] != "SYSTEM" {
		t.Errorf("expected channelType 'SYSTEM', got %v", decl["channelType"])
	}
	if decl["kind"] != "CONFIGURATION" {
		t.Errorf("expected kind 'CONFIGURATION', got %v", decl["kind"])
	}
	if decl["payloadKey"] != float64(1) {
		t.Errorf("expected payloadKey 1, got %v", decl["payloadKey"])
	}
	if decl["type"] != "com.apple.configuration.test" {
		t.Errorf("expected type 'com.apple.configuration.test', got %v", decl["type"])
	}

	payload, ok := decl["payload"].(map[string]any)
	if !ok {
		t.Fatal("expected payload to be a map")
	}
	if payload["key"] != "value" {
		t.Errorf("expected payload key 'value', got %v", payload["key"])
	}
}

func TestCustomDeclarations_ToRawConfiguration_Empty(t *testing.T) {
	c := &CustomDeclarationsComponent{}

	rawCfg, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var config map[string]any
	if err := json.Unmarshal(rawCfg, &config); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	if _, exists := config["declarations"]; exists {
		t.Error("expected no declarations key for empty component")
	}
}

func TestCustomDeclarations_ToRawConfiguration_InvalidPayloadJSON(t *testing.T) {
	c := &CustomDeclarationsComponent{
		Declarations: []CustomDeclarationModel{
			{
				ChannelType: types.StringValue("SYSTEM"),
				Kind:        types.StringValue("CONFIGURATION"),
				Payload:     types.StringValue("not-valid-json"),
				Type:        types.StringValue("com.apple.configuration.test"),
			},
		},
	}

	_, err := c.ToRawConfiguration()
	if err == nil {
		t.Error("expected error for invalid payload JSON")
	}
}

func TestCustomDeclarations_FromRawConfiguration(t *testing.T) {
	rawMap := map[string]any{
		"declarations": []any{
			map[string]any{
				"channelType": "USER",
				"kind":        "ASSET",
				"payload":     map[string]any{"setting": true},
				"payloadKey":  float64(1),
				"type":        "com.apple.asset.test",
			},
		},
	}
	raw, _ := json.Marshal(rawMap)

	c := &CustomDeclarationsComponent{}
	if err := c.FromRawConfiguration(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(c.Declarations) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(c.Declarations))
	}

	decl := c.Declarations[0]
	if decl.ChannelType.ValueString() != "USER" {
		t.Errorf("expected ChannelType 'USER', got %q", decl.ChannelType.ValueString())
	}
	if decl.Kind.ValueString() != "ASSET" {
		t.Errorf("expected Kind 'ASSET', got %q", decl.Kind.ValueString())
	}
	if decl.Type.ValueString() != "com.apple.asset.test" {
		t.Errorf("expected Type 'com.apple.asset.test', got %q", decl.Type.ValueString())
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(decl.Payload.ValueString()), &payload); err != nil {
		t.Fatalf("failed to parse payload JSON: %v", err)
	}
	if payload["setting"] != true {
		t.Errorf("expected payload setting true, got %v", payload["setting"])
	}
}

func TestCustomDeclarations_FromRawConfiguration_Empty(t *testing.T) {
	c := &CustomDeclarationsComponent{}
	if err := c.FromRawConfiguration(json.RawMessage("{}")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(c.Declarations) != 0 {
		t.Errorf("expected 0 declarations, got %d", len(c.Declarations))
	}
}

func TestCustomDeclarations_ToRawConfiguration_PayloadKeyIndexed(t *testing.T) {
	c := &CustomDeclarationsComponent{
		Declarations: []CustomDeclarationModel{
			{ChannelType: types.StringValue("SYSTEM"), Kind: types.StringValue("CONFIGURATION"), Payload: types.StringValue(`{}`), Type: types.StringValue("com.apple.configuration.a")},
			{ChannelType: types.StringValue("SYSTEM"), Kind: types.StringValue("CONFIGURATION"), Payload: types.StringValue(`{}`), Type: types.StringValue("com.apple.configuration.b")},
		},
	}

	rawCfg, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var config map[string]any
	if err := json.Unmarshal(rawCfg, &config); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	declarations := config["declarations"].([]any)
	for i, raw := range declarations {
		decl := raw.(map[string]any)
		if decl["payloadKey"] != float64(i+1) {
			t.Errorf("declaration[%d]: expected payloadKey %d, got %v", i, i+1, decl["payloadKey"])
		}
	}
}

func TestCustomDeclarations_Roundtrip(t *testing.T) {
	original := &CustomDeclarationsComponent{
		Declarations: []CustomDeclarationModel{
			{
				ChannelType: types.StringValue("SYSTEM"),
				Kind:        types.StringValue("CONFIGURATION"),
				Payload:     types.StringValue(`{"enabled":true,"count":5}`),
				Type:        types.StringValue("com.apple.configuration.wifi"),
			},
			{
				ChannelType: types.StringValue("USER"),
				Kind:        types.StringValue("ASSET"),
				Payload:     types.StringValue(`{"name":"test"}`),
				Type:        types.StringValue("com.apple.asset.data"),
			},
		},
	}

	rawCfg, err := original.ToRawConfiguration()
	if err != nil {
		t.Fatalf("ToRawConfiguration error: %v", err)
	}

	restored := &CustomDeclarationsComponent{}
	if err := restored.FromRawConfiguration(rawCfg); err != nil {
		t.Fatalf("FromRawConfiguration error: %v", err)
	}

	if len(restored.Declarations) != 2 {
		t.Fatalf("roundtrip: expected 2 declarations, got %d", len(restored.Declarations))
	}

	for i, decl := range restored.Declarations {
		if decl.ChannelType.ValueString() != original.Declarations[i].ChannelType.ValueString() {
			t.Errorf("roundtrip[%d]: expected ChannelType %q, got %q", i, original.Declarations[i].ChannelType.ValueString(), decl.ChannelType.ValueString())
		}
		if decl.Kind.ValueString() != original.Declarations[i].Kind.ValueString() {
			t.Errorf("roundtrip[%d]: expected Kind %q, got %q", i, original.Declarations[i].Kind.ValueString(), decl.Kind.ValueString())
		}
		if decl.Type.ValueString() != original.Declarations[i].Type.ValueString() {
			t.Errorf("roundtrip[%d]: expected Type %q, got %q", i, original.Declarations[i].Type.ValueString(), decl.Type.ValueString())
		}
	}
}

func TestCustomDeclarations_ToClientComponent(t *testing.T) {
	c := &CustomDeclarationsComponent{
		Declarations: []CustomDeclarationModel{
			{
				ChannelType: types.StringValue("SYSTEM"),
				Kind:        types.StringValue("CONFIGURATION"),
				Payload:     types.StringValue(`{"key":"val"}`),
				Type:        types.StringValue("com.apple.configuration.test"),
			},
		},
	}

	comp, err := c.ToClientComponent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if comp.Identifier != "com.jamf.ddm.custom-declarations" {
		t.Errorf("expected identifier 'com.jamf.ddm.custom-declarations', got %q", comp.Identifier)
	}
	if comp.Configuration == nil {
		t.Fatal("expected non-nil configuration")
	}
}
