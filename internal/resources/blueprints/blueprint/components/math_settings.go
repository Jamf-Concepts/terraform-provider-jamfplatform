// Copyright 2025 Jamf Software LLC.

package components

import (
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// MathSettingsComponent represents a strongly-typed math settings component
type MathSettingsComponent struct {
	CalculatorBasicModeAddSquareRoot   types.Bool `tfsdk:"calculator_basic_mode_add_square_root"`
	CalculatorScientificModeEnabled    types.Bool `tfsdk:"calculator_scientific_mode_enabled"`
	CalculatorProgrammerModeEnabled    types.Bool `tfsdk:"calculator_programmer_mode_enabled"`
	CalculatorMathNotesModeEnabled     types.Bool `tfsdk:"calculator_math_notes_mode_enabled"`
	CalculatorInputModesUnitConversion types.Bool `tfsdk:"calculator_input_modes_unit_conversion"`
	CalculatorInputModesRPN            types.Bool `tfsdk:"calculator_input_modes_rpn"`
	SystemBehaviorKeyboardSuggestions  types.Bool `tfsdk:"system_behavior_keyboard_suggestions"`
	SystemBehaviorMathNotes            types.Bool `tfsdk:"system_behavior_math_notes"`
}

// GetIdentifier returns the component identifier for math settings
func (c *MathSettingsComponent) GetIdentifier() string {
	return "com.jamf.ddm.math-settings"
}

// MathSettingsComponentSchema returns the Terraform schema for math settings component
func MathSettingsComponentSchema() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"calculator_basic_mode_add_square_root": schema.BoolAttribute{
				Description: "Add the square root button to the basic calculator by replacing the +/- button.",
				Optional:    true,
			},
			"calculator_scientific_mode_enabled": schema.BoolAttribute{
				Description: "Controls whether the scientific mode is enabled in Calculator.",
				Optional:    true,
			},
			"calculator_programmer_mode_enabled": schema.BoolAttribute{
				Description: "Controls whether the programmer mode is enabled in Calculator.",
				Optional:    true,
			},
			"calculator_math_notes_mode_enabled": schema.BoolAttribute{
				Description: "Controls whether the Math Notes mode is enabled in Calculator.",
				Optional:    true,
			},
			"calculator_input_modes_unit_conversion": schema.BoolAttribute{
				Description: "Configures whether unit conversions are enabled in Calculator.",
				Optional:    true,
			},
			"calculator_input_modes_rpn": schema.BoolAttribute{
				Description: "Configures whether RPN input is enabled in Calculator.",
				Optional:    true,
			},
			"system_behavior_keyboard_suggestions": schema.BoolAttribute{
				Description: "Controls whether keyboard suggestions include math solutions.",
				Optional:    true,
			},
			"system_behavior_math_notes": schema.BoolAttribute{
				Description: "Controls whether Math Notes is allowed in other apps such as Notes.",
				Optional:    true,
			},
		},
	}
}

// ToRawConfiguration converts the typed component to raw configuration matching OpenAPI MathSettingsConfiguration schema
func (c *MathSettingsComponent) ToRawConfiguration() (map[string]interface{}, error) {
	config := make(map[string]interface{})

	if (!c.CalculatorBasicModeAddSquareRoot.IsNull() && !c.CalculatorBasicModeAddSquareRoot.IsUnknown()) ||
		(!c.CalculatorScientificModeEnabled.IsNull() && !c.CalculatorScientificModeEnabled.IsUnknown()) ||
		(!c.CalculatorProgrammerModeEnabled.IsNull() && !c.CalculatorProgrammerModeEnabled.IsUnknown()) ||
		(!c.CalculatorMathNotesModeEnabled.IsNull() && !c.CalculatorMathNotesModeEnabled.IsUnknown()) ||
		(!c.CalculatorInputModesUnitConversion.IsNull() && !c.CalculatorInputModesUnitConversion.IsUnknown()) ||
		(!c.CalculatorInputModesRPN.IsNull() && !c.CalculatorInputModesRPN.IsUnknown()) {

		calculator := make(map[string]interface{})

		if !c.CalculatorBasicModeAddSquareRoot.IsNull() && !c.CalculatorBasicModeAddSquareRoot.IsUnknown() {
			calculator["BasicMode"] = map[string]interface{}{
				"AddSquareRoot": c.CalculatorBasicModeAddSquareRoot.ValueBool(),
				"Included":      true,
			}
		}

		if !c.CalculatorScientificModeEnabled.IsNull() && !c.CalculatorScientificModeEnabled.IsUnknown() {
			calculator["ScientificMode"] = map[string]interface{}{
				"Enabled":  c.CalculatorScientificModeEnabled.ValueBool(),
				"Included": true,
			}
		}

		if !c.CalculatorProgrammerModeEnabled.IsNull() && !c.CalculatorProgrammerModeEnabled.IsUnknown() {
			calculator["ProgrammerMode"] = map[string]interface{}{
				"Enabled":  c.CalculatorProgrammerModeEnabled.ValueBool(),
				"Included": true,
			}
		}

		if !c.CalculatorMathNotesModeEnabled.IsNull() && !c.CalculatorMathNotesModeEnabled.IsUnknown() {
			calculator["MathNotesMode"] = map[string]interface{}{
				"Enabled":  c.CalculatorMathNotesModeEnabled.ValueBool(),
				"Included": true,
			}
		}

		if (!c.CalculatorInputModesUnitConversion.IsNull() && !c.CalculatorInputModesUnitConversion.IsUnknown()) ||
			(!c.CalculatorInputModesRPN.IsNull() && !c.CalculatorInputModesRPN.IsUnknown()) {

			inputModes := map[string]interface{}{
				"Included": true,
			}

			if !c.CalculatorInputModesUnitConversion.IsNull() && !c.CalculatorInputModesUnitConversion.IsUnknown() {
				inputModes["UnitConversion"] = c.CalculatorInputModesUnitConversion.ValueBool()
			}

			if !c.CalculatorInputModesRPN.IsNull() && !c.CalculatorInputModesRPN.IsUnknown() {
				inputModes["RPN"] = c.CalculatorInputModesRPN.ValueBool()
			}

			calculator["InputModes"] = inputModes
		}

		config["Calculator"] = calculator
	}

	if (!c.SystemBehaviorKeyboardSuggestions.IsNull() && !c.SystemBehaviorKeyboardSuggestions.IsUnknown()) ||
		(!c.SystemBehaviorMathNotes.IsNull() && !c.SystemBehaviorMathNotes.IsUnknown()) {

		systemBehavior := map[string]interface{}{
			"Included": true,
		}

		if !c.SystemBehaviorKeyboardSuggestions.IsNull() && !c.SystemBehaviorKeyboardSuggestions.IsUnknown() {
			systemBehavior["KeyboardSuggestions"] = c.SystemBehaviorKeyboardSuggestions.ValueBool()
		}

		if !c.SystemBehaviorMathNotes.IsNull() && !c.SystemBehaviorMathNotes.IsUnknown() {
			systemBehavior["MathNotes"] = c.SystemBehaviorMathNotes.ValueBool()
		}

		config["SystemBehavior"] = systemBehavior
	}

	return config, nil
}

// FromRawConfiguration populates the typed component from raw configuration data
func (c *MathSettingsComponent) FromRawConfiguration(raw map[string]interface{}) error {
	if calculatorRaw, exists := raw["Calculator"]; exists {
		if calculatorMap, ok := calculatorRaw.(map[string]interface{}); ok {
			if basicModeRaw, exists := calculatorMap["BasicMode"]; exists {
				if basicModeMap, ok := basicModeRaw.(map[string]interface{}); ok {
					if addSquareRoot, exists := basicModeMap["AddSquareRoot"]; exists {
						if addSquareRootBool, ok := addSquareRoot.(bool); ok {
							c.CalculatorBasicModeAddSquareRoot = types.BoolValue(addSquareRootBool)
						}
					}
				}
			}

			if scientificModeRaw, exists := calculatorMap["ScientificMode"]; exists {
				if scientificModeMap, ok := scientificModeRaw.(map[string]interface{}); ok {
					if enabled, exists := scientificModeMap["Enabled"]; exists {
						if enabledBool, ok := enabled.(bool); ok {
							c.CalculatorScientificModeEnabled = types.BoolValue(enabledBool)
						}
					}
				}
			}

			if programmerModeRaw, exists := calculatorMap["ProgrammerMode"]; exists {
				if programmerModeMap, ok := programmerModeRaw.(map[string]interface{}); ok {
					if enabled, exists := programmerModeMap["Enabled"]; exists {
						if enabledBool, ok := enabled.(bool); ok {
							c.CalculatorProgrammerModeEnabled = types.BoolValue(enabledBool)
						}
					}
				}
			}

			if mathNotesModeRaw, exists := calculatorMap["MathNotesMode"]; exists {
				if mathNotesModeMap, ok := mathNotesModeRaw.(map[string]interface{}); ok {
					if enabled, exists := mathNotesModeMap["Enabled"]; exists {
						if enabledBool, ok := enabled.(bool); ok {
							c.CalculatorMathNotesModeEnabled = types.BoolValue(enabledBool)
						}
					}
				}
			}

			if inputModesRaw, exists := calculatorMap["InputModes"]; exists {
				if inputModesMap, ok := inputModesRaw.(map[string]interface{}); ok {
					if unitConversion, exists := inputModesMap["UnitConversion"]; exists {
						if unitConversionBool, ok := unitConversion.(bool); ok {
							c.CalculatorInputModesUnitConversion = types.BoolValue(unitConversionBool)
						}
					}
					if rpn, exists := inputModesMap["RPN"]; exists {
						if rpnBool, ok := rpn.(bool); ok {
							c.CalculatorInputModesRPN = types.BoolValue(rpnBool)
						}
					}
				}
			}
		}
	}

	if systemBehaviorRaw, exists := raw["SystemBehavior"]; exists {
		if systemBehaviorMap, ok := systemBehaviorRaw.(map[string]interface{}); ok {
			if keyboardSuggestions, exists := systemBehaviorMap["KeyboardSuggestions"]; exists {
				if keyboardSuggestionsBool, ok := keyboardSuggestions.(bool); ok {
					c.SystemBehaviorKeyboardSuggestions = types.BoolValue(keyboardSuggestionsBool)
				}
			}
			if mathNotes, exists := systemBehaviorMap["MathNotes"]; exists {
				if mathNotesBool, ok := mathNotes.(bool); ok {
					c.SystemBehaviorMathNotes = types.BoolValue(mathNotesBool)
				}
			}
		}
	}

	return nil
}

// ToClientComponent converts the typed component to the format expected by the Blueprint API client
func (c *MathSettingsComponent) ToClientComponent() (*BlueprintComponentData, error) {
	config, err := c.ToRawConfiguration()
	if err != nil {
		return nil, err
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	return &BlueprintComponentData{
		Identifier:    c.GetIdentifier(),
		Configuration: json.RawMessage(configJSON),
	}, nil
}
