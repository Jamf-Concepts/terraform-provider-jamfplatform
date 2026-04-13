// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"context"
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
		CustomRegexPattern:           types.StringNull(),
		CustomRegexDescription:       types.MapNull(types.StringType),
	}

	config, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config["version"] != "2" {
		t.Errorf("expected version '2', got %v", config["version"])
	}

	changeAtNextAuth := config["ChangeAtNextAuth"].(map[string]any)
	if changeAtNextAuth["Value"] != true {
		t.Errorf("expected ChangeAtNextAuth Value true, got %v", changeAtNextAuth["Value"])
	}
	if changeAtNextAuth["Included"] != true {
		t.Errorf("expected ChangeAtNextAuth Included true, got %v", changeAtNextAuth["Included"])
	}

	failedAttempts := config["FailedAttemptsResetInMinutes"].(map[string]any)
	if failedAttempts["Value"] != 15 {
		t.Errorf("expected FailedAttemptsResetInMinutes Value 15, got %v", failedAttempts["Value"])
	}
	if failedAttempts["Included"] != true {
		t.Errorf("expected FailedAttemptsResetInMinutes Included true, got %v", failedAttempts["Included"])
	}

	maxFailed := config["MaximumFailedAttempts"].(map[string]any)
	if maxFailed["Value"] != 10 {
		t.Errorf("expected MaximumFailedAttempts Value 10, got %v", maxFailed["Value"])
	}

	minLength := config["MinimumLength"].(map[string]any)
	if minLength["Value"] != 8 {
		t.Errorf("expected MinimumLength Value 8, got %v", minLength["Value"])
	}

	requireAlpha := config["RequireAlphanumericPasscode"].(map[string]any)
	if requireAlpha["Value"] != true {
		t.Errorf("expected RequireAlphanumericPasscode Value true, got %v", requireAlpha["Value"])
	}

	requireComplex := config["RequireComplexPasscode"].(map[string]any)
	if requireComplex["Value"] != false {
		t.Errorf("expected RequireComplexPasscode Value false, got %v", requireComplex["Value"])
	}

	requirePasscode := config["RequirePasscode"].(map[string]any)
	if requirePasscode["Value"] != true {
		t.Errorf("expected RequirePasscode Value true, got %v", requirePasscode["Value"])
	}

	if _, exists := config["CustomRegex"]; exists {
		t.Error("expected no CustomRegex key for null custom regex fields")
	}
}

func TestPasscodePolicy_ToRawConfiguration_NullFieldsOmitted(t *testing.T) {
	c := &PasscodePolicyComponent{
		RequirePasscode:        types.BoolValue(true),
		MinimumLength:          types.Int64Value(6),
		CustomRegexPattern:     types.StringNull(),
		CustomRegexDescription: types.MapNull(types.StringType),
	}

	config, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requirePasscode := config["RequirePasscode"].(map[string]any)
	if requirePasscode["Included"] != true {
		t.Errorf("expected RequirePasscode Included true, got %v", requirePasscode["Included"])
	}

	minLength := config["MinimumLength"].(map[string]any)
	if minLength["Included"] != true {
		t.Errorf("expected MinimumLength Included true, got %v", minLength["Included"])
	}

	if _, exists := config["ChangeAtNextAuth"]; exists {
		t.Error("expected ChangeAtNextAuth to be omitted for null")
	}
	if _, exists := config["MaximumFailedAttempts"]; exists {
		t.Error("expected MaximumFailedAttempts to be omitted for null")
	}
	if _, exists := config["PasscodeReuseLimit"]; exists {
		t.Error("expected PasscodeReuseLimit to be omitted for null")
	}
}

func TestPasscodePolicy_ToRawConfiguration_CustomRegex(t *testing.T) {
	descElems := map[string]types.String{
		"default": types.StringValue("Must be 16 digits"),
		"en-US":   types.StringValue("Must be 16 digits long"),
	}
	descMap, _ := types.MapValueFrom(context.Background(), types.StringType, descElems)

	c := &PasscodePolicyComponent{
		RequirePasscode:        types.BoolValue(true),
		CustomRegexPattern:     types.StringValue("[0-9]{16}"),
		CustomRegexDescription: descMap,
	}

	config, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	customRegex, ok := config["CustomRegex"].(map[string]any)
	if !ok {
		t.Fatal("expected CustomRegex to be a map")
	}
	if customRegex["Included"] != true {
		t.Errorf("expected CustomRegex Included true, got %v", customRegex["Included"])
	}
	if customRegex["Regex"] != "[0-9]{16}" {
		t.Errorf("expected Regex '[0-9]{16}', got %v", customRegex["Regex"])
	}
	desc, ok := customRegex["Description"].(map[string]string)
	if !ok {
		t.Fatal("expected Description to be a map[string]string")
	}
	if desc["default"] != "Must be 16 digits" {
		t.Errorf("expected default description, got %q", desc["default"])
	}
}

func TestPasscodePolicy_FromRawConfiguration_V2Format(t *testing.T) {
	raw := map[string]any{
		"version": float64(2),
		"ChangeAtNextAuth": map[string]any{
			"Value":    true,
			"Included": true,
		},
		"FailedAttemptsResetInMinutes": map[string]any{
			"Value":    float64(15),
			"Included": true,
		},
		"MaximumFailedAttempts": map[string]any{
			"Value":    float64(10),
			"Included": true,
		},
		"MaximumGracePeriodInMinutes": map[string]any{
			"Value":    float64(5),
			"Included": true,
		},
		"MaximumInactivityInMinutes": map[string]any{
			"Value":    float64(10),
			"Included": true,
		},
		"MaximumPasscodeAgeInDays": map[string]any{
			"Value":    float64(90),
			"Included": true,
		},
		"MinimumComplexCharacters": map[string]any{
			"Value":    float64(2),
			"Included": true,
		},
		"MinimumLength": map[string]any{
			"Value":    float64(8),
			"Included": true,
		},
		"PasscodeReuseLimit": map[string]any{
			"Value":    float64(5),
			"Included": true,
		},
		"RequireAlphanumericPasscode": map[string]any{
			"Value":    true,
			"Included": true,
		},
		"RequireComplexPasscode": map[string]any{
			"Value":    false,
			"Included": true,
		},
		"RequirePasscode": map[string]any{
			"Value":    true,
			"Included": true,
		},
		"CustomRegex": map[string]any{
			"Regex": "[0-9]{16}",
			"Description": map[string]any{
				"default": "Must be 16 digits",
			},
			"Included": true,
		},
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
	if c.CustomRegexPattern.ValueString() != "[0-9]{16}" {
		t.Errorf("expected CustomRegexPattern '[0-9]{16}', got %q", c.CustomRegexPattern.ValueString())
	}
}

func TestPasscodePolicy_FromRawConfiguration_V1FlatFormat(t *testing.T) {
	raw := map[string]any{
		"ChangeAtNextAuth":             true,
		"FailedAttemptsResetInMinutes": float64(15),
		"MaximumFailedAttempts":        float64(10),
		"MinimumLength":                float64(8),
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
	if c.MinimumLength.ValueInt64() != 8 {
		t.Errorf("expected MinimumLength 8, got %d", c.MinimumLength.ValueInt64())
	}
}

func TestPasscodePolicy_FromRawConfiguration_V2NotIncluded(t *testing.T) {
	raw := map[string]any{
		"version": float64(2),
		"ChangeAtNextAuth": map[string]any{
			"Value":    false,
			"Included": false,
		},
		"MinimumLength": map[string]any{
			"Value":    float64(0),
			"Included": false,
		},
		"RequirePasscode": map[string]any{
			"Value":    false,
			"Included": false,
		},
	}

	c := &PasscodePolicyComponent{}
	if err := c.FromRawConfiguration(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !c.ChangeAtNextAuth.IsNull() {
		t.Error("expected null ChangeAtNextAuth when not included")
	}
	if !c.MinimumLength.IsNull() {
		t.Error("expected null MinimumLength when not included")
	}
	if !c.RequirePasscode.IsNull() {
		t.Error("expected null RequirePasscode when not included")
	}
}

func TestPasscodePolicy_FromRawConfiguration_MissingFields(t *testing.T) {
	raw := map[string]any{
		"RequirePasscode": map[string]any{
			"Value":    true,
			"Included": true,
		},
	}

	c := &PasscodePolicyComponent{}
	if err := c.FromRawConfiguration(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.RequirePasscode.ValueBool() != true {
		t.Errorf("expected RequirePasscode true")
	}
	if !c.ChangeAtNextAuth.IsNull() {
		t.Error("expected ChangeAtNextAuth to be null when missing from raw config")
	}
	if !c.MinimumLength.IsNull() {
		t.Error("expected MinimumLength to be null when missing from raw config")
	}
	if !c.CustomRegexPattern.IsNull() {
		t.Error("expected CustomRegexPattern to be null when missing from raw config")
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
		CustomRegexPattern:           types.StringNull(),
		CustomRegexDescription:       types.MapNull(types.StringType),
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
		RequirePasscode:        types.BoolValue(true),
		MinimumLength:          types.Int64Value(8),
		CustomRegexPattern:     types.StringNull(),
		CustomRegexDescription: types.MapNull(types.StringType),
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
	if config["version"] != "2" {
		t.Errorf("expected version '2' in JSON, got %v", config["version"])
	}
}
