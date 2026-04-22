// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSoftwareUpdate_GetIdentifier(t *testing.T) {
	c := &SoftwareUpdateComponent{}
	if c.GetIdentifier() != "com.jamf.ddm.sw-updates" {
		t.Errorf("expected 'com.jamf.ddm.sw-updates', got %q", c.GetIdentifier())
	}
}

func TestSoftwareUpdate_ToRawConfiguration_Automatic_Latest(t *testing.T) {
	c := &SoftwareUpdateComponent{
		DeploymentTime:      types.StringValue("14:30"),
		EnforceAfterDays:    types.Int64Value(7),
		IgnoreMajorVersions: types.BoolValue(false),
		DetailsURLValue:     types.StringValue("https://example.com/update"),
	}

	rawCfg, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var config map[string]any
	if err := json.Unmarshal(rawCfg, &config); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	if config["enforcementType"] != "AUTOMATIC" {
		t.Errorf("expected enforcementType 'AUTOMATIC', got %v", config["enforcementType"])
	}
	if config["strategy"] != "LATEST" {
		t.Errorf("expected strategy 'LATEST', got %v", config["strategy"])
	}
	if config["deploymentTime"] != "14:30" {
		t.Errorf("expected deploymentTime '14:30', got %v", config["deploymentTime"])
	}
	if config["enforceAfterDays"] != float64(7) {
		t.Errorf("expected enforceAfterDays 7, got %v", config["enforceAfterDays"])
	}

	detailsURL, ok := config["detailsURL"].(map[string]any)
	if !ok {
		t.Fatal("expected detailsURL to be a map")
	}
	if detailsURL["Value"] != "https://example.com/update" {
		t.Errorf("expected detailsURL Value 'https://example.com/update', got %v", detailsURL["Value"])
	}
	if detailsURL["Included"] != true {
		t.Errorf("expected detailsURL Included true, got %v", detailsURL["Included"])
	}
}

func TestSoftwareUpdate_ToRawConfiguration_Automatic_Semantic(t *testing.T) {
	c := &SoftwareUpdateComponent{
		DeploymentTime:      types.StringValue("09:00"),
		EnforceAfterDays:    types.Int64Value(14),
		IgnoreMajorVersions: types.BoolValue(true),
	}

	rawCfg, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var config map[string]any
	if err := json.Unmarshal(rawCfg, &config); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	if config["enforcementType"] != "AUTOMATIC" {
		t.Errorf("expected enforcementType 'AUTOMATIC', got %v", config["enforcementType"])
	}
	if config["strategy"] != "SEMANTIC" {
		t.Errorf("expected strategy 'SEMANTIC', got %v", config["strategy"])
	}

	rules, ok := config["rules"].(map[string]any)
	if !ok {
		t.Fatal("expected rules to be a map")
	}
	minor, ok := rules["minor"].(map[string]any)
	if !ok {
		t.Fatal("expected rules.minor to be a map")
	}
	if minor["deploymentTime"] != "09:00" {
		t.Errorf("expected minor deploymentTime '09:00', got %v", minor["deploymentTime"])
	}
	if minor["enforceAfterDays"] != float64(14) {
		t.Errorf("expected minor enforceAfterDays 14, got %v", minor["enforceAfterDays"])
	}

	if _, exists := config["deploymentTime"]; exists {
		t.Error("expected no top-level deploymentTime for SEMANTIC strategy")
	}
}

func TestSoftwareUpdate_ToRawConfiguration_Manual(t *testing.T) {
	c := &SoftwareUpdateComponent{
		TargetOSVersion:     types.StringValue("15.2.1"),
		TargetLocalDateTime: types.StringValue("2026-03-01T10:00:00"),
	}

	rawCfg, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var config map[string]any
	if err := json.Unmarshal(rawCfg, &config); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	if config["enforcementType"] != "MANUAL" {
		t.Errorf("expected enforcementType 'MANUAL', got %v", config["enforcementType"])
	}
	if config["targetOSVersion"] != "15.2.1" {
		t.Errorf("expected targetOSVersion '15.2.1', got %v", config["targetOSVersion"])
	}
	if config["targetLocalDateTime"] != "2026-03-01T10:00:00" {
		t.Errorf("expected targetLocalDateTime '2026-03-01T10:00:00', got %v", config["targetLocalDateTime"])
	}
}

func TestSoftwareUpdate_FromRawConfiguration_Automatic_Latest(t *testing.T) {
	rawMap := map[string]any{
		"enforcementType":  "AUTOMATIC",
		"strategy":         "LATEST",
		"deploymentTime":   "14:30",
		"enforceAfterDays": float64(7),
		"detailsURL": map[string]any{
			"Value":    "https://example.com/details",
			"Included": true,
		},
	}
	raw, _ := json.Marshal(rawMap)

	c := &SoftwareUpdateComponent{}
	if err := c.FromRawConfiguration(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.EnforcementType.ValueString() != "AUTOMATIC" {
		t.Errorf("expected EnforcementType 'AUTOMATIC', got %q", c.EnforcementType.ValueString())
	}
	if c.DeploymentTime.ValueString() != "14:30" {
		t.Errorf("expected DeploymentTime '14:30', got %q", c.DeploymentTime.ValueString())
	}
	if c.EnforceAfterDays.ValueInt64() != 7 {
		t.Errorf("expected EnforceAfterDays 7, got %d", c.EnforceAfterDays.ValueInt64())
	}
	if c.IgnoreMajorVersions.ValueBool() != false {
		t.Errorf("expected IgnoreMajorVersions false for LATEST, got %v", c.IgnoreMajorVersions.ValueBool())
	}
	if c.DetailsURLValue.ValueString() != "https://example.com/details" {
		t.Errorf("expected DetailsURLValue 'https://example.com/details', got %q", c.DetailsURLValue.ValueString())
	}
}

func TestSoftwareUpdate_FromRawConfiguration_Automatic_Semantic(t *testing.T) {
	rawMap := map[string]any{
		"enforcementType": "AUTOMATIC",
		"strategy":        "SEMANTIC",
		"rules": map[string]any{
			"minor": map[string]any{
				"deploymentTime":   "09:00",
				"enforceAfterDays": float64(14),
			},
		},
	}
	raw, _ := json.Marshal(rawMap)

	c := &SoftwareUpdateComponent{}
	if err := c.FromRawConfiguration(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.IgnoreMajorVersions.ValueBool() != true {
		t.Errorf("expected IgnoreMajorVersions true for SEMANTIC, got %v", c.IgnoreMajorVersions.ValueBool())
	}
	if c.DeploymentTime.ValueString() != "09:00" {
		t.Errorf("expected DeploymentTime '09:00', got %q", c.DeploymentTime.ValueString())
	}
	if c.EnforceAfterDays.ValueInt64() != 14 {
		t.Errorf("expected EnforceAfterDays 14, got %d", c.EnforceAfterDays.ValueInt64())
	}
}

func TestSoftwareUpdate_FromRawConfiguration_Manual(t *testing.T) {
	rawMap := map[string]any{
		"enforcementType":     "MANUAL",
		"targetOSVersion":     "15.2.1",
		"targetLocalDateTime": "2026-03-01T10:00:00",
	}
	raw, _ := json.Marshal(rawMap)

	c := &SoftwareUpdateComponent{}
	if err := c.FromRawConfiguration(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.EnforcementType.ValueString() != "MANUAL" {
		t.Errorf("expected EnforcementType 'MANUAL', got %q", c.EnforcementType.ValueString())
	}
	if c.TargetOSVersion.ValueString() != "15.2.1" {
		t.Errorf("expected TargetOSVersion '15.2.1', got %q", c.TargetOSVersion.ValueString())
	}
	if c.TargetLocalDateTime.ValueString() != "2026-03-01T10:00:00" {
		t.Errorf("expected TargetLocalDateTime '2026-03-01T10:00:00', got %q", c.TargetLocalDateTime.ValueString())
	}
}

func TestSoftwareUpdate_FromRawConfiguration_Empty(t *testing.T) {
	c := &SoftwareUpdateComponent{}
	if err := c.FromRawConfiguration(json.RawMessage("{}")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.IgnoreMajorVersions.IsNull() != true {
		t.Error("expected null IgnoreMajorVersions for empty config")
	}
}

func TestSoftwareUpdate_Roundtrip_Automatic(t *testing.T) {
	original := &SoftwareUpdateComponent{
		DeploymentTime:      types.StringValue("10:00"),
		EnforceAfterDays:    types.Int64Value(5),
		IgnoreMajorVersions: types.BoolValue(false),
		DetailsURLValue:     types.StringValue("https://test.com"),
	}

	rawCfg, err := original.ToRawConfiguration()
	if err != nil {
		t.Fatalf("ToRawConfiguration error: %v", err)
	}

	restored := &SoftwareUpdateComponent{}
	if err := restored.FromRawConfiguration(rawCfg); err != nil {
		t.Fatalf("FromRawConfiguration error: %v", err)
	}

	if restored.EnforcementType.ValueString() != "AUTOMATIC" {
		t.Errorf("roundtrip: expected EnforcementType 'AUTOMATIC', got %q", restored.EnforcementType.ValueString())
	}
	if restored.DeploymentTime.ValueString() != "10:00" {
		t.Errorf("roundtrip: expected DeploymentTime '10:00', got %q", restored.DeploymentTime.ValueString())
	}
	if restored.EnforceAfterDays.ValueInt64() != 5 {
		t.Errorf("roundtrip: expected EnforceAfterDays 5, got %d", restored.EnforceAfterDays.ValueInt64())
	}
	if restored.IgnoreMajorVersions.ValueBool() != false {
		t.Errorf("roundtrip: expected IgnoreMajorVersions false, got %v", restored.IgnoreMajorVersions.ValueBool())
	}
	if restored.DetailsURLValue.ValueString() != "https://test.com" {
		t.Errorf("roundtrip: expected DetailsURLValue 'https://test.com', got %q", restored.DetailsURLValue.ValueString())
	}
}

func TestSoftwareUpdate_Roundtrip_Manual(t *testing.T) {
	original := &SoftwareUpdateComponent{
		TargetOSVersion:     types.StringValue("16.0"),
		TargetLocalDateTime: types.StringValue("2026-06-15T08:00:00"),
	}

	rawCfg, err := original.ToRawConfiguration()
	if err != nil {
		t.Fatalf("ToRawConfiguration error: %v", err)
	}

	restored := &SoftwareUpdateComponent{}
	if err := restored.FromRawConfiguration(rawCfg); err != nil {
		t.Fatalf("FromRawConfiguration error: %v", err)
	}

	if restored.EnforcementType.ValueString() != "MANUAL" {
		t.Errorf("roundtrip: expected EnforcementType 'MANUAL', got %q", restored.EnforcementType.ValueString())
	}
	if restored.TargetOSVersion.ValueString() != "16.0" {
		t.Errorf("roundtrip: expected TargetOSVersion '16.0', got %q", restored.TargetOSVersion.ValueString())
	}
	if restored.TargetLocalDateTime.ValueString() != "2026-06-15T08:00:00" {
		t.Errorf("roundtrip: expected TargetLocalDateTime '2026-06-15T08:00:00', got %q", restored.TargetLocalDateTime.ValueString())
	}
}

func TestSoftwareUpdate_ToClientComponent(t *testing.T) {
	c := &SoftwareUpdateComponent{
		DeploymentTime:      types.StringValue("12:00"),
		EnforceAfterDays:    types.Int64Value(3),
		IgnoreMajorVersions: types.BoolValue(false),
	}

	comp, err := c.ToClientComponent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if comp.Identifier != "com.jamf.ddm.sw-updates" {
		t.Errorf("expected identifier 'com.jamf.ddm.sw-updates', got %q", comp.Identifier)
	}
	if comp.Configuration == nil {
		t.Fatal("expected non-nil configuration")
	}
}
