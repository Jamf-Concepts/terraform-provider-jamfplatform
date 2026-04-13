// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"context"
	"encoding/json"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// PasscodePolicyComponent represents a strongly-typed passcode policy component
type PasscodePolicyComponent struct {
	ChangeAtNextAuth             types.Bool   `tfsdk:"change_at_next_auth"`
	CustomRegexPattern           types.String `tfsdk:"custom_regex_pattern"`
	CustomRegexDescription       types.Map    `tfsdk:"custom_regex_description"`
	FailedAttemptsResetInMinutes types.Int64  `tfsdk:"failed_attempts_reset_in_minutes"`
	MaximumFailedAttempts        types.Int64  `tfsdk:"maximum_failed_attempts"`
	MaximumGracePeriodInMinutes  types.Int64  `tfsdk:"maximum_grace_period_in_minutes"`
	MaximumInactivityInMinutes   types.Int64  `tfsdk:"maximum_inactivity_in_minutes"`
	MaximumPasscodeAgeInDays     types.Int64  `tfsdk:"maximum_passcode_age_in_days"`
	MinimumComplexCharacters     types.Int64  `tfsdk:"minimum_complex_characters"`
	MinimumLength                types.Int64  `tfsdk:"minimum_length"`
	PasscodeReuseLimit           types.Int64  `tfsdk:"passcode_reuse_limit"`
	RequireAlphanumericPasscode  types.Bool   `tfsdk:"require_alphanumeric_passcode"`
	RequireComplexPasscode       types.Bool   `tfsdk:"require_complex_passcode"`
	RequirePasscode              types.Bool   `tfsdk:"require_passcode"`
}

// GetIdentifier returns the component identifier for passcode policy
func (c *PasscodePolicyComponent) GetIdentifier() string {
	return "com.jamf.ddm.passcode-settings"
}

// PasscodePolicyComponentSchema returns the Terraform schema for passcode policy component
func PasscodePolicyComponentSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"change_at_next_auth": schema.BoolAttribute{
			MarkdownDescription: "Change at next auth.",
			Optional:            true,
		},
		"custom_regex_pattern": schema.StringAttribute{
			MarkdownDescription: "Custom regular expression for passcode validation. Maximum length: `2048`.",
			Optional:            true,
			Validators:          []validator.String{stringvalidator.LengthAtMost(2048)},
		},
		"custom_regex_description": schema.MapAttribute{
			MarkdownDescription: "Localized descriptions for the custom regex. Map of OS language ID to description string. Use the `default` key for languages not explicitly listed.",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"failed_attempts_reset_in_minutes": schema.Int64Attribute{
			MarkdownDescription: "Failed attempts reset in minutes. Minimum: `0`.",
			Optional:            true,
		},
		"maximum_failed_attempts": schema.Int64Attribute{
			MarkdownDescription: "Maximum failed attempts. Range: `2`-`11`.",
			Optional:            true,
			Validators:          []validator.Int64{int64validator.Between(2, 11)},
		},
		"maximum_grace_period_in_minutes": schema.Int64Attribute{
			MarkdownDescription: "Maximum grace period in minutes. Minimum: `0`.",
			Optional:            true,
		},
		"maximum_inactivity_in_minutes": schema.Int64Attribute{
			MarkdownDescription: "Maximum inactivity in minutes. Range: `0`-`15`.",
			Optional:            true,
			Validators:          []validator.Int64{int64validator.Between(0, 15)},
		},
		"maximum_passcode_age_in_days": schema.Int64Attribute{
			MarkdownDescription: "Maximum passcode age in days. Range: `0`-`730`.",
			Optional:            true,
			Validators:          []validator.Int64{int64validator.Between(0, 730)},
		},
		"minimum_complex_characters": schema.Int64Attribute{
			MarkdownDescription: "Minimum complex characters. Range: `0`-`4`.",
			Optional:            true,
			Validators:          []validator.Int64{int64validator.Between(0, 4)},
		},
		"minimum_length": schema.Int64Attribute{
			MarkdownDescription: "Minimum length. Range: `0`-`16`.",
			Optional:            true,
			Validators:          []validator.Int64{int64validator.Between(0, 16)},
		},
		"passcode_reuse_limit": schema.Int64Attribute{
			MarkdownDescription: "Passcode reuse limit. Range: `1`-`50`.",
			Optional:            true,
			Validators:          []validator.Int64{int64validator.Between(1, 50)},
		},
		"require_alphanumeric_passcode": schema.BoolAttribute{
			MarkdownDescription: "Require alphanumeric passcode.",
			Optional:            true,
		},
		"require_complex_passcode": schema.BoolAttribute{
			MarkdownDescription: "Require complex passcode.",
			Optional:            true,
		},
		"require_passcode": schema.BoolAttribute{
			MarkdownDescription: "Require passcode.",
			Optional:            true,
		},
	}
}

// ToRawConfiguration converts the typed component to raw configuration matching OpenAPI PasscodeSettingsConfigurationV2 schema
func (c *PasscodePolicyComponent) ToRawConfiguration() (map[string]any, error) {
	config := make(map[string]any)

	config["version"] = "2"

	if helpers.IsConfiguredValue(c.ChangeAtNextAuth) {
		config["ChangeAtNextAuth"] = setBoolFieldWithKey(c.ChangeAtNextAuth, "Value", false)
	}
	if helpers.IsConfiguredValue(c.FailedAttemptsResetInMinutes) {
		config["FailedAttemptsResetInMinutes"] = setInt64Field(c.FailedAttemptsResetInMinutes, 0)
	}
	if helpers.IsConfiguredValue(c.MaximumFailedAttempts) {
		config["MaximumFailedAttempts"] = setInt64Field(c.MaximumFailedAttempts, 0)
	}
	if helpers.IsConfiguredValue(c.MaximumGracePeriodInMinutes) {
		config["MaximumGracePeriodInMinutes"] = setInt64Field(c.MaximumGracePeriodInMinutes, 0)
	}
	if helpers.IsConfiguredValue(c.MaximumInactivityInMinutes) {
		config["MaximumInactivityInMinutes"] = setInt64Field(c.MaximumInactivityInMinutes, 0)
	}
	if helpers.IsConfiguredValue(c.MaximumPasscodeAgeInDays) {
		config["MaximumPasscodeAgeInDays"] = setInt64Field(c.MaximumPasscodeAgeInDays, 0)
	}
	if helpers.IsConfiguredValue(c.MinimumComplexCharacters) {
		config["MinimumComplexCharacters"] = setInt64Field(c.MinimumComplexCharacters, 0)
	}
	if helpers.IsConfiguredValue(c.MinimumLength) {
		config["MinimumLength"] = setInt64Field(c.MinimumLength, 0)
	}
	if helpers.IsConfiguredValue(c.PasscodeReuseLimit) {
		config["PasscodeReuseLimit"] = setInt64Field(c.PasscodeReuseLimit, 0)
	}
	if helpers.IsConfiguredValue(c.RequireAlphanumericPasscode) {
		config["RequireAlphanumericPasscode"] = setBoolFieldWithKey(c.RequireAlphanumericPasscode, "Value", false)
	}
	if helpers.IsConfiguredValue(c.RequireComplexPasscode) {
		config["RequireComplexPasscode"] = setBoolFieldWithKey(c.RequireComplexPasscode, "Value", false)
	}
	if helpers.IsConfiguredValue(c.RequirePasscode) {
		config["RequirePasscode"] = setBoolFieldWithKey(c.RequirePasscode, "Value", false)
	}

	hasCustomRegex := helpers.IsConfiguredValue(c.CustomRegexPattern) ||
		(!c.CustomRegexDescription.IsNull() && !c.CustomRegexDescription.IsUnknown())

	if hasCustomRegex {
		customRegex := map[string]any{
			"Included": true,
		}
		if helpers.IsConfiguredValue(c.CustomRegexPattern) {
			customRegex["Regex"] = c.CustomRegexPattern.ValueString()
		}
		if !c.CustomRegexDescription.IsNull() && !c.CustomRegexDescription.IsUnknown() {
			descMap := make(map[string]string)
			for k, v := range c.CustomRegexDescription.Elements() {
				if strVal, ok := v.(types.String); ok {
					descMap[k] = strVal.ValueString()
				}
			}
			customRegex["Description"] = descMap
		}
		config["CustomRegex"] = customRegex
	}

	return config, nil
}

// FromRawConfiguration populates the typed component from raw configuration data.
// Handles both V2 envelope format ({Value, Included}) and V1 flat format for backwards compatibility.
func (c *PasscodePolicyComponent) FromRawConfiguration(raw map[string]any) error {
	extractBoolValue := func(key string) types.Bool {
		obj, exists := raw[key]
		if !exists {
			return types.BoolNull()
		}
		if boolVal, ok := obj.(bool); ok {
			return types.BoolValue(boolVal)
		}
		if objMap, ok := obj.(map[string]any); ok {
			if included, hasIncluded := objMap["Included"]; hasIncluded && included.(bool) {
				if value, hasValue := objMap["Value"]; hasValue {
					if boolVal, ok := value.(bool); ok {
						return types.BoolValue(boolVal)
					}
				}
			}
		}
		return types.BoolNull()
	}

	extractInt64Value := func(key string) types.Int64 {
		obj, exists := raw[key]
		if !exists {
			return types.Int64Null()
		}
		switch v := obj.(type) {
		case float64:
			return types.Int64Value(int64(v))
		case int:
			return types.Int64Value(int64(v))
		case int64:
			return types.Int64Value(v)
		}
		if objMap, ok := obj.(map[string]any); ok {
			if included, hasIncluded := objMap["Included"]; hasIncluded && included.(bool) {
				if value, hasValue := objMap["Value"]; hasValue {
					switch v := value.(type) {
					case float64:
						return types.Int64Value(int64(v))
					case int:
						return types.Int64Value(int64(v))
					case int64:
						return types.Int64Value(v)
					}
				}
			}
		}
		return types.Int64Null()
	}

	c.ChangeAtNextAuth = extractBoolValue("ChangeAtNextAuth")
	c.FailedAttemptsResetInMinutes = extractInt64Value("FailedAttemptsResetInMinutes")
	c.MaximumFailedAttempts = extractInt64Value("MaximumFailedAttempts")
	c.MaximumGracePeriodInMinutes = extractInt64Value("MaximumGracePeriodInMinutes")
	c.MaximumInactivityInMinutes = extractInt64Value("MaximumInactivityInMinutes")
	c.MaximumPasscodeAgeInDays = extractInt64Value("MaximumPasscodeAgeInDays")
	c.MinimumComplexCharacters = extractInt64Value("MinimumComplexCharacters")
	c.MinimumLength = extractInt64Value("MinimumLength")
	c.PasscodeReuseLimit = extractInt64Value("PasscodeReuseLimit")
	c.RequireAlphanumericPasscode = extractBoolValue("RequireAlphanumericPasscode")
	c.RequireComplexPasscode = extractBoolValue("RequireComplexPasscode")
	c.RequirePasscode = extractBoolValue("RequirePasscode")

	c.CustomRegexPattern = types.StringNull()
	c.CustomRegexDescription = types.MapNull(types.StringType)

	if customRegexRaw, exists := raw["CustomRegex"]; exists {
		if customRegexMap, ok := customRegexRaw.(map[string]any); ok {
			if included, hasIncluded := customRegexMap["Included"]; !hasIncluded || included.(bool) {
				if regex, hasRegex := customRegexMap["Regex"]; hasRegex {
					if regexStr, ok := regex.(string); ok {
						c.CustomRegexPattern = types.StringValue(regexStr)
					}
				}
				if descRaw, hasDesc := customRegexMap["Description"]; hasDesc {
					if descMap, ok := descRaw.(map[string]any); ok {
						elems := make(map[string]types.String, len(descMap))
						for k, v := range descMap {
							if vStr, ok := v.(string); ok {
								elems[k] = types.StringValue(vStr)
							}
						}
						if len(elems) > 0 {
							tfMap, diags := types.MapValueFrom(context.Background(), types.StringType, elems)
							if !diags.HasError() {
								c.CustomRegexDescription = tfMap
							}
						}
					}
				}
			}
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
