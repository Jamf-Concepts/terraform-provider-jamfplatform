// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"context"
	"encoding/json"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/blueprints"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/bpcomponents/declarations"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// PasscodePolicyComponent represents a strongly-typed passcode policy component.
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

// GetIdentifier returns the component identifier for passcode policy.
func (c *PasscodePolicyComponent) GetIdentifier() string {
	return "com.jamf.ddm.passcode-settings"
}

// PasscodePolicyComponentSchema returns the Terraform schema for passcode policy component.
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

// boolPtrFromBool converts a configured types.Bool field to a *bool pointer with Included envelope.
func buildChangeAtNextAuth(field types.Bool) *declarations.ChangeAtNextAuth {
	if !helpers.IsConfiguredValue(field) {
		return nil
	}
	v := field.ValueBool()
	t := true
	return &declarations.ChangeAtNextAuth{Included: &t, Value: &v}
}

// buildRequireBool builds a bool wrapper struct with Included envelope.
func buildRequireAlphanumeric(field types.Bool) *declarations.RequireAlphanumericPasscode {
	if !helpers.IsConfiguredValue(field) {
		return nil
	}
	v := field.ValueBool()
	t := true
	return &declarations.RequireAlphanumericPasscode{Included: &t, Value: &v}
}

// buildRequireComplex builds a require complex passcode wrapper with Included envelope.
func buildRequireComplex(field types.Bool) *declarations.RequireComplexPasscode {
	if !helpers.IsConfiguredValue(field) {
		return nil
	}
	v := field.ValueBool()
	t := true
	return &declarations.RequireComplexPasscode{Included: &t, Value: &v}
}

// buildRequirePasscode builds a require passcode wrapper with Included envelope.
func buildRequirePasscode(field types.Bool) *declarations.RequirePasscode {
	if !helpers.IsConfiguredValue(field) {
		return nil
	}
	v := field.ValueBool()
	t := true
	return &declarations.RequirePasscode{Included: &t, Value: &v}
}

// buildFailedAttemptsReset builds a FailedAttemptsResetInMinutes wrapper.
func buildFailedAttemptsReset(field types.Int64) *declarations.FailedAttemptsResetInMinutes {
	if !helpers.IsConfiguredValue(field) {
		return nil
	}
	v := int(field.ValueInt64())
	t := true
	return &declarations.FailedAttemptsResetInMinutes{Included: &t, Value: &v}
}

// buildMaximumFailedAttempts builds a MaximumFailedAttempts wrapper.
func buildMaximumFailedAttempts(field types.Int64) *declarations.MaximumFailedAttempts {
	if !helpers.IsConfiguredValue(field) {
		return nil
	}
	v := int(field.ValueInt64())
	t := true
	return &declarations.MaximumFailedAttempts{Included: &t, Value: &v}
}

// buildMaximumGracePeriod builds a MaximumGracePeriodInMinutes wrapper.
func buildMaximumGracePeriod(field types.Int64) *declarations.MaximumGracePeriodInMinutes {
	if !helpers.IsConfiguredValue(field) {
		return nil
	}
	v := int(field.ValueInt64())
	t := true
	return &declarations.MaximumGracePeriodInMinutes{Included: &t, Value: &v}
}

// buildMaximumInactivity builds a MaximumInactivityInMinutes wrapper.
func buildMaximumInactivity(field types.Int64) *declarations.MaximumInactivityInMinutes {
	if !helpers.IsConfiguredValue(field) {
		return nil
	}
	v := int(field.ValueInt64())
	t := true
	return &declarations.MaximumInactivityInMinutes{Included: &t, Value: &v}
}

// buildMaximumPasscodeAge builds a MaximumPasscodeAgeInDays wrapper.
func buildMaximumPasscodeAge(field types.Int64) *declarations.MaximumPasscodeAgeInDays {
	if !helpers.IsConfiguredValue(field) {
		return nil
	}
	v := int(field.ValueInt64())
	t := true
	return &declarations.MaximumPasscodeAgeInDays{Included: &t, Value: &v}
}

// buildMinimumComplexChars builds a MinimumComplexCharacters wrapper.
func buildMinimumComplexChars(field types.Int64) *declarations.MinimumComplexCharacters {
	if !helpers.IsConfiguredValue(field) {
		return nil
	}
	v := int(field.ValueInt64())
	t := true
	return &declarations.MinimumComplexCharacters{Included: &t, Value: &v}
}

// buildMinimumLength builds a MinimumLength wrapper.
func buildMinimumLength(field types.Int64) *declarations.MinimumLength {
	if !helpers.IsConfiguredValue(field) {
		return nil
	}
	v := int(field.ValueInt64())
	t := true
	return &declarations.MinimumLength{Included: &t, Value: &v}
}

// buildPasscodeReuseLimit builds a PasscodeReuseLimit wrapper.
func buildPasscodeReuseLimit(field types.Int64) *declarations.PasscodeReuseLimit {
	if !helpers.IsConfiguredValue(field) {
		return nil
	}
	v := int(field.ValueInt64())
	t := true
	return &declarations.PasscodeReuseLimit{Included: &t, Value: &v}
}

// ToRawConfiguration converts the typed component to raw JSON configuration.
func (c *PasscodePolicyComponent) ToRawConfiguration() (json.RawMessage, error) {
	cfg := declarations.PasscodeSettingsConfigurationV2{Version: 2}

	cfg.ChangeAtNextAuth = buildChangeAtNextAuth(c.ChangeAtNextAuth)
	cfg.FailedAttemptsResetInMinutes = buildFailedAttemptsReset(c.FailedAttemptsResetInMinutes)
	cfg.MaximumFailedAttempts = buildMaximumFailedAttempts(c.MaximumFailedAttempts)
	cfg.MaximumGracePeriodInMinutes = buildMaximumGracePeriod(c.MaximumGracePeriodInMinutes)
	cfg.MaximumInactivityInMinutes = buildMaximumInactivity(c.MaximumInactivityInMinutes)
	cfg.MaximumPasscodeAgeInDays = buildMaximumPasscodeAge(c.MaximumPasscodeAgeInDays)
	cfg.MinimumComplexCharacters = buildMinimumComplexChars(c.MinimumComplexCharacters)
	cfg.MinimumLength = buildMinimumLength(c.MinimumLength)
	cfg.PasscodeReuseLimit = buildPasscodeReuseLimit(c.PasscodeReuseLimit)
	cfg.RequireAlphanumericPasscode = buildRequireAlphanumeric(c.RequireAlphanumericPasscode)
	cfg.RequireComplexPasscode = buildRequireComplex(c.RequireComplexPasscode)
	cfg.RequirePasscode = buildRequirePasscode(c.RequirePasscode)

	hasCustomRegex := helpers.IsConfiguredValue(c.CustomRegexPattern) ||
		(!c.CustomRegexDescription.IsNull() && !c.CustomRegexDescription.IsUnknown())

	if hasCustomRegex {
		trueVal := true
		customRegex := &declarations.CustomRegex{
			Included: &trueVal,
		}
		if helpers.IsConfiguredValue(c.CustomRegexPattern) {
			pattern := c.CustomRegexPattern.ValueString()
			customRegex.Regex = &pattern
		}
		if !c.CustomRegexDescription.IsNull() && !c.CustomRegexDescription.IsUnknown() {
			descMap := make(map[string]string)
			for k, v := range c.CustomRegexDescription.Elements() {
				if strVal, ok := v.(types.String); ok {
					descMap[k] = strVal.ValueString()
				}
			}
			customRegex.Description = &descMap
		}
		cfg.CustomRegex = customRegex
	}

	return json.Marshal(cfg)
}

// FromRawConfiguration populates the typed component from raw JSON configuration.
// Handles both V2 envelope format ({Value, Included}) and V1 flat format for backwards compatibility.
func (c *PasscodePolicyComponent) FromRawConfiguration(raw json.RawMessage) error {
	c.ChangeAtNextAuth = types.BoolNull()
	c.FailedAttemptsResetInMinutes = types.Int64Null()
	c.MaximumFailedAttempts = types.Int64Null()
	c.MaximumGracePeriodInMinutes = types.Int64Null()
	c.MaximumInactivityInMinutes = types.Int64Null()
	c.MaximumPasscodeAgeInDays = types.Int64Null()
	c.MinimumComplexCharacters = types.Int64Null()
	c.MinimumLength = types.Int64Null()
	c.PasscodeReuseLimit = types.Int64Null()
	c.RequireAlphanumericPasscode = types.BoolNull()
	c.RequireComplexPasscode = types.BoolNull()
	c.RequirePasscode = types.BoolNull()
	c.CustomRegexPattern = types.StringNull()
	c.CustomRegexDescription = types.MapNull(types.StringType)

	var cfg declarations.PasscodeSettingsConfigurationV2
	if err := json.Unmarshal(raw, &cfg); err != nil {
		var v1 declarations.PasscodeSettingsConfigurationV1
		if err2 := json.Unmarshal(raw, &v1); err2 != nil {
			return err
		}
		return c.fromV1Config(v1)
	}

	if cfg.Version == 0 {
		var v1 declarations.PasscodeSettingsConfigurationV1
		if err := json.Unmarshal(raw, &v1); err == nil && v1.RequirePasscode != nil {
			return c.fromV1Config(v1)
		}
	}

	if f := cfg.ChangeAtNextAuth; f != nil && f.Included != nil && *f.Included && f.Value != nil {
		c.ChangeAtNextAuth = types.BoolValue(*f.Value)
	}
	if f := cfg.FailedAttemptsResetInMinutes; f != nil && f.Included != nil && *f.Included && f.Value != nil {
		c.FailedAttemptsResetInMinutes = types.Int64Value(int64(*f.Value))
	}
	if f := cfg.MaximumFailedAttempts; f != nil && f.Included != nil && *f.Included && f.Value != nil {
		c.MaximumFailedAttempts = types.Int64Value(int64(*f.Value))
	}
	if f := cfg.MaximumGracePeriodInMinutes; f != nil && f.Included != nil && *f.Included && f.Value != nil {
		c.MaximumGracePeriodInMinutes = types.Int64Value(int64(*f.Value))
	}
	if f := cfg.MaximumInactivityInMinutes; f != nil && f.Included != nil && *f.Included && f.Value != nil {
		c.MaximumInactivityInMinutes = types.Int64Value(int64(*f.Value))
	}
	if f := cfg.MaximumPasscodeAgeInDays; f != nil && f.Included != nil && *f.Included && f.Value != nil {
		c.MaximumPasscodeAgeInDays = types.Int64Value(int64(*f.Value))
	}
	if f := cfg.MinimumComplexCharacters; f != nil && f.Included != nil && *f.Included && f.Value != nil {
		c.MinimumComplexCharacters = types.Int64Value(int64(*f.Value))
	}
	if f := cfg.MinimumLength; f != nil && f.Included != nil && *f.Included && f.Value != nil {
		c.MinimumLength = types.Int64Value(int64(*f.Value))
	}
	if f := cfg.PasscodeReuseLimit; f != nil && f.Included != nil && *f.Included && f.Value != nil {
		c.PasscodeReuseLimit = types.Int64Value(int64(*f.Value))
	}
	if f := cfg.RequireAlphanumericPasscode; f != nil && f.Included != nil && *f.Included && f.Value != nil {
		c.RequireAlphanumericPasscode = types.BoolValue(*f.Value)
	}
	if f := cfg.RequireComplexPasscode; f != nil && f.Included != nil && *f.Included && f.Value != nil {
		c.RequireComplexPasscode = types.BoolValue(*f.Value)
	}
	if f := cfg.RequirePasscode; f != nil && f.Included != nil && *f.Included && f.Value != nil {
		c.RequirePasscode = types.BoolValue(*f.Value)
	}

	if cr := cfg.CustomRegex; cr != nil {
		if cr.Included == nil || *cr.Included {
			if cr.Regex != nil {
				c.CustomRegexPattern = types.StringValue(*cr.Regex)
			}
			if cr.Description != nil {
				elems := make(map[string]types.String, len(*cr.Description))
				for k, v := range *cr.Description {
					elems[k] = types.StringValue(v)
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

	return nil
}

// fromV1Config populates the component from a V1 flat passcode configuration.
func (c *PasscodePolicyComponent) fromV1Config(v1 declarations.PasscodeSettingsConfigurationV1) error {
	if v1.ChangeAtNextAuth != nil {
		c.ChangeAtNextAuth = types.BoolValue(*v1.ChangeAtNextAuth)
	}
	if v1.FailedAttemptsResetInMinutes != nil {
		c.FailedAttemptsResetInMinutes = types.Int64Value(int64(*v1.FailedAttemptsResetInMinutes))
	}
	if v1.MaximumFailedAttempts != nil {
		c.MaximumFailedAttempts = types.Int64Value(int64(*v1.MaximumFailedAttempts))
	}
	if v1.MaximumGracePeriodInMinutes != nil {
		c.MaximumGracePeriodInMinutes = types.Int64Value(int64(*v1.MaximumGracePeriodInMinutes))
	}
	if v1.MaximumInactivityInMinutes != nil {
		c.MaximumInactivityInMinutes = types.Int64Value(int64(*v1.MaximumInactivityInMinutes))
	}
	if v1.MaximumPasscodeAgeInDays != nil {
		c.MaximumPasscodeAgeInDays = types.Int64Value(int64(*v1.MaximumPasscodeAgeInDays))
	}
	if v1.MinimumComplexCharacters != nil {
		c.MinimumComplexCharacters = types.Int64Value(int64(*v1.MinimumComplexCharacters))
	}
	if v1.MinimumLength != nil {
		c.MinimumLength = types.Int64Value(int64(*v1.MinimumLength))
	}
	if v1.PasscodeReuseLimit != nil {
		c.PasscodeReuseLimit = types.Int64Value(int64(*v1.PasscodeReuseLimit))
	}
	if v1.RequireAlphanumericPasscode != nil {
		c.RequireAlphanumericPasscode = types.BoolValue(*v1.RequireAlphanumericPasscode)
	}
	if v1.RequireComplexPasscode != nil {
		c.RequireComplexPasscode = types.BoolValue(*v1.RequireComplexPasscode)
	}
	if v1.RequirePasscode != nil {
		c.RequirePasscode = types.BoolValue(*v1.RequirePasscode)
	}
	if cr := v1.CustomRegex; cr != nil {
		if cr.Included == nil || *cr.Included {
			if cr.Regex != nil {
				c.CustomRegexPattern = types.StringValue(*cr.Regex)
			}
			if cr.Description != nil {
				elems := make(map[string]types.String, len(*cr.Description))
				for k, v := range *cr.Description {
					elems[k] = types.StringValue(v)
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
	return nil
}

// ToClientComponent converts the typed component to the format expected by the Blueprint API client.
func (c *PasscodePolicyComponent) ToClientComponent() (*blueprints.Component, error) {
	cfg, err := c.ToRawConfiguration()
	if err != nil {
		return nil, err
	}
	return &blueprints.Component{
		Identifier:    c.GetIdentifier(),
		Configuration: cfg,
	}, nil
}
