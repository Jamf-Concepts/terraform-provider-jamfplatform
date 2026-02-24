// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSoftwareUpdateSettings_GetIdentifier(t *testing.T) {
	c := &SoftwareUpdateSettingsComponent{}
	if c.GetIdentifier() != "com.jamf.ddm.software-update-settings" {
		t.Errorf("expected 'com.jamf.ddm.software-update-settings', got %q", c.GetIdentifier())
	}
}

func TestSoftwareUpdateSettings_ToRawConfiguration_AllFields(t *testing.T) {
	c := &SoftwareUpdateSettingsComponent{
		AllowStandardUserOSUpdates:     types.BoolValue(true),
		AutomaticDownload:              types.StringValue("AlwaysOn"),
		AutomaticInstallOSUpdates:      types.StringValue("AlwaysOff"),
		AutomaticInstallSecurityUpdate: types.StringValue("Allowed"),
		BetaProgramEnrollment:          types.StringValue("AlwaysOn"),
		BetaRequireProgramToken:        types.StringValue("beta-token"),
		BetaRequireProgramDescription:  types.StringValue("Beta Description"),
		BetaOfferPrograms: []BetaProgramModel{
			{
				Token:       types.StringValue("offer-token-1"),
				Description: types.StringValue("Offer 1"),
			},
		},
		DeferralCombinedPeriod:               types.StringValue("30"),
		DeferralMajorPeriod:                  types.StringValue("60"),
		DeferralMinorPeriod:                  types.StringValue("14"),
		DeferralSystemPeriod:                 types.StringValue("7"),
		NotificationsEnabled:                 types.BoolValue(true),
		RapidSecurityResponseEnabled:         types.BoolValue(true),
		RapidSecurityResponseRollbackEnabled: types.BoolValue(false),
		RecommendedCadence:                   types.StringValue("Newest"),
	}

	config, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	allowUpdates, ok := config["AllowStandardUserOSUpdates"].(map[string]any)
	if !ok {
		t.Fatal("expected AllowStandardUserOSUpdates to be a map")
	}
	if allowUpdates["Enabled"] != true {
		t.Errorf("expected AllowStandardUserOSUpdates Enabled true, got %v", allowUpdates["Enabled"])
	}
	if allowUpdates["Included"] != true {
		t.Errorf("expected AllowStandardUserOSUpdates Included true, got %v", allowUpdates["Included"])
	}

	automaticActions, ok := config["AutomaticActions"].(map[string]any)
	if !ok {
		t.Fatal("expected AutomaticActions to be a map")
	}
	download := automaticActions["Download"].(map[string]any)
	if download["Value"] != "AlwaysOn" {
		t.Errorf("expected Download Value 'AlwaysOn', got %v", download["Value"])
	}
	if download["Included"] != true {
		t.Errorf("expected Download Included true, got %v", download["Included"])
	}

	beta, ok := config["Beta"].(map[string]any)
	if !ok {
		t.Fatal("expected Beta to be a map")
	}
	if beta["Included"] != true {
		t.Errorf("expected Beta Included true, got %v", beta["Included"])
	}
	betaValue := beta["Value"].(map[string]any)
	if betaValue["ProgramEnrollment"] != "AlwaysOn" {
		t.Errorf("expected ProgramEnrollment 'AlwaysOn', got %v", betaValue["ProgramEnrollment"])
	}

	offerPrograms := betaValue["OfferPrograms"].([]map[string]any)
	if len(offerPrograms) != 1 {
		t.Fatalf("expected 1 offer program, got %d", len(offerPrograms))
	}
	if offerPrograms[0]["Token"] != "offer-token-1" {
		t.Errorf("expected Token 'offer-token-1', got %v", offerPrograms[0]["Token"])
	}

	requireProgram := betaValue["RequireProgram"].(map[string]any)
	if requireProgram["Token"] != "beta-token" {
		t.Errorf("expected RequireProgram Token 'beta-token', got %v", requireProgram["Token"])
	}
	if requireProgram["Description"] != "Beta Description" {
		t.Errorf("expected RequireProgram Description 'Beta Description', got %v", requireProgram["Description"])
	}

	deferrals, ok := config["Deferrals"].(map[string]any)
	if !ok {
		t.Fatal("expected Deferrals to be a map")
	}
	combined := deferrals["CombinedPeriodInDays"].(map[string]any)
	if combined["Value"] != "30" {
		t.Errorf("expected CombinedPeriodInDays Value '30', got %v", combined["Value"])
	}

	notifications := config["Notifications"].(map[string]any)
	if notifications["Enabled"] != true {
		t.Errorf("expected Notifications Enabled true, got %v", notifications["Enabled"])
	}

	rsr := config["RapidSecurityResponse"].(map[string]any)
	enable := rsr["Enable"].(map[string]any)
	if enable["Enabled"] != true {
		t.Errorf("expected RapidSecurityResponse Enable Enabled true, got %v", enable["Enabled"])
	}
	rollback := rsr["EnableRollback"].(map[string]any)
	if rollback["Enabled"] != false {
		t.Errorf("expected RapidSecurityResponse EnableRollback Enabled false, got %v", rollback["Enabled"])
	}

	cadence := config["RecommendedCadence"].(map[string]any)
	if cadence["Value"] != "Newest" {
		t.Errorf("expected RecommendedCadence Value 'Newest', got %v", cadence["Value"])
	}
}

func TestSoftwareUpdateSettings_ToRawConfiguration_NullDefaults(t *testing.T) {
	c := &SoftwareUpdateSettingsComponent{}

	config, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	allowUpdates := config["AllowStandardUserOSUpdates"].(map[string]any)
	if allowUpdates["Included"] != false {
		t.Errorf("expected AllowStandardUserOSUpdates Included false for null, got %v", allowUpdates["Included"])
	}

	automaticActions := config["AutomaticActions"].(map[string]any)
	download := automaticActions["Download"].(map[string]any)
	if download["Value"] != "Allowed" {
		t.Errorf("expected default Download Value 'Allowed', got %v", download["Value"])
	}
	if download["Included"] != false {
		t.Errorf("expected Download Included false for null, got %v", download["Included"])
	}

	if _, exists := config["Beta"]; exists {
		t.Error("expected no Beta key for null beta settings")
	}

	cadence := config["RecommendedCadence"].(map[string]any)
	if cadence["Value"] != "All" {
		t.Errorf("expected default RecommendedCadence Value 'All', got %v", cadence["Value"])
	}
}

func TestSoftwareUpdateSettings_FromRawConfiguration(t *testing.T) {
	raw := map[string]any{
		"AllowStandardUserOSUpdates": map[string]any{
			"Enabled":  true,
			"Included": true,
		},
		"AutomaticActions": map[string]any{
			"Download": map[string]any{
				"Value":    "AlwaysOn",
				"Included": true,
			},
			"InstallOSUpdates": map[string]any{
				"Value":    "AlwaysOff",
				"Included": true,
			},
			"InstallSecurityUpdate": map[string]any{
				"Value":    "Allowed",
				"Included": true,
			},
		},
		"Beta": map[string]any{
			"Included": true,
			"Value": map[string]any{
				"ProgramEnrollment": "AlwaysOn",
				"OfferPrograms": []any{
					map[string]any{
						"Token":       "token-1",
						"Description": "Program 1",
					},
				},
				"RequireProgram": map[string]any{
					"Token":       "req-token",
					"Description": "Req Desc",
				},
			},
		},
		"Deferrals": map[string]any{
			"CombinedPeriodInDays": map[string]any{
				"Value":    "30",
				"Included": true,
			},
			"MajorPeriodInDays": map[string]any{
				"Value":    "60",
				"Included": true,
			},
			"MinorPeriodInDays": map[string]any{
				"Value":    "14",
				"Included": true,
			},
			"SystemPeriodInDays": map[string]any{
				"Value":    "7",
				"Included": true,
			},
		},
		"Notifications": map[string]any{
			"Enabled":  true,
			"Included": true,
		},
		"RapidSecurityResponse": map[string]any{
			"Enable": map[string]any{
				"Enabled":  true,
				"Included": true,
			},
			"EnableRollback": map[string]any{
				"Enabled":  false,
				"Included": true,
			},
		},
		"RecommendedCadence": map[string]any{
			"Value":    "Newest",
			"Included": true,
		},
	}

	c := &SoftwareUpdateSettingsComponent{}
	if err := c.FromRawConfiguration(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.AllowStandardUserOSUpdates.ValueBool() != true {
		t.Errorf("expected AllowStandardUserOSUpdates true, got %v", c.AllowStandardUserOSUpdates.ValueBool())
	}
	if c.AutomaticDownload.ValueString() != "AlwaysOn" {
		t.Errorf("expected AutomaticDownload 'AlwaysOn', got %q", c.AutomaticDownload.ValueString())
	}
	if c.AutomaticInstallOSUpdates.ValueString() != "AlwaysOff" {
		t.Errorf("expected AutomaticInstallOSUpdates 'AlwaysOff', got %q", c.AutomaticInstallOSUpdates.ValueString())
	}
	if c.AutomaticInstallSecurityUpdate.ValueString() != "Allowed" {
		t.Errorf("expected AutomaticInstallSecurityUpdate 'Allowed', got %q", c.AutomaticInstallSecurityUpdate.ValueString())
	}
	if c.BetaProgramEnrollment.ValueString() != "AlwaysOn" {
		t.Errorf("expected BetaProgramEnrollment 'AlwaysOn', got %q", c.BetaProgramEnrollment.ValueString())
	}
	if len(c.BetaOfferPrograms) != 1 {
		t.Fatalf("expected 1 beta offer program, got %d", len(c.BetaOfferPrograms))
	}
	if c.BetaOfferPrograms[0].Token.ValueString() != "token-1" {
		t.Errorf("expected offer Token 'token-1', got %q", c.BetaOfferPrograms[0].Token.ValueString())
	}
	if c.BetaOfferPrograms[0].Description.ValueString() != "Program 1" {
		t.Errorf("expected offer Description 'Program 1', got %q", c.BetaOfferPrograms[0].Description.ValueString())
	}
	if c.BetaRequireProgramToken.ValueString() != "req-token" {
		t.Errorf("expected BetaRequireProgramToken 'req-token', got %q", c.BetaRequireProgramToken.ValueString())
	}
	if c.BetaRequireProgramDescription.ValueString() != "Req Desc" {
		t.Errorf("expected BetaRequireProgramDescription 'Req Desc', got %q", c.BetaRequireProgramDescription.ValueString())
	}
	if c.DeferralCombinedPeriod.ValueString() != "30" {
		t.Errorf("expected DeferralCombinedPeriod '30', got %q", c.DeferralCombinedPeriod.ValueString())
	}
	if c.DeferralMajorPeriod.ValueString() != "60" {
		t.Errorf("expected DeferralMajorPeriod '60', got %q", c.DeferralMajorPeriod.ValueString())
	}
	if c.DeferralMinorPeriod.ValueString() != "14" {
		t.Errorf("expected DeferralMinorPeriod '14', got %q", c.DeferralMinorPeriod.ValueString())
	}
	if c.DeferralSystemPeriod.ValueString() != "7" {
		t.Errorf("expected DeferralSystemPeriod '7', got %q", c.DeferralSystemPeriod.ValueString())
	}
	if c.NotificationsEnabled.ValueBool() != true {
		t.Errorf("expected NotificationsEnabled true, got %v", c.NotificationsEnabled.ValueBool())
	}
	if c.RapidSecurityResponseEnabled.ValueBool() != true {
		t.Errorf("expected RapidSecurityResponseEnabled true, got %v", c.RapidSecurityResponseEnabled.ValueBool())
	}
	if c.RapidSecurityResponseRollbackEnabled.ValueBool() != false {
		t.Errorf("expected RapidSecurityResponseRollbackEnabled false, got %v", c.RapidSecurityResponseRollbackEnabled.ValueBool())
	}
	if c.RecommendedCadence.ValueString() != "Newest" {
		t.Errorf("expected RecommendedCadence 'Newest', got %q", c.RecommendedCadence.ValueString())
	}
}

func TestSoftwareUpdateSettings_FromRawConfiguration_NotIncluded(t *testing.T) {
	raw := map[string]any{
		"AllowStandardUserOSUpdates": map[string]any{
			"Enabled":  false,
			"Included": false,
		},
		"AutomaticActions": map[string]any{
			"Download": map[string]any{
				"Value":    "Allowed",
				"Included": false,
			},
			"InstallOSUpdates": map[string]any{
				"Value":    "Allowed",
				"Included": false,
			},
			"InstallSecurityUpdate": map[string]any{
				"Value":    "Allowed",
				"Included": false,
			},
		},
		"Deferrals": map[string]any{
			"CombinedPeriodInDays": map[string]any{
				"Value":    "",
				"Included": false,
			},
			"MajorPeriodInDays": map[string]any{
				"Value":    "",
				"Included": false,
			},
			"MinorPeriodInDays": map[string]any{
				"Value":    "",
				"Included": false,
			},
			"SystemPeriodInDays": map[string]any{
				"Value":    "",
				"Included": false,
			},
		},
		"Notifications": map[string]any{
			"Enabled":  false,
			"Included": false,
		},
		"RapidSecurityResponse": map[string]any{
			"Enable": map[string]any{
				"Enabled":  false,
				"Included": false,
			},
			"EnableRollback": map[string]any{
				"Enabled":  false,
				"Included": false,
			},
		},
		"RecommendedCadence": map[string]any{
			"Value":    "All",
			"Included": false,
		},
	}

	c := &SoftwareUpdateSettingsComponent{}
	if err := c.FromRawConfiguration(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !c.AllowStandardUserOSUpdates.IsNull() {
		t.Error("expected null AllowStandardUserOSUpdates when not included")
	}
	if !c.AutomaticDownload.IsNull() {
		t.Error("expected null AutomaticDownload when not included")
	}
	if !c.DeferralCombinedPeriod.IsNull() {
		t.Error("expected null DeferralCombinedPeriod when not included")
	}
	if !c.NotificationsEnabled.IsNull() {
		t.Error("expected null NotificationsEnabled when not included")
	}
	if !c.RecommendedCadence.IsNull() {
		t.Error("expected null RecommendedCadence when not included")
	}
}

func TestSoftwareUpdateSettings_Roundtrip(t *testing.T) {
	original := &SoftwareUpdateSettingsComponent{
		AllowStandardUserOSUpdates:           types.BoolValue(true),
		AutomaticDownload:                    types.StringValue("AlwaysOn"),
		AutomaticInstallOSUpdates:            types.StringValue("AlwaysOff"),
		AutomaticInstallSecurityUpdate:       types.StringValue("Allowed"),
		DeferralCombinedPeriod:               types.StringValue("15"),
		DeferralMajorPeriod:                  types.StringValue("30"),
		DeferralMinorPeriod:                  types.StringValue("7"),
		DeferralSystemPeriod:                 types.StringValue("14"),
		NotificationsEnabled:                 types.BoolValue(true),
		RapidSecurityResponseEnabled:         types.BoolValue(true),
		RapidSecurityResponseRollbackEnabled: types.BoolValue(false),
		RecommendedCadence:                   types.StringValue("Oldest"),
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

	restored := &SoftwareUpdateSettingsComponent{}
	if err := restored.FromRawConfiguration(parsed); err != nil {
		t.Fatalf("FromRawConfiguration error: %v", err)
	}

	if restored.AllowStandardUserOSUpdates.ValueBool() != true {
		t.Errorf("roundtrip: expected AllowStandardUserOSUpdates true, got %v", restored.AllowStandardUserOSUpdates.ValueBool())
	}
	if restored.AutomaticDownload.ValueString() != "AlwaysOn" {
		t.Errorf("roundtrip: expected AutomaticDownload 'AlwaysOn', got %q", restored.AutomaticDownload.ValueString())
	}
	if restored.AutomaticInstallOSUpdates.ValueString() != "AlwaysOff" {
		t.Errorf("roundtrip: expected AutomaticInstallOSUpdates 'AlwaysOff', got %q", restored.AutomaticInstallOSUpdates.ValueString())
	}
	if restored.DeferralCombinedPeriod.ValueString() != "15" {
		t.Errorf("roundtrip: expected DeferralCombinedPeriod '15', got %q", restored.DeferralCombinedPeriod.ValueString())
	}
	if restored.NotificationsEnabled.ValueBool() != true {
		t.Errorf("roundtrip: expected NotificationsEnabled true, got %v", restored.NotificationsEnabled.ValueBool())
	}
	if restored.RapidSecurityResponseEnabled.ValueBool() != true {
		t.Errorf("roundtrip: expected RapidSecurityResponseEnabled true, got %v", restored.RapidSecurityResponseEnabled.ValueBool())
	}
	if restored.RecommendedCadence.ValueString() != "Oldest" {
		t.Errorf("roundtrip: expected RecommendedCadence 'Oldest', got %q", restored.RecommendedCadence.ValueString())
	}
}

func TestSoftwareUpdateSettings_ToClientComponent(t *testing.T) {
	c := &SoftwareUpdateSettingsComponent{
		AllowStandardUserOSUpdates: types.BoolValue(true),
		NotificationsEnabled:       types.BoolValue(false),
	}

	comp, err := c.ToClientComponent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if comp.Identifier != "com.jamf.ddm.software-update-settings" {
		t.Errorf("expected identifier 'com.jamf.ddm.software-update-settings', got %q", comp.Identifier)
	}
	if comp.Configuration == nil {
		t.Fatal("expected non-nil configuration")
	}
}
