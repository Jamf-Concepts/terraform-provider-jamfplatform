// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDiskManagementPolicy_GetIdentifier(t *testing.T) {
	c := &DiskManagementPolicyComponent{}
	if c.GetIdentifier() != "com.jamf.ddm.disk-management" {
		t.Errorf("expected 'com.jamf.ddm.disk-management', got %q", c.GetIdentifier())
	}
}

func TestDiskManagementPolicy_ToRawConfiguration_AllFields(t *testing.T) {
	c := &DiskManagementPolicyComponent{
		ExternalStorage: types.StringValue("Disallowed"),
		NetworkStorage:  types.StringValue("ReadOnly"),
	}

	raw, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var config map[string]any
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if config["version"] != float64(2) {
		t.Errorf("expected version 2, got %v", config["version"])
	}

	restrictions, ok := config["Restrictions"].(map[string]any)
	if !ok {
		t.Fatal("expected Restrictions to be a map")
	}

	extStorage, ok := restrictions["ExternalStorage"].(map[string]any)
	if !ok {
		t.Fatal("expected ExternalStorage to be a map")
	}
	if extStorage["Value"] != "Disallowed" {
		t.Errorf("expected ExternalStorage Value 'Disallowed', got %v", extStorage["Value"])
	}
	if extStorage["Included"] != true {
		t.Errorf("expected ExternalStorage Included true, got %v", extStorage["Included"])
	}

	netStorage, ok := restrictions["NetworkStorage"].(map[string]any)
	if !ok {
		t.Fatal("expected NetworkStorage to be a map")
	}
	if netStorage["Value"] != "ReadOnly" {
		t.Errorf("expected NetworkStorage Value 'ReadOnly', got %v", netStorage["Value"])
	}
	if netStorage["Included"] != true {
		t.Errorf("expected NetworkStorage Included true, got %v", netStorage["Included"])
	}
}

func TestDiskManagementPolicy_ToRawConfiguration_NullFieldsDefaults(t *testing.T) {
	c := &DiskManagementPolicyComponent{
		ExternalStorage: types.StringNull(),
		NetworkStorage:  types.StringNull(),
	}

	raw, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var config map[string]any
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	restrictions := config["Restrictions"].(map[string]any)

	extStorage := restrictions["ExternalStorage"].(map[string]any)
	if extStorage["Value"] != "Allowed" {
		t.Errorf("expected default ExternalStorage Value 'Allowed', got %v", extStorage["Value"])
	}
	if extStorage["Included"] != false {
		t.Errorf("expected ExternalStorage Included false for null, got %v", extStorage["Included"])
	}

	netStorage := restrictions["NetworkStorage"].(map[string]any)
	if netStorage["Value"] != "Allowed" {
		t.Errorf("expected default NetworkStorage Value 'Allowed', got %v", netStorage["Value"])
	}
	if netStorage["Included"] != false {
		t.Errorf("expected NetworkStorage Included false for null, got %v", netStorage["Included"])
	}
}

func TestDiskManagementPolicy_FromRawConfiguration(t *testing.T) {
	inputMap := map[string]any{
		"Restrictions": map[string]any{
			"ExternalStorage": map[string]any{
				"Value":    "ReadOnly",
				"Included": true,
			},
			"NetworkStorage": map[string]any{
				"Value":    "Disallowed",
				"Included": true,
			},
		},
	}
	raw, _ := json.Marshal(inputMap)

	c := &DiskManagementPolicyComponent{}
	if err := c.FromRawConfiguration(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.ExternalStorage.ValueString() != "ReadOnly" {
		t.Errorf("expected ExternalStorage 'ReadOnly', got %q", c.ExternalStorage.ValueString())
	}
	if c.NetworkStorage.ValueString() != "Disallowed" {
		t.Errorf("expected NetworkStorage 'Disallowed', got %q", c.NetworkStorage.ValueString())
	}
}

func TestDiskManagementPolicy_FromRawConfiguration_NotIncluded(t *testing.T) {
	inputMap := map[string]any{
		"Restrictions": map[string]any{
			"ExternalStorage": map[string]any{
				"Value":    "Allowed",
				"Included": false,
			},
			"NetworkStorage": map[string]any{
				"Value":    "Allowed",
				"Included": false,
			},
		},
	}
	raw, _ := json.Marshal(inputMap)

	c := &DiskManagementPolicyComponent{}
	if err := c.FromRawConfiguration(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !c.ExternalStorage.IsNull() {
		t.Error("expected null ExternalStorage when not included")
	}
	if !c.NetworkStorage.IsNull() {
		t.Error("expected null NetworkStorage when not included")
	}
}

func TestDiskManagementPolicy_FromRawConfiguration_Empty(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{})

	c := &DiskManagementPolicyComponent{}
	if err := c.FromRawConfiguration(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !c.ExternalStorage.IsNull() {
		t.Error("expected null ExternalStorage for empty config")
	}
	if !c.NetworkStorage.IsNull() {
		t.Error("expected null NetworkStorage for empty config")
	}
}

func TestDiskManagementPolicy_Roundtrip(t *testing.T) {
	original := &DiskManagementPolicyComponent{
		ExternalStorage: types.StringValue("Disallowed"),
		NetworkStorage:  types.StringValue("ReadOnly"),
	}

	raw, err := original.ToRawConfiguration()
	if err != nil {
		t.Fatalf("ToRawConfiguration error: %v", err)
	}

	restored := &DiskManagementPolicyComponent{}
	if err := restored.FromRawConfiguration(raw); err != nil {
		t.Fatalf("FromRawConfiguration error: %v", err)
	}

	if restored.ExternalStorage.ValueString() != "Disallowed" {
		t.Errorf("roundtrip: expected ExternalStorage 'Disallowed', got %q", restored.ExternalStorage.ValueString())
	}
	if restored.NetworkStorage.ValueString() != "ReadOnly" {
		t.Errorf("roundtrip: expected NetworkStorage 'ReadOnly', got %q", restored.NetworkStorage.ValueString())
	}
}

func TestDiskManagementPolicy_ToClientComponent(t *testing.T) {
	c := &DiskManagementPolicyComponent{
		ExternalStorage: types.StringValue("Allowed"),
		NetworkStorage:  types.StringValue("Allowed"),
	}

	comp, err := c.ToClientComponent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if comp.Identifier != "com.jamf.ddm.disk-management" {
		t.Errorf("expected identifier 'com.jamf.ddm.disk-management', got %q", comp.Identifier)
	}
	if comp.Configuration == nil {
		t.Fatal("expected non-nil configuration")
	}
}
