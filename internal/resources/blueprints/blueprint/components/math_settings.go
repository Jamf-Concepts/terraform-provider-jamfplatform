// Copyright 2025 Jamf Software LLC.

package components

import (
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
				Validators: []validator.Bool{
					boolvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("calculator_input_modes_rpn")),
				},
			},
			"calculator_input_modes_rpn": schema.BoolAttribute{
				Description: "Configures whether RPN input is enabled in Calculator.",
				Optional:    true,
				Validators: []validator.Bool{
					boolvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("calculator_input_modes_unit_conversion")),
				},
			},
			"system_behavior_keyboard_suggestions": schema.BoolAttribute{
				Description: "Controls whether keyboard suggestions include math solutions.",
				Optional:    true,
				Validators: []validator.Bool{
					boolvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("system_behavior_math_notes")),
				},
			},
			"system_behavior_math_notes": schema.BoolAttribute{
				Description: "Controls whether Math Notes is allowed in other apps such as Notes.",
				Optional:    true,
				Validators: []validator.Bool{
					boolvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("system_behavior_keyboard_suggestions")),
				},
			},
		},
	}
}

// ToRawConfiguration converts the typed component to raw configuration matching OpenAPI MathSettingsConfiguration schema
func (c *MathSettingsComponent) ToRawConfiguration() (map[string]interface{}, error) {
	config := make(map[string]interface{})

	calculator := map[string]interface{}{
		"BasicMode":      setBoolFieldWithKey(c.CalculatorBasicModeAddSquareRoot, "AddSquareRoot", true),
		"ScientificMode": setBoolField(c.CalculatorScientificModeEnabled, true),
		"ProgrammerMode": setBoolField(c.CalculatorProgrammerModeEnabled, true),
		"MathNotesMode":  setBoolField(c.CalculatorMathNotesModeEnabled, true),
	}

	hasInputModes := (!c.CalculatorInputModesUnitConversion.IsNull() && !c.CalculatorInputModesUnitConversion.IsUnknown()) ||
		(!c.CalculatorInputModesRPN.IsNull() && !c.CalculatorInputModesRPN.IsUnknown())

	inputModes := make(map[string]interface{})
	if hasInputModes {
		inputModes["Included"] = true
		if !c.CalculatorInputModesUnitConversion.IsNull() && !c.CalculatorInputModesUnitConversion.IsUnknown() {
			inputModes["UnitConversion"] = c.CalculatorInputModesUnitConversion.ValueBool()
		} else {
			inputModes["UnitConversion"] = true
		}
		if !c.CalculatorInputModesRPN.IsNull() && !c.CalculatorInputModesRPN.IsUnknown() {
			inputModes["RPN"] = c.CalculatorInputModesRPN.ValueBool()
		} else {
			inputModes["RPN"] = true
		}
	} else {
		inputModes["Included"] = false
		inputModes["UnitConversion"] = true
		inputModes["RPN"] = true
	}
	calculator["InputModes"] = inputModes

	config["Calculator"] = calculator

	hasSystemBehavior := (!c.SystemBehaviorKeyboardSuggestions.IsNull() && !c.SystemBehaviorKeyboardSuggestions.IsUnknown()) ||
		(!c.SystemBehaviorMathNotes.IsNull() && !c.SystemBehaviorMathNotes.IsUnknown())

	systemBehavior := make(map[string]interface{})
	if hasSystemBehavior {
		systemBehavior["Included"] = true
		if !c.SystemBehaviorKeyboardSuggestions.IsNull() && !c.SystemBehaviorKeyboardSuggestions.IsUnknown() {
			systemBehavior["KeyboardSuggestions"] = c.SystemBehaviorKeyboardSuggestions.ValueBool()
		} else {
			systemBehavior["KeyboardSuggestions"] = true
		}
		if !c.SystemBehaviorMathNotes.IsNull() && !c.SystemBehaviorMathNotes.IsUnknown() {
			systemBehavior["MathNotes"] = c.SystemBehaviorMathNotes.ValueBool()
		} else {
			systemBehavior["MathNotes"] = true
		}
	} else {
		systemBehavior["Included"] = false
		systemBehavior["KeyboardSuggestions"] = true
		systemBehavior["MathNotes"] = true
	}
	config["SystemBehavior"] = systemBehavior

	return config, nil
}

// FromRawConfiguration populates the typed component from raw configuration data
func (c *MathSettingsComponent) FromRawConfiguration(raw map[string]interface{}) error {
	extractBool := func(path ...string) types.Bool {
		current := raw
		for i, key := range path {
			if next, exists := current[key]; exists {
				if nextMap, ok := next.(map[string]interface{}); ok {
					if i == len(path)-1 {
						if included, hasIncluded := nextMap["Included"]; hasIncluded && included.(bool) {
							for _, valueKey := range []string{"Enabled", "AddSquareRoot"} {
								if value, hasValue := nextMap[valueKey]; hasValue {
									if boolVal, ok := value.(bool); ok {
										return types.BoolValue(boolVal)
									}
								}
							}
						}
						return types.BoolNull()
					}
					current = nextMap
				} else {
					return types.BoolNull()
				}
			} else {
				return types.BoolNull()
			}
		}
		return types.BoolNull()
	}

	extractGroupBool := func(groupPath []string, fieldKey string) types.Bool {
		current := raw
		for _, key := range groupPath {
			if next, exists := current[key]; exists {
				if nextMap, ok := next.(map[string]interface{}); ok {
					current = nextMap
				} else {
					return types.BoolNull()
				}
			} else {
				return types.BoolNull()
			}
		}

		if included, hasIncluded := current["Included"]; hasIncluded && included.(bool) {
			if value, hasValue := current[fieldKey]; hasValue {
				if boolVal, ok := value.(bool); ok {
					return types.BoolValue(boolVal)
				}
			}
		}
		return types.BoolNull()
	}

	c.CalculatorBasicModeAddSquareRoot = extractBool("Calculator", "BasicMode")
	c.CalculatorScientificModeEnabled = extractBool("Calculator", "ScientificMode")
	c.CalculatorProgrammerModeEnabled = extractBool("Calculator", "ProgrammerMode")
	c.CalculatorMathNotesModeEnabled = extractBool("Calculator", "MathNotesMode")

	c.CalculatorInputModesUnitConversion = extractGroupBool([]string{"Calculator", "InputModes"}, "UnitConversion")
	c.CalculatorInputModesRPN = extractGroupBool([]string{"Calculator", "InputModes"}, "RPN")

	c.SystemBehaviorKeyboardSuggestions = extractGroupBool([]string{"SystemBehavior"}, "KeyboardSuggestions")
	c.SystemBehaviorMathNotes = extractGroupBool([]string{"SystemBehavior"}, "MathNotes")

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
