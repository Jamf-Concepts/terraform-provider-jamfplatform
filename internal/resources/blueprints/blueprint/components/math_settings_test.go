// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMathSettings_GetIdentifier(t *testing.T) {
	c := &MathSettingsComponent{}
	if c.GetIdentifier() != "com.jamf.ddm.math-settings" {
		t.Errorf("expected 'com.jamf.ddm.math-settings', got %q", c.GetIdentifier())
	}
}

func TestMathSettings_ToRawConfiguration_AllConfigured(t *testing.T) {
	c := &MathSettingsComponent{
		CalculatorBasicModeAddSquareRoot:   types.BoolValue(true),
		CalculatorScientificModeEnabled:    types.BoolValue(false),
		CalculatorProgrammerModeEnabled:    types.BoolValue(true),
		CalculatorMathNotesModeEnabled:     types.BoolValue(false),
		CalculatorInputModesUnitConversion: types.BoolValue(true),
		CalculatorInputModesRPN:            types.BoolValue(false),
		SystemBehaviorKeyboardSuggestions:  types.BoolValue(true),
		SystemBehaviorMathNotes:            types.BoolValue(false),
	}

	raw, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var config map[string]any
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	calc, ok := config["Calculator"].(map[string]any)
	if !ok {
		t.Fatal("expected Calculator to be a map")
	}

	basicMode, ok := calc["BasicMode"].(map[string]any)
	if !ok {
		t.Fatal("expected BasicMode to be a map")
	}
	if basicMode["AddSquareRoot"] != true {
		t.Errorf("expected AddSquareRoot true, got %v", basicMode["AddSquareRoot"])
	}
	if basicMode["Included"] != true {
		t.Errorf("expected BasicMode Included true, got %v", basicMode["Included"])
	}

	inputModes, ok := calc["InputModes"].(map[string]any)
	if !ok {
		t.Fatal("expected InputModes to be a map")
	}
	if inputModes["UnitConversion"] != true {
		t.Errorf("expected UnitConversion true, got %v", inputModes["UnitConversion"])
	}
	if inputModes["RPN"] != false {
		t.Errorf("expected RPN false, got %v", inputModes["RPN"])
	}
	if inputModes["Included"] != true {
		t.Errorf("expected InputModes Included true, got %v", inputModes["Included"])
	}

	sysBehavior, ok := config["SystemBehavior"].(map[string]any)
	if !ok {
		t.Fatal("expected SystemBehavior to be a map")
	}
	if sysBehavior["KeyboardSuggestions"] != true {
		t.Errorf("expected KeyboardSuggestions true, got %v", sysBehavior["KeyboardSuggestions"])
	}
	if sysBehavior["MathNotes"] != false {
		t.Errorf("expected MathNotes false, got %v", sysBehavior["MathNotes"])
	}
	if sysBehavior["Included"] != true {
		t.Errorf("expected SystemBehavior Included true, got %v", sysBehavior["Included"])
	}
}

func TestMathSettings_ToRawConfiguration_NullDefaults(t *testing.T) {
	c := &MathSettingsComponent{}

	raw, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var config map[string]any
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	calc := config["Calculator"].(map[string]any)
	inputModes := calc["InputModes"].(map[string]any)
	if inputModes["Included"] != false {
		t.Errorf("expected InputModes Included false for null, got %v", inputModes["Included"])
	}
	if inputModes["UnitConversion"] != true {
		t.Errorf("expected default UnitConversion true, got %v", inputModes["UnitConversion"])
	}

	sysBehavior := config["SystemBehavior"].(map[string]any)
	if sysBehavior["Included"] != false {
		t.Errorf("expected SystemBehavior Included false for null, got %v", sysBehavior["Included"])
	}
}

func TestMathSettings_FromRawConfiguration(t *testing.T) {
	inputMap := map[string]any{
		"Calculator": map[string]any{
			"BasicMode": map[string]any{
				"Included":      true,
				"AddSquareRoot": true,
			},
			"ScientificMode": map[string]any{
				"Included": true,
				"Enabled":  false,
			},
			"ProgrammerMode": map[string]any{
				"Included": false,
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
			"KeyboardSuggestions": false,
			"MathNotes":           true,
		},
	}
	raw, _ := json.Marshal(inputMap)

	c := &MathSettingsComponent{}
	if err := c.FromRawConfiguration(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.CalculatorBasicModeAddSquareRoot.ValueBool() != true {
		t.Errorf("expected BasicModeAddSquareRoot true, got %v", c.CalculatorBasicModeAddSquareRoot.ValueBool())
	}
	if c.CalculatorScientificModeEnabled.ValueBool() != false {
		t.Errorf("expected ScientificModeEnabled false, got %v", c.CalculatorScientificModeEnabled.ValueBool())
	}
	if !c.CalculatorProgrammerModeEnabled.IsNull() {
		t.Error("expected null ProgrammerModeEnabled when not included")
	}
	if c.CalculatorMathNotesModeEnabled.ValueBool() != true {
		t.Errorf("expected MathNotesModeEnabled true, got %v", c.CalculatorMathNotesModeEnabled.ValueBool())
	}
	if c.CalculatorInputModesUnitConversion.ValueBool() != true {
		t.Errorf("expected UnitConversion true, got %v", c.CalculatorInputModesUnitConversion.ValueBool())
	}
	if c.CalculatorInputModesRPN.ValueBool() != false {
		t.Errorf("expected RPN false, got %v", c.CalculatorInputModesRPN.ValueBool())
	}
	if c.SystemBehaviorKeyboardSuggestions.ValueBool() != false {
		t.Errorf("expected KeyboardSuggestions false, got %v", c.SystemBehaviorKeyboardSuggestions.ValueBool())
	}
	if c.SystemBehaviorMathNotes.ValueBool() != true {
		t.Errorf("expected MathNotes true, got %v", c.SystemBehaviorMathNotes.ValueBool())
	}
}

func TestMathSettings_Roundtrip(t *testing.T) {
	original := &MathSettingsComponent{
		CalculatorBasicModeAddSquareRoot:   types.BoolValue(false),
		CalculatorScientificModeEnabled:    types.BoolValue(true),
		CalculatorProgrammerModeEnabled:    types.BoolValue(true),
		CalculatorMathNotesModeEnabled:     types.BoolValue(false),
		CalculatorInputModesUnitConversion: types.BoolValue(false),
		CalculatorInputModesRPN:            types.BoolValue(true),
		SystemBehaviorKeyboardSuggestions:  types.BoolValue(false),
		SystemBehaviorMathNotes:            types.BoolValue(true),
	}

	raw, err := original.ToRawConfiguration()
	if err != nil {
		t.Fatalf("ToRawConfiguration error: %v", err)
	}

	restored := &MathSettingsComponent{}
	if err := restored.FromRawConfiguration(raw); err != nil {
		t.Fatalf("FromRawConfiguration error: %v", err)
	}

	if restored.CalculatorBasicModeAddSquareRoot.ValueBool() != false {
		t.Errorf("roundtrip: expected AddSquareRoot false, got %v", restored.CalculatorBasicModeAddSquareRoot.ValueBool())
	}
	if restored.CalculatorScientificModeEnabled.ValueBool() != true {
		t.Errorf("roundtrip: expected ScientificMode true, got %v", restored.CalculatorScientificModeEnabled.ValueBool())
	}
	if restored.CalculatorInputModesUnitConversion.ValueBool() != false {
		t.Errorf("roundtrip: expected UnitConversion false, got %v", restored.CalculatorInputModesUnitConversion.ValueBool())
	}
	if restored.CalculatorInputModesRPN.ValueBool() != true {
		t.Errorf("roundtrip: expected RPN true, got %v", restored.CalculatorInputModesRPN.ValueBool())
	}
	if restored.SystemBehaviorMathNotes.ValueBool() != true {
		t.Errorf("roundtrip: expected MathNotes true, got %v", restored.SystemBehaviorMathNotes.ValueBool())
	}
}

func TestMathSettings_ToClientComponent(t *testing.T) {
	c := &MathSettingsComponent{
		CalculatorBasicModeAddSquareRoot: types.BoolValue(true),
	}

	comp, err := c.ToClientComponent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if comp.Identifier != "com.jamf.ddm.math-settings" {
		t.Errorf("expected identifier 'com.jamf.ddm.math-settings', got %q", comp.Identifier)
	}
	if comp.Configuration == nil {
		t.Fatal("expected non-nil configuration")
	}
}
