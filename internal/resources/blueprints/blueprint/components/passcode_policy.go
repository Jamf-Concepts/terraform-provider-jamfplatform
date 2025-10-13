// Copyright 2025 Jamf Software LLC.

package components

import (
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// PasscodePolicyComponent represents a strongly-typed passcode policy component
type PasscodePolicyComponent struct {
	ChangeAtNextAuth             types.Bool  `tfsdk:"change_at_next_auth"`
	FailedAttemptsResetInMinutes types.Int64 `tfsdk:"failed_attempts_reset_in_minutes"`
	MaximumFailedAttempts        types.Int64 `tfsdk:"maximum_failed_attempts"`
	MaximumGracePeriodInMinutes  types.Int64 `tfsdk:"maximum_grace_period_in_minutes"`
	MaximumInactivityInMinutes   types.Int64 `tfsdk:"maximum_inactivity_in_minutes"`
	MaximumPasscodeAgeInDays     types.Int64 `tfsdk:"maximum_passcode_age_in_days"`
	MinimumComplexCharacters     types.Int64 `tfsdk:"minimum_complex_characters"`
	MinimumLength                types.Int64 `tfsdk:"minimum_length"`
	PasscodeReuseLimit           types.Int64 `tfsdk:"passcode_reuse_limit"`
	RequireAlphanumericPasscode  types.Bool  `tfsdk:"require_alphanumeric_passcode"`
	RequireComplexPasscode       types.Bool  `tfsdk:"require_complex_passcode"`
	RequirePasscode              types.Bool  `tfsdk:"require_passcode"`
}

// GetIdentifier returns the component identifier for passcode policy
func (c *PasscodePolicyComponent) GetIdentifier() string {
	return "com.jamf.ddm.passcode-settings"
}

// PasscodePolicyComponentSchema returns the Terraform schema for passcode policy component
func PasscodePolicyComponentSchema() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"change_at_next_auth": schema.BoolAttribute{
				Description: "Change at next auth.",
				Optional:    true,
			},
			"failed_attempts_reset_in_minutes": schema.Int64Attribute{
				Description: "Failed attempts reset in minutes. Minimum: 0.",
				Optional:    true,
			},
			"maximum_failed_attempts": schema.Int64Attribute{
				Description: "Maximum failed attempts. Range: 2-11.",
				Optional:    true,
				Validators:  []validator.Int64{int64validator.Between(2, 11)},
			},
			"maximum_grace_period_in_minutes": schema.Int64Attribute{
				Description: "Maximum grace period in minutes. Minimum: 0.",
				Optional:    true,
			},
			"maximum_inactivity_in_minutes": schema.Int64Attribute{
				Description: "Maximum inactivity in minutes. Range: 0-15.",
				Optional:    true,
				Validators:  []validator.Int64{int64validator.Between(0, 15)},
			},
			"maximum_passcode_age_in_days": schema.Int64Attribute{
				Description: "Maximum passcode age in days. Range: 0-730.",
				Optional:    true,
				Validators:  []validator.Int64{int64validator.Between(0, 730)},
			},
			"minimum_complex_characters": schema.Int64Attribute{
				Description: "Minimum complex characters. Range: 0-4.",
				Optional:    true,
				Validators:  []validator.Int64{int64validator.Between(0, 4)},
			},
			"minimum_length": schema.Int64Attribute{
				Description: "Minimum length. Range: 0-16.",
				Optional:    true,
				Validators:  []validator.Int64{int64validator.Between(0, 16)},
			},
			"passcode_reuse_limit": schema.Int64Attribute{
				Description: "Passcode reuse limit. Range: 1-50.",
				Optional:    true,
				Validators:  []validator.Int64{int64validator.Between(1, 50)},
			},
			"require_alphanumeric_passcode": schema.BoolAttribute{
				Description: "Require alphanumeric passcode.",
				Optional:    true,
			},
			"require_complex_passcode": schema.BoolAttribute{
				Description: "Require complex passcode.",
				Optional:    true,
			},
			"require_passcode": schema.BoolAttribute{
				Description: "Require passcode.",
				Optional:    true,
			},
		},
	}
}

// ToRawConfiguration converts the typed component to raw configuration matching OpenAPI PasscodePolicyConfiguration schema
func (c *PasscodePolicyComponent) ToRawConfiguration() (map[string]interface{}, error) {
	config := make(map[string]interface{})

	if !c.ChangeAtNextAuth.IsNull() && !c.ChangeAtNextAuth.IsUnknown() {
		config["ChangeAtNextAuth"] = c.ChangeAtNextAuth.ValueBool()
	}

	if !c.FailedAttemptsResetInMinutes.IsNull() && !c.FailedAttemptsResetInMinutes.IsUnknown() {
		config["FailedAttemptsResetInMinutes"] = int(c.FailedAttemptsResetInMinutes.ValueInt64())
	}

	if !c.MaximumFailedAttempts.IsNull() && !c.MaximumFailedAttempts.IsUnknown() {
		config["MaximumFailedAttempts"] = int(c.MaximumFailedAttempts.ValueInt64())
	}

	if !c.MaximumGracePeriodInMinutes.IsNull() && !c.MaximumGracePeriodInMinutes.IsUnknown() {
		config["MaximumGracePeriodInMinutes"] = int(c.MaximumGracePeriodInMinutes.ValueInt64())
	}

	if !c.MaximumInactivityInMinutes.IsNull() && !c.MaximumInactivityInMinutes.IsUnknown() {
		config["MaximumInactivityInMinutes"] = int(c.MaximumInactivityInMinutes.ValueInt64())
	}

	if !c.MaximumPasscodeAgeInDays.IsNull() && !c.MaximumPasscodeAgeInDays.IsUnknown() {
		config["MaximumPasscodeAgeInDays"] = int(c.MaximumPasscodeAgeInDays.ValueInt64())
	}

	if !c.MinimumComplexCharacters.IsNull() && !c.MinimumComplexCharacters.IsUnknown() {
		config["MinimumComplexCharacters"] = int(c.MinimumComplexCharacters.ValueInt64())
	}

	if !c.MinimumLength.IsNull() && !c.MinimumLength.IsUnknown() {
		config["MinimumLength"] = int(c.MinimumLength.ValueInt64())
	}

	if !c.PasscodeReuseLimit.IsNull() && !c.PasscodeReuseLimit.IsUnknown() {
		config["PasscodeReuseLimit"] = int(c.PasscodeReuseLimit.ValueInt64())
	}

	if !c.RequireAlphanumericPasscode.IsNull() && !c.RequireAlphanumericPasscode.IsUnknown() {
		config["RequireAlphanumericPasscode"] = c.RequireAlphanumericPasscode.ValueBool()
	}

	if !c.RequireComplexPasscode.IsNull() && !c.RequireComplexPasscode.IsUnknown() {
		config["RequireComplexPasscode"] = c.RequireComplexPasscode.ValueBool()
	}

	if !c.RequirePasscode.IsNull() && !c.RequirePasscode.IsUnknown() {
		config["RequirePasscode"] = c.RequirePasscode.ValueBool()
	}

	return config, nil
}

// FromRawConfiguration populates the typed component from raw configuration data
func (c *PasscodePolicyComponent) FromRawConfiguration(raw map[string]interface{}) error {
	if changeAtNextAuth, exists := raw["ChangeAtNextAuth"]; exists {
		if changeAtNextAuthBool, ok := changeAtNextAuth.(bool); ok {
			c.ChangeAtNextAuth = types.BoolValue(changeAtNextAuthBool)
		}
	}

	if failedAttemptsReset, exists := raw["FailedAttemptsResetInMinutes"]; exists {
		if failedAttemptsResetFloat, ok := failedAttemptsReset.(float64); ok {
			c.FailedAttemptsResetInMinutes = types.Int64Value(int64(failedAttemptsResetFloat))
		} else if failedAttemptsResetInt, ok := failedAttemptsReset.(int); ok {
			c.FailedAttemptsResetInMinutes = types.Int64Value(int64(failedAttemptsResetInt))
		}
	}

	if maximumFailedAttempts, exists := raw["MaximumFailedAttempts"]; exists {
		if maximumFailedAttemptsFloat, ok := maximumFailedAttempts.(float64); ok {
			c.MaximumFailedAttempts = types.Int64Value(int64(maximumFailedAttemptsFloat))
		} else if maximumFailedAttemptsInt, ok := maximumFailedAttempts.(int); ok {
			c.MaximumFailedAttempts = types.Int64Value(int64(maximumFailedAttemptsInt))
		}
	}

	if maximumGracePeriod, exists := raw["MaximumGracePeriodInMinutes"]; exists {
		if maximumGracePeriodFloat, ok := maximumGracePeriod.(float64); ok {
			c.MaximumGracePeriodInMinutes = types.Int64Value(int64(maximumGracePeriodFloat))
		} else if maximumGracePeriodInt, ok := maximumGracePeriod.(int); ok {
			c.MaximumGracePeriodInMinutes = types.Int64Value(int64(maximumGracePeriodInt))
		}
	}

	if maximumInactivity, exists := raw["MaximumInactivityInMinutes"]; exists {
		if maximumInactivityFloat, ok := maximumInactivity.(float64); ok {
			c.MaximumInactivityInMinutes = types.Int64Value(int64(maximumInactivityFloat))
		} else if maximumInactivityInt, ok := maximumInactivity.(int); ok {
			c.MaximumInactivityInMinutes = types.Int64Value(int64(maximumInactivityInt))
		}
	}

	if maximumPasscodeAge, exists := raw["MaximumPasscodeAgeInDays"]; exists {
		if maximumPasscodeAgeFloat, ok := maximumPasscodeAge.(float64); ok {
			c.MaximumPasscodeAgeInDays = types.Int64Value(int64(maximumPasscodeAgeFloat))
		} else if maximumPasscodeAgeInt, ok := maximumPasscodeAge.(int); ok {
			c.MaximumPasscodeAgeInDays = types.Int64Value(int64(maximumPasscodeAgeInt))
		}
	}

	if minimumComplexCharacters, exists := raw["MinimumComplexCharacters"]; exists {
		if minimumComplexCharactersFloat, ok := minimumComplexCharacters.(float64); ok {
			c.MinimumComplexCharacters = types.Int64Value(int64(minimumComplexCharactersFloat))
		} else if minimumComplexCharactersInt, ok := minimumComplexCharacters.(int); ok {
			c.MinimumComplexCharacters = types.Int64Value(int64(minimumComplexCharactersInt))
		}
	}

	if minimumLength, exists := raw["MinimumLength"]; exists {
		if minimumLengthFloat, ok := minimumLength.(float64); ok {
			c.MinimumLength = types.Int64Value(int64(minimumLengthFloat))
		} else if minimumLengthInt, ok := minimumLength.(int); ok {
			c.MinimumLength = types.Int64Value(int64(minimumLengthInt))
		}
	}

	if passcodeReuseLimit, exists := raw["PasscodeReuseLimit"]; exists {
		if passcodeReuseLimitFloat, ok := passcodeReuseLimit.(float64); ok {
			c.PasscodeReuseLimit = types.Int64Value(int64(passcodeReuseLimitFloat))
		} else if passcodeReuseLimitInt, ok := passcodeReuseLimit.(int); ok {
			c.PasscodeReuseLimit = types.Int64Value(int64(passcodeReuseLimitInt))
		}
	}

	if requireAlphanumeric, exists := raw["RequireAlphanumericPasscode"]; exists {
		if requireAlphanumericBool, ok := requireAlphanumeric.(bool); ok {
			c.RequireAlphanumericPasscode = types.BoolValue(requireAlphanumericBool)
		}
	}

	if requireComplex, exists := raw["RequireComplexPasscode"]; exists {
		if requireComplexBool, ok := requireComplex.(bool); ok {
			c.RequireComplexPasscode = types.BoolValue(requireComplexBool)
		}
	}

	if requirePasscode, exists := raw["RequirePasscode"]; exists {
		if requirePasscodeBool, ok := requirePasscode.(bool); ok {
			c.RequirePasscode = types.BoolValue(requirePasscodeBool)
		}
	}

	return nil
}

// ToClientComponent converts the typed component to the format expected by the Blueprint API client
func (c *PasscodePolicyComponent) ToClientComponent() (*BlueprintComponentData, error) {
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
