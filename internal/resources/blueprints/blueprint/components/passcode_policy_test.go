// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPasscodePolicy_GetIdentifier(t *testing.T) {
	c := &PasscodePolicyComponent{}
	if c.GetIdentifier() != "com.jamf.ddm.passcode-settings" {
		t.Errorf("expected 'com.jamf.ddm.passcode-settings', got %q", c.GetIdentifier())
	}
}

func TestPasscodePolicy_ToRawConfiguration_AllFields(t *testing.T) {
	c := &PasscodePolicyComponent{
		ChangeAtNextAuth:             types.BoolValue(true),
		FailedAttemptsResetInMinutes: types.Int64Value(15),
		MaximumFailedAttempts:        types.Int64Value(10),
		MaximumGracePeriodInMinutes:  types.Int64Value(5),
		MaximumInactivityInMinutes:   types.Int64Value(10),
		MaximumPasscodeAgeInDays:     types.Int64Value(90),
		MinimumComplexCharacters:     types.Int64Value(2),
		MinimumLength:                types.Int64Value(8),
		PasscodeReuseLimit:           types.Int64Value(5),
		RequireAlphanumericPasscode:  types.BoolValue(true),
		RequireComplexPasscode:       types.BoolValue(false),
		RequirePasscode:              types.BoolValue(true),
	}

	config, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config["ChangeAtNextAuth"] != true {
		t.Errorf("expected ChangeAtNextAuth true, got %v", config["ChangeAtNextAuth"])
	}
	if config["FailedAttemptsResetInMinutes"] != 15 {
		t.Errorf("expected FailedAttemptsResetInMinutes 15, got %v", config["FailedAttemptsResetInMinutes"])
	}
	if config["MaximumFailedAttempts"] != 10 {
		t.Errorf("expected MaximumFailedAttempts 10, got %v", config["MaximumFailedAttempts"])
	}
	if config["MinimumLength"] != 8 {
		t.Errorf("expected MinimumLength 8, got %v", config["MinimumLength"])
	}
	if config["RequireAlphanumericPasscode"] != true {
		t.Errorf("expected RequireAlphanumericPasscode true, got %v", config["RequireAlphanumericPasscode"])
	}
	if config["RequireComplexPasscode"] != false {
		t.Errorf("expected RequireComplexPasscode false, got %v", config["RequireComplexPasscode"])
	}
	if config["RequirePasscode"] != true {
		t.Errorf("expected RequirePasscode true, got %v", config["RequirePasscode"])
	}
}

func TestPasscodePolicy_ToRawConfiguration_NullFieldsOmitted(t *testing.T) {
	c := &PasscodePolicyComponent{
		RequirePasscode: types.BoolValue(true),
		MinimumLength:   types.Int64Value(6),
	}

	config, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config["RequirePasscode"] != true {
		t.Errorf("expected RequirePasscode true, got %v", config["RequirePasscode"])
	}
	if config["MinimumLength"] != 6 {
		t.Errorf("expected MinimumLength 6, got %v", config["MinimumLength"])
	}
	if _, exists := config["ChangeAtNextAuth"]; exists {
		t.Error("expected ChangeAtNextAuth to be omitted for null")
	}
	if _, exists := config["MaximumFailedAttempts"]; exists {
		t.Error("expected MaximumFailedAttempts to be omitted for null")
	}
}

func TestPasscodePolicy_FromRawConfiguration_Float64Values(t *testing.T) {
	raw := map[string]any{
		"ChangeAtNextAuth":             true,
		"FailedAttemptsResetInMinutes": float64(15),
		"MaximumFailedAttempts":        float64(10),
		"MaximumGracePeriodInMinutes":  float64(5),
		"MaximumInactivityInMinutes":   float64(10),
		"MaximumPasscodeAgeInDays":     float64(90),
		"MinimumComplexCharacters":     float64(2),
		"MinimumLength":                float64(8),
		"PasscodeReuseLimit":           float64(5),
		"RequireAlphanumericPasscode":  true,
		"RequireComplexPasscode":       false,
		"RequirePasscode":              true,
	}

	c := &PasscodePolicyComponent{}
	if err := c.FromRawConfiguration(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.ChangeAtNextAuth.ValueBool() != true {
		t.Errorf("expected ChangeAtNextAuth true, got %v", c.ChangeAtNextAuth.ValueBool())
	}
	if c.FailedAttemptsResetInMinutes.ValueInt64() != 15 {
		t.Errorf("expected FailedAttemptsResetInMinutes 15, got %d", c.FailedAttemptsResetInMinutes.ValueInt64())
	}
	if c.MaximumFailedAttempts.ValueInt64() != 10 {
		t.Errorf("expected MaximumFailedAttempts 10, got %d", c.MaximumFailedAttempts.ValueInt64())
	}
	if c.MinimumLength.ValueInt64() != 8 {
		t.Errorf("expected MinimumLength 8, got %d", c.MinimumLength.ValueInt64())
	}
	if c.RequirePasscode.ValueBool() != true {
		t.Errorf("expected RequirePasscode true, got %v", c.RequirePasscode.ValueBool())
	}
	if c.RequireComplexPasscode.ValueBool() != false {
		t.Errorf("expected RequireComplexPasscode false, got %v", c.RequireComplexPasscode.ValueBool())
	}
}

func TestPasscodePolicy_FromRawConfiguration_IntValues(t *testing.T) {
	raw := map[string]any{
		"MinimumLength":   int(12),
		"RequirePasscode": true,
	}

	c := &PasscodePolicyComponent{}
	if err := c.FromRawConfiguration(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.MinimumLength.ValueInt64() != 12 {
		t.Errorf("expected MinimumLength 12, got %d", c.MinimumLength.ValueInt64())
	}
}

func TestPasscodePolicy_FromRawConfiguration_MissingFields(t *testing.T) {
	raw := map[string]any{
		"RequirePasscode": true,
	}

	c := &PasscodePolicyComponent{}
	if err := c.FromRawConfiguration(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.RequirePasscode.ValueBool() != true {
		t.Errorf("expected RequirePasscode true")
	}
	if !c.ChangeAtNextAuth.IsNull() && !c.ChangeAtNextAuth.IsUnknown() {
		t.Error("expected ChangeAtNextAuth to be null or unknown when missing from raw config")
	}
	if !c.MinimumLength.IsNull() && !c.MinimumLength.IsUnknown() {
		t.Error("expected MinimumLength to be null or unknown when missing from raw config")
	}
}

func TestPasscodePolicy_Roundtrip(t *testing.T) {
	original := &PasscodePolicyComponent{
		ChangeAtNextAuth:             types.BoolValue(false),
		FailedAttemptsResetInMinutes: types.Int64Value(30),
		MaximumFailedAttempts:        types.Int64Value(5),
		MaximumGracePeriodInMinutes:  types.Int64Value(10),
		MaximumInactivityInMinutes:   types.Int64Value(15),
		MaximumPasscodeAgeInDays:     types.Int64Value(365),
		MinimumComplexCharacters:     types.Int64Value(1),
		MinimumLength:                types.Int64Value(10),
		PasscodeReuseLimit:           types.Int64Value(3),
		RequireAlphanumericPasscode:  types.BoolValue(false),
		RequireComplexPasscode:       types.BoolValue(true),
		RequirePasscode:              types.BoolValue(true),
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

	restored := &PasscodePolicyComponent{}
	if err := restored.FromRawConfiguration(parsed); err != nil {
		t.Fatalf("FromRawConfiguration error: %v", err)
	}

	if restored.ChangeAtNextAuth.ValueBool() != false {
		t.Errorf("roundtrip: expected ChangeAtNextAuth false, got %v", restored.ChangeAtNextAuth.ValueBool())
	}
	if restored.FailedAttemptsResetInMinutes.ValueInt64() != 30 {
		t.Errorf("roundtrip: expected FailedAttemptsResetInMinutes 30, got %d", restored.FailedAttemptsResetInMinutes.ValueInt64())
	}
	if restored.MaximumFailedAttempts.ValueInt64() != 5 {
		t.Errorf("roundtrip: expected MaximumFailedAttempts 5, got %d", restored.MaximumFailedAttempts.ValueInt64())
	}
	if restored.MinimumLength.ValueInt64() != 10 {
		t.Errorf("roundtrip: expected MinimumLength 10, got %d", restored.MinimumLength.ValueInt64())
	}
	if restored.RequireComplexPasscode.ValueBool() != true {
		t.Errorf("roundtrip: expected RequireComplexPasscode true, got %v", restored.RequireComplexPasscode.ValueBool())
	}
	if restored.RequirePasscode.ValueBool() != true {
		t.Errorf("roundtrip: expected RequirePasscode true, got %v", restored.RequirePasscode.ValueBool())
	}
}

func TestPasscodePolicy_ToClientComponent(t *testing.T) {
	c := &PasscodePolicyComponent{
		RequirePasscode: types.BoolValue(true),
		MinimumLength:   types.Int64Value(8),
	}

	comp, err := c.ToClientComponent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if comp.Identifier != "com.jamf.ddm.passcode-settings" {
		t.Errorf("expected identifier 'com.jamf.ddm.passcode-settings', got %q", comp.Identifier)
	}
	if comp.Configuration == nil {
		t.Fatal("expected non-nil configuration")
	}

	var config map[string]any
	if err := json.Unmarshal(comp.Configuration, &config); err != nil {
		t.Fatalf("failed to parse configuration JSON: %v", err)
	}
	if config["RequirePasscode"] != true {
		t.Errorf("expected RequirePasscode true in JSON, got %v", config["RequirePasscode"])
	}
}
