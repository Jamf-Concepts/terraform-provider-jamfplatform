// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"encoding/json"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SoftwareUpdateSettingsComponent represents a strongly-typed software update settings component
type SoftwareUpdateSettingsComponent struct {
	AllowStandardUserOSUpdates           types.Bool         `tfsdk:"allow_standard_user_os_updates"`
	AutomaticDownload                    types.String       `tfsdk:"automatic_download"`
	AutomaticInstallOSUpdates            types.String       `tfsdk:"automatic_install_os_updates"`
	AutomaticInstallSecurityUpdate       types.String       `tfsdk:"automatic_install_security_updates"`
	BetaProgramEnrollment                types.String       `tfsdk:"beta_program_enrollment"`
	BetaOfferPrograms                    []BetaProgramModel `tfsdk:"beta_offer_programs"`
	BetaRequireProgramToken              types.String       `tfsdk:"beta_require_program_token"`
	BetaRequireProgramDescription        types.String       `tfsdk:"beta_require_program_description"`
	DeferralCombinedPeriod               types.Int64        `tfsdk:"deferral_combined_period_days"`
	DeferralMajorPeriod                  types.Int64        `tfsdk:"deferral_major_period_days"`
	DeferralMinorPeriod                  types.Int64        `tfsdk:"deferral_minor_period_days"`
	DeferralSystemPeriod                 types.Int64        `tfsdk:"deferral_system_period_days"`
	NotificationsEnabled                 types.Bool         `tfsdk:"notifications_enabled"`
	RapidSecurityResponseEnabled         types.Bool         `tfsdk:"rapid_security_response_enabled"`
	RapidSecurityResponseRollbackEnabled types.Bool         `tfsdk:"rapid_security_response_rollback_enabled"`
	RecommendedCadence                   types.String       `tfsdk:"recommended_cadence"`
}

// BetaProgramModel represents a beta program configuration
type BetaProgramModel struct {
	Token       types.String `tfsdk:"token"`
	Description types.String `tfsdk:"description"`
}

// SoftwareUpdateSettingsComponentSchema returns the Terraform schema for software update component
func SoftwareUpdateSettingsComponentSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"allow_standard_user_os_updates": schema.BoolAttribute{
			MarkdownDescription: "Allow standard users to install OS updates without administrator privileges.",
			Optional:            true,
		},
		"automatic_download": schema.StringAttribute{
			MarkdownDescription: "Automatic download behavior for updates. Valid values: `Allowed`, `AlwaysOn`, `AlwaysOff`.",
			Optional:            true,
			Validators:          []validator.String{stringvalidator.OneOf("Allowed", "AlwaysOn", "AlwaysOff")},
		},
		"automatic_install_os_updates": schema.StringAttribute{
			MarkdownDescription: "Automatic installation behavior for OS updates. Valid values: `Allowed`, `AlwaysOn`, `AlwaysOff`.",
			Optional:            true,
			Validators:          []validator.String{stringvalidator.OneOf("Allowed", "AlwaysOn", "AlwaysOff")},
		},
		"automatic_install_security_updates": schema.StringAttribute{
			MarkdownDescription: "Automatic installation behavior for security updates. Valid values: `Allowed`, `AlwaysOn`, `AlwaysOff`.",
			Optional:            true,
			Validators:          []validator.String{stringvalidator.OneOf("Allowed", "AlwaysOn", "AlwaysOff")},
		},
		"beta_program_enrollment": schema.StringAttribute{
			MarkdownDescription: "Beta program enrollment setting. Valid values: `Allowed`, `AlwaysOn`, `AlwaysOff`.",
			Optional:            true,
			Validators:          []validator.String{stringvalidator.OneOf("Allowed", "AlwaysOn", "AlwaysOff")},
		},
		"deferral_combined_period_days": schema.Int64Attribute{
			MarkdownDescription: "Number of days to defer combined updates. Range: `1`-`90`.",
			Optional:            true,
			Validators:          []validator.Int64{int64validator.Between(1, 90)},
		},
		"deferral_major_period_days": schema.Int64Attribute{
			MarkdownDescription: "Number of days to defer major updates. Range: `1`-`90`.",
			Optional:            true,
			Validators:          []validator.Int64{int64validator.Between(1, 90)},
		},
		"deferral_minor_period_days": schema.Int64Attribute{
			MarkdownDescription: "Number of days to defer minor updates. Range: `1`-`90`.",
			Optional:            true,
			Validators:          []validator.Int64{int64validator.Between(1, 90)},
		},
		"deferral_system_period_days": schema.Int64Attribute{
			MarkdownDescription: "Number of days to defer system updates. Range: `1`-`90`.",
			Optional:            true,
			Validators:          []validator.Int64{int64validator.Between(1, 90)},
		},
		"notifications_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable update notifications to users.",
			Optional:            true,
		},
		"rapid_security_response_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable Rapid Security Response updates.",
			Optional:            true,
		},
		"rapid_security_response_rollback_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable rollback capability for Rapid Security Response updates.",
			Optional:            true,
		},
		"recommended_cadence": schema.StringAttribute{
			MarkdownDescription: "Recommended update cadence policy. Valid values: `All`, `Oldest`, `Newest`.",
			Optional:            true,
			Validators:          []validator.String{stringvalidator.OneOf("All", "Oldest", "Newest")},
		},
		"beta_require_program_token": schema.StringAttribute{
			MarkdownDescription: "Required beta program token (1-1000 characters). Must be specified with `beta_require_program_description`.",
			Optional:            true,
		},
		"beta_require_program_description": schema.StringAttribute{
			MarkdownDescription: "Required beta program description (1-1000 characters). Must be specified with `beta_require_program_token`.",
			Optional:            true,
		},
		"beta_offer_programs": schema.SetNestedAttribute{
			MarkdownDescription: "Beta programs to offer (max 100). Each program must have a token and description (1-1000 characters each).",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"token": schema.StringAttribute{
						MarkdownDescription: "Beta program token (1-1000 characters).",
						Required:            true,
					},
					"description": schema.StringAttribute{
						MarkdownDescription: "Beta program description (1-1000 characters).",
						Required:            true,
					},
				},
			},
		},
	}
}

// ToRawConfiguration converts the strongly-typed component to the OpenAPI nested format
func (c *SoftwareUpdateSettingsComponent) ToRawConfiguration() (map[string]any, error) {
	config := make(map[string]any)

	config["AllowStandardUserOSUpdates"] = setBoolField(c.AllowStandardUserOSUpdates, false)

	automaticActions := map[string]any{
		"Download":              setStringField(c.AutomaticDownload, "Allowed"),
		"InstallOSUpdates":      setStringField(c.AutomaticInstallOSUpdates, "Allowed"),
		"InstallSecurityUpdate": setStringField(c.AutomaticInstallSecurityUpdate, "Allowed"),
	}
	config["AutomaticActions"] = automaticActions

	hasBetaSettings := helpers.IsConfiguredValue(c.BetaProgramEnrollment) ||
		len(c.BetaOfferPrograms) > 0 ||
		(helpers.IsConfiguredValue(c.BetaRequireProgramToken) && helpers.IsConfiguredValue(c.BetaRequireProgramDescription))

	if hasBetaSettings {
		betaValue := make(map[string]any)
		if helpers.IsConfiguredValue(c.BetaProgramEnrollment) {
			betaValue["ProgramEnrollment"] = c.BetaProgramEnrollment.ValueString()
		}

		if len(c.BetaOfferPrograms) > 0 {
			offerPrograms := make([]map[string]any, len(c.BetaOfferPrograms))
			for i, program := range c.BetaOfferPrograms {
				offerPrograms[i] = map[string]any{
					"Token":       program.Token.ValueString(),
					"Description": program.Description.ValueString(),
				}
			}
			betaValue["OfferPrograms"] = offerPrograms
		}

		if helpers.IsConfiguredValue(c.BetaRequireProgramToken) && helpers.IsConfiguredValue(c.BetaRequireProgramDescription) {
			betaValue["RequireProgram"] = map[string]any{
				"Token":       c.BetaRequireProgramToken.ValueString(),
				"Description": c.BetaRequireProgramDescription.ValueString(),
			}
		}

		config["Beta"] = setValueField(betaValue, true)
	}

	deferrals := map[string]any{
		"CombinedPeriodInDays": setInt64Field(c.DeferralCombinedPeriod, 0),
		"MajorPeriodInDays":    setInt64Field(c.DeferralMajorPeriod, 0),
		"MinorPeriodInDays":    setInt64Field(c.DeferralMinorPeriod, 0),
		"SystemPeriodInDays":   setInt64Field(c.DeferralSystemPeriod, 0),
	}
	config["Deferrals"] = deferrals

	config["Notifications"] = setBoolField(c.NotificationsEnabled, false)

	rapidSecurityResponse := map[string]any{
		"Enable":         setBoolField(c.RapidSecurityResponseEnabled, false),
		"EnableRollback": setBoolField(c.RapidSecurityResponseRollbackEnabled, false),
	}
	config["RapidSecurityResponse"] = rapidSecurityResponse

	config["RecommendedCadence"] = setStringField(c.RecommendedCadence, "All")

	return config, nil
}

// FromRawConfiguration populates the strongly-typed component from OpenAPI nested configuration
func (c *SoftwareUpdateSettingsComponent) FromRawConfiguration(rawConfig map[string]any) error {
	extractOptionallyEnabled := func(key string) types.Bool {
		if obj, exists := rawConfig[key]; exists {
			if objMap, ok := obj.(map[string]any); ok {
				if enabled, hasEnabled := objMap["Enabled"]; hasEnabled {
					if included, hasIncluded := objMap["Included"]; hasIncluded && included.(bool) {
						return types.BoolValue(enabled.(bool))
					}
				}
			}
		}
		return types.BoolNull()
	}

	extractValue := func(path ...string) any {
		current := rawConfig
		for _, key := range path[:len(path)-1] {
			if next, exists := current[key]; exists {
				if nextMap, ok := next.(map[string]any); ok {
					current = nextMap
				} else {
					return nil
				}
			} else {
				return nil
			}
		}

		finalKey := path[len(path)-1]
		if obj, exists := current[finalKey]; exists {
			if objMap, ok := obj.(map[string]any); ok {
				if value, hasValue := objMap["Value"]; hasValue {
					if included, hasIncluded := objMap["Included"]; hasIncluded && included.(bool) {
						return value
					}
				}
			}
		}
		return nil
	}

	extractInt64Value := func(path ...string) types.Int64 {
		val := extractValue(path...)
		if val == nil {
			return types.Int64Null()
		}
		switch v := val.(type) {
		case float64:
			return types.Int64Value(int64(v))
		case int:
			return types.Int64Value(int64(v))
		case int64:
			return types.Int64Value(v)
		}
		return types.Int64Null()
	}

	c.AllowStandardUserOSUpdates = extractOptionallyEnabled("AllowStandardUserOSUpdates")
	c.NotificationsEnabled = extractOptionallyEnabled("Notifications")

	if val := extractValue("AutomaticActions", "Download"); val != nil {
		c.AutomaticDownload = types.StringValue(val.(string))
	} else {
		c.AutomaticDownload = types.StringNull()
	}

	if val := extractValue("AutomaticActions", "InstallOSUpdates"); val != nil {
		c.AutomaticInstallOSUpdates = types.StringValue(val.(string))
	} else {
		c.AutomaticInstallOSUpdates = types.StringNull()
	}

	if val := extractValue("AutomaticActions", "InstallSecurityUpdate"); val != nil {
		c.AutomaticInstallSecurityUpdate = types.StringValue(val.(string))
	} else {
		c.AutomaticInstallSecurityUpdate = types.StringNull()
	}

	if beta, exists := rawConfig["Beta"]; exists {
		if betaMap, ok := beta.(map[string]any); ok {
			if value, hasValue := betaMap["Value"]; hasValue {
				if included, hasIncluded := betaMap["Included"]; hasIncluded && included.(bool) {
					if valueMap, ok := value.(map[string]any); ok {
						if enrollment, hasEnrollment := valueMap["ProgramEnrollment"]; hasEnrollment {
							c.BetaProgramEnrollment = types.StringValue(enrollment.(string))
						}

						if offerPrograms, hasOffer := valueMap["OfferPrograms"]; hasOffer {
							if programList, ok := offerPrograms.([]any); ok {
								c.BetaOfferPrograms = make([]BetaProgramModel, len(programList))
								for i, program := range programList {
									if progMap, ok := program.(map[string]any); ok {
										c.BetaOfferPrograms[i] = BetaProgramModel{
											Token:       types.StringValue(progMap["Token"].(string)),
											Description: types.StringValue(progMap["Description"].(string)),
										}
									}
								}
							}
						}

						if requireProgram, hasRequire := valueMap["RequireProgram"]; hasRequire {
							if progMap, ok := requireProgram.(map[string]any); ok {
								if token, hasToken := progMap["Token"]; hasToken {
									if tokenStr, ok := token.(string); ok {
										c.BetaRequireProgramToken = types.StringValue(tokenStr)
									}
								}
								if desc, hasDesc := progMap["Description"]; hasDesc {
									if descStr, ok := desc.(string); ok {
										c.BetaRequireProgramDescription = types.StringValue(descStr)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	c.DeferralCombinedPeriod = extractInt64Value("Deferrals", "CombinedPeriodInDays")
	c.DeferralMajorPeriod = extractInt64Value("Deferrals", "MajorPeriodInDays")
	c.DeferralMinorPeriod = extractInt64Value("Deferrals", "MinorPeriodInDays")
	c.DeferralSystemPeriod = extractInt64Value("Deferrals", "SystemPeriodInDays")

	if rsr, exists := rawConfig["RapidSecurityResponse"]; exists {
		if rsrMap, ok := rsr.(map[string]any); ok {
			if enable, hasEnable := rsrMap["Enable"]; hasEnable {
				if enableMap, ok := enable.(map[string]any); ok {
					if enabled, hasEnabled := enableMap["Enabled"]; hasEnabled {
						if included, hasIncluded := enableMap["Included"]; hasIncluded && included.(bool) {
							c.RapidSecurityResponseEnabled = types.BoolValue(enabled.(bool))
						}
					}
				}
			}

			if rollback, hasRollback := rsrMap["EnableRollback"]; hasRollback {
				if rollbackMap, ok := rollback.(map[string]any); ok {
					if enabled, hasEnabled := rollbackMap["Enabled"]; hasEnabled {
						if included, hasIncluded := rollbackMap["Included"]; hasIncluded && included.(bool) {
							c.RapidSecurityResponseRollbackEnabled = types.BoolValue(enabled.(bool))
						}
					}
				}
			}
		}
	}

	if val := extractValue("RecommendedCadence"); val != nil {
		c.RecommendedCadence = types.StringValue(val.(string))
	} else {
		c.RecommendedCadence = types.StringNull()
	}

	return nil
}

// GetIdentifier returns the component identifier for software update settings
func (c *SoftwareUpdateSettingsComponent) GetIdentifier() string {
	return "com.jamf.ddm.software-update-settings"
}

// ToClientComponent converts the strongly-typed component to a client.BlueprintComponent
func (c *SoftwareUpdateSettingsComponent) ToClientComponent() (*BlueprintComponentData, error) {
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
