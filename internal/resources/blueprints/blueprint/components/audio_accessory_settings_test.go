// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAudioAccessorySettings_GetIdentifier(t *testing.T) {
	c := &AudioAccessorySettingsComponent{}
	if c.GetIdentifier() != "com.jamf.ddm.audio-accessory-settings" {
		t.Errorf("expected 'com.jamf.ddm.audio-accessory-settings', got %q", c.GetIdentifier())
	}
}

func TestAudioAccessorySettings_ToRawConfiguration_Full(t *testing.T) {
	c := &AudioAccessorySettingsComponent{
		TemporaryPairingDisabled: types.BoolValue(true),
		UnpairingTimePolicy:      types.StringValue("Hour"),
		UnpairingTimeHour:        types.Int64Value(14),
	}

	config, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tp, ok := config["TemporaryPairing"].(map[string]any)
	if !ok {
		t.Fatal("expected TemporaryPairing to be a map")
	}
	if tp["Included"] != true {
		t.Errorf("expected Included true, got %v", tp["Included"])
	}
	if tp["Disabled"] != true {
		t.Errorf("expected Disabled true, got %v", tp["Disabled"])
	}

	cfg, ok := tp["Configuration"].(map[string]any)
	if !ok {
		t.Fatal("expected Configuration to be a map")
	}
	ut, ok := cfg["UnpairingTime"].(map[string]any)
	if !ok {
		t.Fatal("expected UnpairingTime to be a map")
	}
	if ut["Policy"] != "Hour" {
		t.Errorf("expected Policy 'Hour', got %v", ut["Policy"])
	}
	if ut["Hour"] != 14 {
		t.Errorf("expected Hour 14, got %v", ut["Hour"])
	}
}

func TestAudioAccessorySettings_ToRawConfiguration_NullFields(t *testing.T) {
	c := &AudioAccessorySettingsComponent{
		TemporaryPairingDisabled: types.BoolNull(),
		UnpairingTimePolicy:      types.StringNull(),
		UnpairingTimeHour:        types.Int64Null(),
	}

	config, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, exists := config["TemporaryPairing"]; exists {
		t.Error("expected TemporaryPairing to be absent for all-null fields")
	}
}

func TestAudioAccessorySettings_FromRawConfiguration_Full(t *testing.T) {
	raw := map[string]any{
		"TemporaryPairing": map[string]any{
			"Included": true,
			"Disabled": true,
			"Configuration": map[string]any{
				"UnpairingTime": map[string]any{
					"Policy": "Hour",
					"Hour":   float64(18),
				},
			},
		},
	}

	c := &AudioAccessorySettingsComponent{}
	if err := c.FromRawConfiguration(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.TemporaryPairingDisabled.ValueBool() != true {
		t.Errorf("expected TemporaryPairingDisabled true, got %v", c.TemporaryPairingDisabled.ValueBool())
	}
	if c.UnpairingTimePolicy.ValueString() != "Hour" {
		t.Errorf("expected UnpairingTimePolicy 'Hour', got %q", c.UnpairingTimePolicy.ValueString())
	}
	if c.UnpairingTimeHour.ValueInt64() != 18 {
		t.Errorf("expected UnpairingTimeHour 18, got %d", c.UnpairingTimeHour.ValueInt64())
	}
}

func TestAudioAccessorySettings_FromRawConfiguration_NotIncluded(t *testing.T) {
	raw := map[string]any{
		"TemporaryPairing": map[string]any{
			"Included": false,
		},
	}

	c := &AudioAccessorySettingsComponent{}
	if err := c.FromRawConfiguration(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !c.TemporaryPairingDisabled.IsNull() {
		t.Error("expected null TemporaryPairingDisabled when not included")
	}
}

func TestAudioAccessorySettings_FromRawConfiguration_Empty(t *testing.T) {
	c := &AudioAccessorySettingsComponent{}
	if err := c.FromRawConfiguration(map[string]any{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !c.TemporaryPairingDisabled.IsNull() {
		t.Error("expected null TemporaryPairingDisabled for empty config")
	}
	if !c.UnpairingTimePolicy.IsNull() {
		t.Error("expected null UnpairingTimePolicy for empty config")
	}
}

func TestAudioAccessorySettings_Roundtrip(t *testing.T) {
	original := &AudioAccessorySettingsComponent{
		TemporaryPairingDisabled: types.BoolValue(false),
		UnpairingTimePolicy:      types.StringValue("Hour"),
		UnpairingTimeHour:        types.Int64Value(9),
	}

	config, err := original.ToRawConfiguration()
	if err != nil {
		t.Fatalf("ToRawConfiguration error: %v", err)
	}

	jsonBytes, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	restored := &AudioAccessorySettingsComponent{}
	if err := restored.FromRawConfiguration(parsed); err != nil {
		t.Fatalf("FromRawConfiguration error: %v", err)
	}

	if restored.TemporaryPairingDisabled.ValueBool() != false {
		t.Errorf("roundtrip: expected TemporaryPairingDisabled false, got %v", restored.TemporaryPairingDisabled.ValueBool())
	}
	if restored.UnpairingTimePolicy.ValueString() != "Hour" {
		t.Errorf("roundtrip: expected UnpairingTimePolicy 'Hour', got %q", restored.UnpairingTimePolicy.ValueString())
	}
	if restored.UnpairingTimeHour.ValueInt64() != 9 {
		t.Errorf("roundtrip: expected UnpairingTimeHour 9, got %d", restored.UnpairingTimeHour.ValueInt64())
	}
}

func TestAudioAccessorySettings_ToClientComponent(t *testing.T) {
	c := &AudioAccessorySettingsComponent{
		TemporaryPairingDisabled: types.BoolValue(true),
	}

	comp, err := c.ToClientComponent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if comp.Identifier != "com.jamf.ddm.audio-accessory-settings" {
		t.Errorf("expected identifier 'com.jamf.ddm.audio-accessory-settings', got %q", comp.Identifier)
	}
	if comp.Configuration == nil {
		t.Fatal("expected non-nil configuration")
	}
}
