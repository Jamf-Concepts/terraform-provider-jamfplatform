// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance

package client_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/testhelpers"
)

// createTestBlueprint is a helper that creates a blueprint with the given steps, registers cleanup,
// and returns the full blueprint detail for assertions.
func createTestBlueprint(t *testing.T, c *client.Client, name string, groupID string, steps []client.BlueprintStepV1) *client.BlueprintDetailV1 {
	t.Helper()
	ctx := context.Background()

	createReq := &client.BlueprintCreateRequestV1{
		Name:        name,
		Description: "Acceptance test — safe to delete",
		Scope:       client.BlueprintCreateScopeV1{DeviceGroups: []string{groupID}},
		Steps:       steps,
	}

	createResp, err := c.CreateBlueprintV1(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateBlueprintV1 failed for %q: %v", name, err)
	}

	t.Cleanup(func() {
		_ = c.DeleteBlueprintV1(ctx, createResp.ID)
	})

	blueprint, err := c.GetBlueprintByIDV1(ctx, createResp.ID)
	if err != nil {
		t.Fatalf("GetBlueprintByIDV1 failed for %q: %v", name, err)
	}

	return blueprint
}

// makeStep wraps a single component into a BlueprintStepV1.
func makeStep(identifier string, config any) []client.BlueprintStepV1 {
	configJSON, _ := json.Marshal(config)
	return []client.BlueprintStepV1{
		{
			Name: "Step 1",
			Components: []client.BlueprintComponentV1{
				{
					Identifier:    identifier,
					Configuration: json.RawMessage(configJSON),
				},
			},
		},
	}
}

func TestAcceptance_Blueprint_EmptyBlueprint(t *testing.T) {
	groupID := testhelpers.RequireSmartGroupFixture(t)
	c := testhelpers.NewAcceptanceClient(t)
	suffix := testhelpers.RunSuffix()

	name := "tf-acc-empty-blueprint-" + suffix
	bp := createTestBlueprint(t, c, name, groupID, []client.BlueprintStepV1{})

	if bp.Name != name {
		t.Errorf("expected name %q, got %q", name, bp.Name)
	}
	if len(bp.Scope.DeviceGroups) != 1 || bp.Scope.DeviceGroups[0] != groupID {
		t.Errorf("expected scope with group %q, got %v", groupID, bp.Scope.DeviceGroups)
	}

	t.Logf("Created empty blueprint ID: %s", bp.ID)
}

func TestAcceptance_Blueprint_PasscodePolicy(t *testing.T) {
	groupID := testhelpers.RequireSmartGroupFixture(t)
	c := testhelpers.NewAcceptanceClient(t)
	suffix := testhelpers.RunSuffix()

	config := map[string]any{
		"RequirePasscode":              true,
		"MinimumLength":                8,
		"MaximumFailedAttempts":        5,
		"MaximumInactivityInMinutes":   10,
		"MaximumPasscodeAgeInDays":     90,
		"MinimumComplexCharacters":     1,
		"RequireAlphanumericPasscode":  true,
		"FailedAttemptsResetInMinutes": 30,
	}

	steps := makeStep("com.jamf.ddm.passcode-settings", config)
	name := "tf-acc-passcode-policy-" + suffix
	bp := createTestBlueprint(t, c, name, groupID, steps)

	if len(bp.Steps) == 0 || len(bp.Steps[0].Components) == 0 {
		t.Fatal("expected at least one step with one component")
	}

	comp := bp.Steps[0].Components[0]
	if comp.Identifier != "com.jamf.ddm.passcode-settings" {
		t.Errorf("expected identifier 'com.jamf.ddm.passcode-settings', got %q", comp.Identifier)
	}

	t.Logf("Created passcode policy blueprint ID: %s", bp.ID)
}

func TestAcceptance_Blueprint_MathSettings(t *testing.T) {
	groupID := testhelpers.RequireSmartGroupFixture(t)
	c := testhelpers.NewAcceptanceClient(t)
	suffix := testhelpers.RunSuffix()

	config := map[string]any{
		"Calculator": map[string]any{
			"BasicMode": map[string]any{
				"Included":      true,
				"AddSquareRoot": true,
			},
			"ScientificMode": map[string]any{
				"Included": true,
				"Enabled":  true,
			},
			"ProgrammerMode": map[string]any{
				"Included": true,
				"Enabled":  false,
			},
			"MathNotesMode": map[string]any{
				"Included": true,
				"Enabled":  true,
			},
			"InputModes": map[string]any{
				"Included":       true,
				"UnitConversion": true,
				"RPN":            false,
			},
		},
		"SystemBehavior": map[string]any{
			"Included":            true,
			"KeyboardSuggestions": true,
			"MathNotes":           true,
		},
	}

	steps := makeStep("com.jamf.ddm.math-settings", config)
	name := "tf-acc-math-settings-" + suffix
	bp := createTestBlueprint(t, c, name, groupID, steps)

	if len(bp.Steps) == 0 || len(bp.Steps[0].Components) == 0 {
		t.Fatal("expected at least one step with one component")
	}
	if bp.Steps[0].Components[0].Identifier != "com.jamf.ddm.math-settings" {
		t.Errorf("expected identifier 'com.jamf.ddm.math-settings', got %q", bp.Steps[0].Components[0].Identifier)
	}

	t.Logf("Created math settings blueprint ID: %s", bp.ID)
}

func TestAcceptance_Blueprint_AudioAccessorySettings(t *testing.T) {
	groupID := testhelpers.RequireSmartGroupFixture(t)
	c := testhelpers.NewAcceptanceClient(t)
	suffix := testhelpers.RunSuffix()

	config := map[string]any{
		"TemporaryPairing": map[string]any{
			"Included": true,
			"Disabled": true,
			"Configuration": map[string]any{
				"UnpairingTime": map[string]any{
					"Policy": "Hour",
					"Hour":   14,
				},
			},
		},
	}

	steps := makeStep("com.jamf.ddm.audio-accessory-settings", config)
	name := "tf-acc-audio-accessory-" + suffix
	bp := createTestBlueprint(t, c, name, groupID, steps)

	if len(bp.Steps) == 0 || len(bp.Steps[0].Components) == 0 {
		t.Fatal("expected at least one step with one component")
	}
	if bp.Steps[0].Components[0].Identifier != "com.jamf.ddm.audio-accessory-settings" {
		t.Errorf("expected identifier 'com.jamf.ddm.audio-accessory-settings', got %q", bp.Steps[0].Components[0].Identifier)
	}

	t.Logf("Created audio accessory settings blueprint ID: %s", bp.ID)
}

func TestAcceptance_Blueprint_DiskManagement(t *testing.T) {
	groupID := testhelpers.RequireSmartGroupFixture(t)
	c := testhelpers.NewAcceptanceClient(t)
	suffix := testhelpers.RunSuffix()

	config := map[string]any{
		"version": 2,
		"Restrictions": map[string]any{
			"ExternalStorage": map[string]any{
				"Included": true,
				"Value":    "ReadOnly",
			},
			"NetworkStorage": map[string]any{
				"Included": true,
				"Value":    "Disallowed",
			},
		},
	}

	steps := makeStep("com.jamf.ddm.disk-management", config)
	name := "tf-acc-disk-management-" + suffix
	bp := createTestBlueprint(t, c, name, groupID, steps)

	if len(bp.Steps) == 0 || len(bp.Steps[0].Components) == 0 {
		t.Fatal("expected at least one step with one component")
	}
	if bp.Steps[0].Components[0].Identifier != "com.jamf.ddm.disk-management" {
		t.Errorf("expected identifier 'com.jamf.ddm.disk-management', got %q", bp.Steps[0].Components[0].Identifier)
	}

	t.Logf("Created disk management blueprint ID: %s", bp.ID)
}

func TestAcceptance_Blueprint_SafariSettings(t *testing.T) {
	groupID := testhelpers.RequireSmartGroupFixture(t)
	c := testhelpers.NewAcceptanceClient(t)
	suffix := testhelpers.RunSuffix()

	config := map[string]any{
		"AcceptCookies": map[string]any{
			"Included": true,
			"Value":    "VisitedWebsites",
		},
		"AllowPrivateBrowsing": map[string]any{
			"Included": true,
			"Value":    false,
		},
		"AllowJavaScript": map[string]any{
			"Included": true,
			"Value":    true,
		},
		"AllowPopups": map[string]any{
			"Included": true,
			"Value":    false,
		},
		"AllowHistoryClearing": map[string]any{
			"Included": true,
			"Value":    false,
		},
	}

	steps := makeStep("com.jamf.ddm.safari-settings", config)
	name := "tf-acc-safari-settings-" + suffix
	bp := createTestBlueprint(t, c, name, groupID, steps)

	if len(bp.Steps) == 0 || len(bp.Steps[0].Components) == 0 {
		t.Fatal("expected at least one step with one component")
	}
	if bp.Steps[0].Components[0].Identifier != "com.jamf.ddm.safari-settings" {
		t.Errorf("expected identifier 'com.jamf.ddm.safari-settings', got %q", bp.Steps[0].Components[0].Identifier)
	}

	t.Logf("Created safari settings blueprint ID: %s", bp.ID)
}

func TestAcceptance_Blueprint_SoftwareUpdateSettings(t *testing.T) {
	groupID := testhelpers.RequireSmartGroupFixture(t)
	c := testhelpers.NewAcceptanceClient(t)
	suffix := testhelpers.RunSuffix()

	config := map[string]any{
		"AllowStandardUserOSUpdates": map[string]any{
			"Included": true,
			"Enabled":  true,
		},
		"AutomaticActions": map[string]any{
			"Download": map[string]any{
				"Included": true,
				"Value":    "AlwaysOn",
			},
			"InstallOSUpdates": map[string]any{
				"Included": true,
				"Value":    "AlwaysOn",
			},
			"InstallSecurityUpdate": map[string]any{
				"Included": true,
				"Value":    "AlwaysOn",
			},
		},
		"Deferrals": map[string]any{
			"CombinedPeriodInDays": map[string]any{
				"Included": true,
				"Value":    "7",
			},
			"MajorPeriodInDays": map[string]any{
				"Included": true,
				"Value":    "30",
			},
			"MinorPeriodInDays": map[string]any{
				"Included": true,
				"Value":    "14",
			},
			"SystemPeriodInDays": map[string]any{
				"Included": true,
				"Value":    "3",
			},
		},
		"Notifications": map[string]any{
			"Included": true,
			"Enabled":  true,
		},
		"RapidSecurityResponse": map[string]any{
			"Enable": map[string]any{
				"Included": true,
				"Enabled":  true,
			},
			"EnableRollback": map[string]any{
				"Included": true,
				"Enabled":  false,
			},
		},
		"RecommendedCadence": map[string]any{
			"Included": true,
			"Value":    "Newest",
		},
	}

	steps := makeStep("com.jamf.ddm.software-update-settings", config)
	name := "tf-acc-sw-update-settings-" + suffix
	bp := createTestBlueprint(t, c, name, groupID, steps)

	if len(bp.Steps) == 0 || len(bp.Steps[0].Components) == 0 {
		t.Fatal("expected at least one step with one component")
	}
	if bp.Steps[0].Components[0].Identifier != "com.jamf.ddm.software-update-settings" {
		t.Errorf("expected identifier 'com.jamf.ddm.software-update-settings', got %q", bp.Steps[0].Components[0].Identifier)
	}

	t.Logf("Created software update settings blueprint ID: %s", bp.ID)
}

func TestAcceptance_Blueprint_SoftwareUpdate_Automatic(t *testing.T) {
	groupID := testhelpers.RequireSmartGroupFixture(t)
	c := testhelpers.NewAcceptanceClient(t)
	suffix := testhelpers.RunSuffix()

	config := map[string]any{
		"enforcementType":  "AUTOMATIC",
		"strategy":         "LATEST",
		"deploymentTime":   "02:00",
		"enforceAfterDays": 7,
		"detailsURL": map[string]any{
			"Included": false,
			"Value":    "",
		},
	}

	steps := makeStep("com.jamf.ddm.sw-updates", config)
	name := "tf-acc-sw-update-automatic-" + suffix
	bp := createTestBlueprint(t, c, name, groupID, steps)

	if len(bp.Steps) == 0 || len(bp.Steps[0].Components) == 0 {
		t.Fatal("expected at least one step with one component")
	}
	if bp.Steps[0].Components[0].Identifier != "com.jamf.ddm.sw-updates" {
		t.Errorf("expected identifier 'com.jamf.ddm.sw-updates', got %q", bp.Steps[0].Components[0].Identifier)
	}

	t.Logf("Created automatic software update blueprint ID: %s", bp.ID)
}

func TestAcceptance_Blueprint_LegacyPayloads(t *testing.T) {
	groupID := testhelpers.RequireSmartGroupFixture(t)
	c := testhelpers.NewAcceptanceClient(t)
	suffix := testhelpers.RunSuffix()

	config := map[string]any{
		"payloadContent": []map[string]any{
			{
				"allowSafariHistoryClearing": false,
				"allowSafariPrivateBrowsing": false,
				"payloadType":                "com.apple.applicationaccess",
				"payloadIdentifier":          "tf-acc-test-payload-001",
			},
		},
		"payloadDisplayName": "tf-acc-legacy-payloads-" + suffix,
	}

	steps := makeStep("com.jamf.ddm-configuration-profile", config)
	name := "tf-acc-legacy-payloads-" + suffix
	bp := createTestBlueprint(t, c, name, groupID, steps)

	if len(bp.Steps) == 0 || len(bp.Steps[0].Components) == 0 {
		t.Fatal("expected at least one step with one component")
	}
	if bp.Steps[0].Components[0].Identifier != "com.jamf.ddm-configuration-profile" {
		t.Errorf("expected identifier 'com.jamf.ddm-configuration-profile', got %q", bp.Steps[0].Components[0].Identifier)
	}

	t.Logf("Created legacy payloads blueprint ID: %s", bp.ID)
}

func TestAcceptance_Blueprint_CustomDeclarations(t *testing.T) {
	groupID := testhelpers.RequireSmartGroupFixture(t)
	c := testhelpers.NewAcceptanceClient(t)
	suffix := testhelpers.RunSuffix()

	config := map[string]any{
		"declarations": []map[string]any{
			{
				"channelType": "SYSTEM",
				"kind":        "CONFIGURATION",
				"type":        "com.apple.configuration.softwareupdate.settings",
				"payload": map[string]any{
					"Beta": map[string]any{
						"ProgramEnrollment": "AlwaysOff",
					},
				},
				"payloadKey": 1,
			},
		},
	}

	steps := makeStep("com.jamf.ddm.custom-declarations", config)
	name := "tf-acc-custom-declarations-" + suffix
	bp := createTestBlueprint(t, c, name, groupID, steps)

	if len(bp.Steps) == 0 || len(bp.Steps[0].Components) == 0 {
		t.Fatal("expected at least one step with one component")
	}
	if bp.Steps[0].Components[0].Identifier != "com.jamf.ddm.custom-declarations" {
		t.Errorf("expected identifier 'com.jamf.ddm.custom-declarations', got %q", bp.Steps[0].Components[0].Identifier)
	}

	t.Logf("Created custom declarations blueprint ID: %s", bp.ID)
}

func TestAcceptance_Blueprint_SafariBookmarks(t *testing.T) {
	groupID := testhelpers.RequireSmartGroupFixture(t)
	c := testhelpers.NewAcceptanceClient(t)
	suffix := testhelpers.RunSuffix()

	config := map[string]any{
		"ManagedBookmarks": []map[string]any{
			{
				"GroupIdentifier": "tf-acc-test-group-1",
				"Title":           "Test Bookmarks",
				"Bookmarks": []map[string]any{
					{
						"Type":  "BOOKMARK",
						"Title": "Jamf",
						"URL":   "https://www.jamf.com",
					},
				},
			},
		},
	}

	steps := makeStep("com.jamf.ddm.safari-bookmarks", config)
	name := "tf-acc-safari-bookmarks-" + suffix
	bp := createTestBlueprint(t, c, name, groupID, steps)

	if len(bp.Steps) == 0 || len(bp.Steps[0].Components) == 0 {
		t.Fatal("expected at least one step with one component")
	}
	if bp.Steps[0].Components[0].Identifier != "com.jamf.ddm.safari-bookmarks" {
		t.Errorf("expected identifier 'com.jamf.ddm.safari-bookmarks', got %q", bp.Steps[0].Components[0].Identifier)
	}

	t.Logf("Created safari bookmarks blueprint ID: %s", bp.ID)
}

func TestAcceptance_Blueprint_SafariExtensions(t *testing.T) {
	groupID := testhelpers.RequireSmartGroupFixture(t)
	c := testhelpers.NewAcceptanceClient(t)
	suffix := testhelpers.RunSuffix()

	config := map[string]any{
		"ManagedExtensions": map[string]any{
			"com.example.test.extension (ABC1234567)": map[string]any{
				"State":           "Allowed",
				"PrivateBrowsing": "AlwaysOff",
			},
		},
	}

	steps := makeStep("com.jamf.ddm.safari-extensions", config)
	name := "tf-acc-safari-extensions-" + suffix
	bp := createTestBlueprint(t, c, name, groupID, steps)

	if len(bp.Steps) == 0 || len(bp.Steps[0].Components) == 0 {
		t.Fatal("expected at least one step with one component")
	}
	if bp.Steps[0].Components[0].Identifier != "com.jamf.ddm.safari-extensions" {
		t.Errorf("expected identifier 'com.jamf.ddm.safari-extensions', got %q", bp.Steps[0].Components[0].Identifier)
	}

	t.Logf("Created safari extensions blueprint ID: %s", bp.ID)
}

func TestAcceptance_Blueprint_MultipleComponents(t *testing.T) {
	groupID := testhelpers.RequireSmartGroupFixture(t)
	c := testhelpers.NewAcceptanceClient(t)
	ctx := context.Background()
	suffix := testhelpers.RunSuffix()

	passcodeConfig, _ := json.Marshal(map[string]any{
		"RequirePasscode": true,
		"MinimumLength":   6,
	})
	mathConfig, _ := json.Marshal(map[string]any{
		"Calculator": map[string]any{
			"BasicMode": map[string]any{
				"Included":      true,
				"AddSquareRoot": false,
			},
			"ScientificMode": map[string]any{
				"Included": true,
				"Enabled":  true,
			},
			"ProgrammerMode": map[string]any{
				"Included": true,
				"Enabled":  true,
			},
			"MathNotesMode": map[string]any{
				"Included": true,
				"Enabled":  true,
			},
			"InputModes": map[string]any{
				"Included":       false,
				"UnitConversion": true,
				"RPN":            true,
			},
		},
		"SystemBehavior": map[string]any{
			"Included":            false,
			"KeyboardSuggestions": true,
			"MathNotes":           true,
		},
	})

	steps := []client.BlueprintStepV1{
		{
			Name: "Security",
			Components: []client.BlueprintComponentV1{
				{
					Identifier:    "com.jamf.ddm.passcode-settings",
					Configuration: json.RawMessage(passcodeConfig),
				},
			},
		},
		{
			Name: "Education",
			Components: []client.BlueprintComponentV1{
				{
					Identifier:    "com.jamf.ddm.math-settings",
					Configuration: json.RawMessage(mathConfig),
				},
			},
		},
	}

	name := "tf-acc-multi-component-" + suffix
	createReq := &client.BlueprintCreateRequestV1{
		Name:        name,
		Description: "Acceptance test — safe to delete",
		Scope:       client.BlueprintCreateScopeV1{DeviceGroups: []string{groupID}},
		Steps:       steps,
	}

	createResp, err := c.CreateBlueprintV1(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateBlueprintV1 failed: %v", err)
	}

	t.Cleanup(func() {
		_ = c.DeleteBlueprintV1(ctx, createResp.ID)
	})

	bp, err := c.GetBlueprintByIDV1(ctx, createResp.ID)
	if err != nil {
		t.Fatalf("GetBlueprintByIDV1 failed: %v", err)
	}

	if len(bp.Steps) < 2 {
		t.Fatalf("expected at least 2 steps, got %d", len(bp.Steps))
	}

	componentIDs := make(map[string]bool)
	for _, step := range bp.Steps {
		for _, comp := range step.Components {
			componentIDs[comp.Identifier] = true
		}
	}

	if !componentIDs["com.jamf.ddm.passcode-settings"] {
		t.Error("expected passcode-settings component in blueprint")
	}
	if !componentIDs["com.jamf.ddm.math-settings"] {
		t.Error("expected math-settings component in blueprint")
	}

	t.Logf("Created multi-component blueprint ID: %s with %d steps", bp.ID, len(bp.Steps))
}

func TestAcceptance_Blueprint_UpdateAndRead(t *testing.T) {
	groupID := testhelpers.RequireSmartGroupFixture(t)
	c := testhelpers.NewAcceptanceClient(t)
	ctx := context.Background()
	suffix := testhelpers.RunSuffix()

	steps := makeStep("com.jamf.ddm.passcode-settings", map[string]any{
		"RequirePasscode": true,
		"MinimumLength":   6,
	})
	name := "tf-acc-update-test-" + suffix
	bp := createTestBlueprint(t, c, name, groupID, steps)

	renamedName := "tf-acc-update-test-renamed-" + suffix
	updateReq := &client.BlueprintUpdateRequestV1{
		Name:        renamedName,
		Description: "Updated description",
		Scope:       client.BlueprintUpdateScopeV1{DeviceGroups: []string{groupID}},
		Steps:       steps,
	}

	err := c.UpdateBlueprintV1(ctx, bp.ID, updateReq)
	if err != nil {
		t.Fatalf("UpdateBlueprintV1 failed: %v", err)
	}

	updated, err := c.GetBlueprintByIDV1(ctx, bp.ID)
	if err != nil {
		t.Fatalf("GetBlueprintByIDV1 after update failed: %v", err)
	}

	if updated.Name != renamedName {
		t.Errorf("expected name %q, got %q", renamedName, updated.Name)
	}
	if updated.Description != "Updated description" {
		t.Errorf("expected description 'Updated description', got %q", updated.Description)
	}

	t.Logf("Updated blueprint ID: %s", bp.ID)
}

func TestAcceptance_Blueprint_GetByName(t *testing.T) {
	groupID := testhelpers.RequireSmartGroupFixture(t)
	c := testhelpers.NewAcceptanceClient(t)
	suffix := testhelpers.RunSuffix()

	steps := makeStep("com.jamf.ddm.passcode-settings", map[string]any{
		"RequirePasscode": true,
		"MinimumLength":   6,
	})
	name := "tf-acc-find-by-name-" + suffix
	_ = createTestBlueprint(t, c, name, groupID, steps)

	found, err := c.GetBlueprintByNameV1(context.Background(), name)
	if err != nil {
		t.Fatalf("GetBlueprintByNameV1 failed: %v", err)
	}

	if found.Name != name {
		t.Errorf("expected name %q, got %q", name, found.Name)
	}

	t.Logf("Found blueprint by name: ID %s", found.ID)
}
