// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSafariSettings_GetIdentifier(t *testing.T) {
	c := &SafariSettingsComponent{}
	if c.GetIdentifier() != "com.jamf.ddm.safari-settings" {
		t.Errorf("expected 'com.jamf.ddm.safari-settings', got %q", c.GetIdentifier())
	}
}

func TestSafariSettings_ToRawConfiguration_BoolFields(t *testing.T) {
	c := &SafariSettingsComponent{
		AllowDisablingFraudWarning: types.BoolValue(true),
		AllowHistoryClearing:       types.BoolValue(false),
		AllowJavaScript:            types.BoolValue(true),
		AllowPrivateBrowsing:       types.BoolValue(false),
		AllowPopups:                types.BoolValue(true),
		AllowSummary:               types.BoolValue(false),
	}

	config, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fraud, ok := config["AllowDisablingFraudWarning"].(map[string]any)
	if !ok {
		t.Fatal("expected AllowDisablingFraudWarning to be a map")
	}
	if fraud["Value"] != true {
		t.Errorf("expected AllowDisablingFraudWarning Value true, got %v", fraud["Value"])
	}
	if fraud["Included"] != true {
		t.Errorf("expected AllowDisablingFraudWarning Included true, got %v", fraud["Included"])
	}

	history, ok := config["AllowHistoryClearing"].(map[string]any)
	if !ok {
		t.Fatal("expected AllowHistoryClearing to be a map")
	}
	if history["Value"] != false {
		t.Errorf("expected AllowHistoryClearing Value false, got %v", history["Value"])
	}
}

func TestSafariSettings_ToRawConfiguration_AcceptCookies(t *testing.T) {
	c := &SafariSettingsComponent{
		AcceptCookies: types.StringValue("Never"),
	}

	config, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cookies, ok := config["AcceptCookies"].(map[string]any)
	if !ok {
		t.Fatal("expected AcceptCookies to be a map")
	}
	if cookies["Value"] != "Never" {
		t.Errorf("expected AcceptCookies Value 'Never', got %v", cookies["Value"])
	}
	if cookies["Included"] != true {
		t.Errorf("expected AcceptCookies Included true, got %v", cookies["Included"])
	}
}

func TestSafariSettings_ToRawConfiguration_NewTabStartPage(t *testing.T) {
	c := &SafariSettingsComponent{
		NewTabStartPageType:        types.StringValue("Home"),
		NewTabStartPageHomepageURL: types.StringValue("https://example.com"),
	}

	config, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newTab, ok := config["NewTabStartPage"].(map[string]any)
	if !ok {
		t.Fatal("expected NewTabStartPage to be a map")
	}
	if newTab["Included"] != true {
		t.Errorf("expected NewTabStartPage Included true, got %v", newTab["Included"])
	}
	if newTab["PageType"] != "Home" {
		t.Errorf("expected PageType 'Home', got %v", newTab["PageType"])
	}
	if newTab["HomepageURL"] != "https://example.com" {
		t.Errorf("expected HomepageURL 'https://example.com', got %v", newTab["HomepageURL"])
	}
}

func TestSafariSettings_ToRawConfiguration_NewTabStartPageExtension(t *testing.T) {
	c := &SafariSettingsComponent{
		NewTabStartPageType:        types.StringValue("Extension"),
		NewTabStartPageExtensionID: types.StringValue("com.example.ext (ABC123)"),
	}

	config, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newTab, ok := config["NewTabStartPage"].(map[string]any)
	if !ok {
		t.Fatal("expected NewTabStartPage to be a map")
	}
	if newTab["ExtensionIdentifier"] != "com.example.ext (ABC123)" {
		t.Errorf("expected ExtensionIdentifier 'com.example.ext (ABC123)', got %v", newTab["ExtensionIdentifier"])
	}
}

func TestSafariSettings_ToRawConfiguration_NullFields(t *testing.T) {
	c := &SafariSettingsComponent{}

	config, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, exists := config["AcceptCookies"]; exists {
		t.Error("expected AcceptCookies absent for null")
	}
	if _, exists := config["AllowJavaScript"]; exists {
		t.Error("expected AllowJavaScript absent for null")
	}
	if _, exists := config["NewTabStartPage"]; exists {
		t.Error("expected NewTabStartPage absent for null")
	}
}

func TestSafariSettings_FromRawConfiguration(t *testing.T) {
	raw := map[string]any{
		"AcceptCookies": map[string]any{
			"Value":    "Always",
			"Included": true,
		},
		"AllowDisablingFraudWarning": map[string]any{
			"Value":    true,
			"Included": true,
		},
		"AllowHistoryClearing": map[string]any{
			"Value":    false,
			"Included": true,
		},
		"AllowJavaScript": map[string]any{
			"Value":    true,
			"Included": true,
		},
		"AllowPrivateBrowsing": map[string]any{
			"Value":    false,
			"Included": true,
		},
		"AllowPopups": map[string]any{
			"Value":    true,
			"Included": true,
		},
		"AllowSummary": map[string]any{
			"Value":    false,
			"Included": true,
		},
		"NewTabStartPage": map[string]any{
			"Included":    true,
			"PageType":    "Home",
			"HomepageURL": "https://example.com",
		},
	}

	c := &SafariSettingsComponent{}
	if err := c.FromRawConfiguration(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.AcceptCookies.ValueString() != "Always" {
		t.Errorf("expected AcceptCookies 'Always', got %q", c.AcceptCookies.ValueString())
	}
	if c.AllowDisablingFraudWarning.ValueBool() != true {
		t.Errorf("expected AllowDisablingFraudWarning true, got %v", c.AllowDisablingFraudWarning.ValueBool())
	}
	if c.AllowHistoryClearing.ValueBool() != false {
		t.Errorf("expected AllowHistoryClearing false, got %v", c.AllowHistoryClearing.ValueBool())
	}
	if c.AllowJavaScript.ValueBool() != true {
		t.Errorf("expected AllowJavaScript true, got %v", c.AllowJavaScript.ValueBool())
	}
	if c.AllowPrivateBrowsing.ValueBool() != false {
		t.Errorf("expected AllowPrivateBrowsing false, got %v", c.AllowPrivateBrowsing.ValueBool())
	}
	if c.AllowPopups.ValueBool() != true {
		t.Errorf("expected AllowPopups true, got %v", c.AllowPopups.ValueBool())
	}
	if c.AllowSummary.ValueBool() != false {
		t.Errorf("expected AllowSummary false, got %v", c.AllowSummary.ValueBool())
	}
	if c.NewTabStartPageType.ValueString() != "Home" {
		t.Errorf("expected NewTabStartPageType 'Home', got %q", c.NewTabStartPageType.ValueString())
	}
	if c.NewTabStartPageHomepageURL.ValueString() != "https://example.com" {
		t.Errorf("expected NewTabStartPageHomepageURL 'https://example.com', got %q", c.NewTabStartPageHomepageURL.ValueString())
	}
}

func TestSafariSettings_FromRawConfiguration_Empty(t *testing.T) {
	c := &SafariSettingsComponent{}
	if err := c.FromRawConfiguration(map[string]any{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.AcceptCookies.ValueString() != "" && !c.AcceptCookies.IsNull() {
		t.Error("expected empty/null AcceptCookies for empty config")
	}
}

func TestSafariSettings_Roundtrip(t *testing.T) {
	original := &SafariSettingsComponent{
		AcceptCookies:              types.StringValue("CurrentWebsite"),
		AllowDisablingFraudWarning: types.BoolValue(false),
		AllowJavaScript:            types.BoolValue(true),
		AllowPopups:                types.BoolValue(false),
		NewTabStartPageType:        types.StringValue("Home"),
		NewTabStartPageHomepageURL: types.StringValue("https://jamf.com"),
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

	restored := &SafariSettingsComponent{}
	if err := restored.FromRawConfiguration(parsed); err != nil {
		t.Fatalf("FromRawConfiguration error: %v", err)
	}

	if restored.AcceptCookies.ValueString() != "CurrentWebsite" {
		t.Errorf("roundtrip: expected AcceptCookies 'CurrentWebsite', got %q", restored.AcceptCookies.ValueString())
	}
	if restored.AllowDisablingFraudWarning.ValueBool() != false {
		t.Errorf("roundtrip: expected AllowDisablingFraudWarning false, got %v", restored.AllowDisablingFraudWarning.ValueBool())
	}
	if restored.AllowJavaScript.ValueBool() != true {
		t.Errorf("roundtrip: expected AllowJavaScript true, got %v", restored.AllowJavaScript.ValueBool())
	}
	if restored.NewTabStartPageType.ValueString() != "Home" {
		t.Errorf("roundtrip: expected NewTabStartPageType 'Home', got %q", restored.NewTabStartPageType.ValueString())
	}
	if restored.NewTabStartPageHomepageURL.ValueString() != "https://jamf.com" {
		t.Errorf("roundtrip: expected NewTabStartPageHomepageURL 'https://jamf.com', got %q", restored.NewTabStartPageHomepageURL.ValueString())
	}
}

func TestSafariSettings_ToClientComponent(t *testing.T) {
	c := &SafariSettingsComponent{
		AllowJavaScript: types.BoolValue(true),
	}

	comp, err := c.ToClientComponent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if comp.Identifier != "com.jamf.ddm.safari-settings" {
		t.Errorf("expected identifier 'com.jamf.ddm.safari-settings', got %q", comp.Identifier)
	}
	if comp.Configuration == nil {
		t.Fatal("expected non-nil configuration")
	}
}
