// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSafariExtensions_GetIdentifier(t *testing.T) {
	c := &SafariExtensionsComponent{}
	if c.GetIdentifier() != "com.jamf.ddm.safari-extensions" {
		t.Errorf("expected 'com.jamf.ddm.safari-extensions', got %q", c.GetIdentifier())
	}
}

func TestSafariExtensions_ToRawConfiguration_WithExtensions(t *testing.T) {
	c := &SafariExtensionsComponent{
		ManagedExtensions: []ManagedExtensionModel{
			{
				ExtensionID:     types.StringValue("com.example.ext1"),
				State:           types.StringValue("AlwaysOn"),
				PrivateBrowsing: types.StringValue("Allowed"),
				AllowedDomains: []ManagedExtensionDomainModel{
					{Domain: types.StringValue("example.com")},
				},
				DeniedDomains: []ManagedExtensionDomainModel{
					{Domain: types.StringValue("blocked.com")},
				},
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

	managedExts, ok := config["ManagedExtensions"].(map[string]any)
	if !ok {
		t.Fatal("expected ManagedExtensions to be a map")
	}

	extConfig, ok := managedExts["com.example.ext1"].(map[string]any)
	if !ok {
		t.Fatal("expected extension config to be a map")
	}
	if extConfig["State"] != "AlwaysOn" {
		t.Errorf("expected State 'AlwaysOn', got %v", extConfig["State"])
	}
	if extConfig["PrivateBrowsing"] != "Allowed" {
		t.Errorf("expected PrivateBrowsing 'Allowed', got %v", extConfig["PrivateBrowsing"])
	}

	allowedDomains, ok := extConfig["AllowedDomains"].([]any)
	if !ok {
		t.Fatal("expected AllowedDomains to be a slice")
	}
	if len(allowedDomains) != 1 {
		t.Fatalf("expected 1 allowed domain, got %d", len(allowedDomains))
	}
	ad := allowedDomains[0].(map[string]any)
	if ad["Domain"] != "example.com" {
		t.Errorf("expected allowed domain 'example.com', got %v", ad["Domain"])
	}

	deniedDomains, ok := extConfig["DeniedDomains"].([]any)
	if !ok {
		t.Fatal("expected DeniedDomains to be a slice")
	}
	if len(deniedDomains) != 1 {
		t.Fatalf("expected 1 denied domain, got %d", len(deniedDomains))
	}
	dd := deniedDomains[0].(map[string]any)
	if dd["Domain"] != "blocked.com" {
		t.Errorf("expected denied domain 'blocked.com', got %v", dd["Domain"])
	}
}

func TestSafariExtensions_ToRawConfiguration_Empty(t *testing.T) {
	c := &SafariExtensionsComponent{}

	rawCfg, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var config map[string]any
	if err := json.Unmarshal(rawCfg, &config); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	exts, ok := config["ManagedExtensions"].(map[string]any)
	if !ok || len(exts) != 0 {
		t.Error("expected empty ManagedExtensions map for empty component")
	}
}

func TestSafariExtensions_FromRawConfiguration(t *testing.T) {
	rawMap := map[string]any{
		"ManagedExtensions": map[string]any{
			"com.example.ext1": map[string]any{
				"State":           "AlwaysOff",
				"PrivateBrowsing": "AlwaysOn",
				"AllowedDomains": []any{
					map[string]any{"Domain": "safe.com"},
				},
				"DeniedDomains": []any{
					map[string]any{"Domain": "bad.com"},
				},
			},
		},
	}
	raw, _ := json.Marshal(rawMap)

	c := &SafariExtensionsComponent{}
	if err := c.FromRawConfiguration(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(c.ManagedExtensions) != 1 {
		t.Fatalf("expected 1 extension, got %d", len(c.ManagedExtensions))
	}

	ext := c.ManagedExtensions[0]
	if ext.ExtensionID.ValueString() != "com.example.ext1" {
		t.Errorf("expected ExtensionID 'com.example.ext1', got %q", ext.ExtensionID.ValueString())
	}
	if ext.State.ValueString() != "AlwaysOff" {
		t.Errorf("expected State 'AlwaysOff', got %q", ext.State.ValueString())
	}
	if ext.PrivateBrowsing.ValueString() != "AlwaysOn" {
		t.Errorf("expected PrivateBrowsing 'AlwaysOn', got %q", ext.PrivateBrowsing.ValueString())
	}
	if len(ext.AllowedDomains) != 1 {
		t.Fatalf("expected 1 allowed domain, got %d", len(ext.AllowedDomains))
	}
	if ext.AllowedDomains[0].Domain.ValueString() != "safe.com" {
		t.Errorf("expected allowed domain 'safe.com', got %q", ext.AllowedDomains[0].Domain.ValueString())
	}
	if len(ext.DeniedDomains) != 1 {
		t.Fatalf("expected 1 denied domain, got %d", len(ext.DeniedDomains))
	}
	if ext.DeniedDomains[0].Domain.ValueString() != "bad.com" {
		t.Errorf("expected denied domain 'bad.com', got %q", ext.DeniedDomains[0].Domain.ValueString())
	}
}

func TestSafariExtensions_FromRawConfiguration_Empty(t *testing.T) {
	c := &SafariExtensionsComponent{}
	if err := c.FromRawConfiguration(json.RawMessage("{}")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(c.ManagedExtensions) != 0 {
		t.Errorf("expected 0 extensions, got %d", len(c.ManagedExtensions))
	}
}

func TestSafariExtensions_Roundtrip(t *testing.T) {
	original := &SafariExtensionsComponent{
		ManagedExtensions: []ManagedExtensionModel{
			{
				ExtensionID:     types.StringValue("com.test.safari-ext"),
				State:           types.StringValue("Allowed"),
				PrivateBrowsing: types.StringValue("AlwaysOff"),
				AllowedDomains: []ManagedExtensionDomainModel{
					{Domain: types.StringValue("good.com")},
				},
			},
		},
	}

	rawCfg, err := original.ToRawConfiguration()
	if err != nil {
		t.Fatalf("ToRawConfiguration error: %v", err)
	}

	restored := &SafariExtensionsComponent{}
	if err := restored.FromRawConfiguration(rawCfg); err != nil {
		t.Fatalf("FromRawConfiguration error: %v", err)
	}

	if len(restored.ManagedExtensions) != 1 {
		t.Fatalf("roundtrip: expected 1 extension, got %d", len(restored.ManagedExtensions))
	}

	ext := restored.ManagedExtensions[0]
	if ext.ExtensionID.ValueString() != "com.test.safari-ext" {
		t.Errorf("roundtrip: expected ExtensionID 'com.test.safari-ext', got %q", ext.ExtensionID.ValueString())
	}
	if ext.State.ValueString() != "Allowed" {
		t.Errorf("roundtrip: expected State 'Allowed', got %q", ext.State.ValueString())
	}
	if len(ext.AllowedDomains) != 1 {
		t.Fatalf("roundtrip: expected 1 allowed domain, got %d", len(ext.AllowedDomains))
	}
	if ext.AllowedDomains[0].Domain.ValueString() != "good.com" {
		t.Errorf("roundtrip: expected allowed domain 'good.com', got %q", ext.AllowedDomains[0].Domain.ValueString())
	}
}

func TestSafariExtensions_ToClientComponent(t *testing.T) {
	c := &SafariExtensionsComponent{
		ManagedExtensions: []ManagedExtensionModel{
			{
				ExtensionID: types.StringValue("com.example.ext"),
				State:       types.StringValue("Allowed"),
			},
		},
	}

	comp, err := c.ToClientComponent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if comp.Identifier != "com.jamf.ddm.safari-extensions" {
		t.Errorf("expected identifier 'com.jamf.ddm.safari-extensions', got %q", comp.Identifier)
	}
	if comp.Configuration == nil {
		t.Fatal("expected non-nil configuration")
	}
}
