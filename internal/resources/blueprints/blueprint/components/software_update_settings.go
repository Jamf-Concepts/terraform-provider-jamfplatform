// Copyright 2025 Jamf Software LLC.

package components

import (
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	DeferralCombinedPeriod               types.String       `tfsdk:"deferral_combined_period_days"`
	DeferralMajorPeriod                  types.String       `tfsdk:"deferral_major_period_days"`
	DeferralMinorPeriod                  types.String       `tfsdk:"deferral_minor_period_days"`
	DeferralSystemPeriod                 types.String       `tfsdk:"deferral_system_period_days"`
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
func SoftwareUpdateSettingsComponentSchema() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"allow_standard_user_os_updates": schema.BoolAttribute{
				Description: "Allow standard users to install OS updates without administrator privileges.",
				Optional:    true,
			},
			"automatic_download": schema.StringAttribute{
				Description: "Automatic download behavior for updates. Valid values: Allowed, AlwaysOn, AlwaysOff.",
				Optional:    true,
			},
			"automatic_install_os_updates": schema.StringAttribute{
				Description: "Automatic installation behavior for OS updates. Valid values: Allowed, AlwaysOn, AlwaysOff.",
				Optional:    true,
			},
			"automatic_install_security_updates": schema.StringAttribute{
				Description: "Automatic installation behavior for security updates. Valid values: Allowed, AlwaysOn, AlwaysOff.",
				Optional:    true,
			},
			"beta_program_enrollment": schema.StringAttribute{
				Description: "Beta program enrollment setting. Valid values: Allowed, AlwaysOn, AlwaysOff.",
				Optional:    true,
			},
			"deferral_combined_period_days": schema.StringAttribute{
				Description: "Number of days to defer combined updates (1-90 days).",
				Optional:    true,
			},
			"deferral_major_period_days": schema.StringAttribute{
				Description: "Number of days to defer major updates (1-90 days).",
				Optional:    true,
			},
			"deferral_minor_period_days": schema.StringAttribute{
				Description: "Number of days to defer minor updates (1-90 days).",
				Optional:    true,
			},
			"deferral_system_period_days": schema.StringAttribute{
				Description: "Number of days to defer system updates (1-90 days).",
				Optional:    true,
			},
			"notifications_enabled": schema.BoolAttribute{
				Description: "Enable update notifications to users.",
				Optional:    true,
			},
			"rapid_security_response_enabled": schema.BoolAttribute{
				Description: "Enable Rapid Security Response updates.",
				Optional:    true,
			},
			"rapid_security_response_rollback_enabled": schema.BoolAttribute{
				Description: "Enable rollback capability for Rapid Security Response updates.",
				Optional:    true,
			},
			"recommended_cadence": schema.StringAttribute{
				Description: "Recommended update cadence policy. Valid values: All, Oldest, Newest.",
				Optional:    true,
			},
			"beta_require_program_token": schema.StringAttribute{
				Description: "Required beta program token (1-1000 characters). Must be specified with beta_require_program_description.",
				Optional:    true,
			},
			"beta_require_program_description": schema.StringAttribute{
				Description: "Required beta program description (1-1000 characters). Must be specified with beta_require_program_token.",
				Optional:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"beta_offer_programs": schema.ListNestedBlock{
				Description: "Beta programs to offer (max 100). Each program must have a token and description (1-1000 characters each).",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"token": schema.StringAttribute{
							Description: "Beta program token (1-1000 characters).",
							Required:    true,
						},
						"description": schema.StringAttribute{
							Description: "Beta program description (1-1000 characters).",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

// ToRawConfiguration converts the strongly-typed component to the OpenAPI nested format
func (c *SoftwareUpdateSettingsComponent) ToRawConfiguration() (map[string]interface{}, error) {
	config := make(map[string]interface{})

	if !c.AllowStandardUserOSUpdates.IsNull() && !c.AllowStandardUserOSUpdates.IsUnknown() {
		config["AllowStandardUserOSUpdates"] = map[string]interface{}{
			"Enabled":  c.AllowStandardUserOSUpdates.ValueBool(),
			"Included": true,
		}
	}

	automaticActions := make(map[string]interface{})
	if !c.AutomaticDownload.IsNull() && !c.AutomaticDownload.IsUnknown() {
		automaticActions["Download"] = map[string]interface{}{
			"Value":    c.AutomaticDownload.ValueString(),
			"Included": true,
		}
	}
	if !c.AutomaticInstallOSUpdates.IsNull() && !c.AutomaticInstallOSUpdates.IsUnknown() {
		automaticActions["InstallOSUpdates"] = map[string]interface{}{
			"Value":    c.AutomaticInstallOSUpdates.ValueString(),
			"Included": true,
		}
	}
	if !c.AutomaticInstallSecurityUpdate.IsNull() && !c.AutomaticInstallSecurityUpdate.IsUnknown() {
		automaticActions["InstallSecurityUpdate"] = map[string]interface{}{
			"Value":    c.AutomaticInstallSecurityUpdate.ValueString(),
			"Included": true,
		}
	}
	if len(automaticActions) > 0 {
		config["AutomaticActions"] = automaticActions
	}

	hasBetaSettings := !c.BetaProgramEnrollment.IsNull() && !c.BetaProgramEnrollment.IsUnknown() ||
		len(c.BetaOfferPrograms) > 0 ||
		(!c.BetaRequireProgramToken.IsNull() && !c.BetaRequireProgramToken.IsUnknown() &&
			!c.BetaRequireProgramDescription.IsNull() && !c.BetaRequireProgramDescription.IsUnknown())

	if hasBetaSettings {
		betaValue := make(map[string]interface{})
		if !c.BetaProgramEnrollment.IsNull() && !c.BetaProgramEnrollment.IsUnknown() {
			betaValue["ProgramEnrollment"] = c.BetaProgramEnrollment.ValueString()
		}

		if len(c.BetaOfferPrograms) > 0 {
			offerPrograms := make([]map[string]interface{}, len(c.BetaOfferPrograms))
			for i, program := range c.BetaOfferPrograms {
				offerPrograms[i] = map[string]interface{}{
					"Token":       program.Token.ValueString(),
					"Description": program.Description.ValueString(),
				}
			}
			betaValue["OfferPrograms"] = offerPrograms
		}

		if !c.BetaRequireProgramToken.IsNull() && !c.BetaRequireProgramToken.IsUnknown() &&
			!c.BetaRequireProgramDescription.IsNull() && !c.BetaRequireProgramDescription.IsUnknown() {
			betaValue["RequireProgram"] = map[string]interface{}{
				"Token":       c.BetaRequireProgramToken.ValueString(),
				"Description": c.BetaRequireProgramDescription.ValueString(),
			}
		}

		config["Beta"] = map[string]interface{}{
			"Value":    betaValue,
			"Included": true,
		}
	}

	deferrals := make(map[string]interface{})
	if !c.DeferralCombinedPeriod.IsNull() && !c.DeferralCombinedPeriod.IsUnknown() {
		deferrals["CombinedPeriodInDays"] = map[string]interface{}{
			"Value":    c.DeferralCombinedPeriod.ValueString(),
			"Included": true,
		}
	}
	if !c.DeferralMajorPeriod.IsNull() && !c.DeferralMajorPeriod.IsUnknown() {
		deferrals["MajorPeriodInDays"] = map[string]interface{}{
			"Value":    c.DeferralMajorPeriod.ValueString(),
			"Included": true,
		}
	}
	if !c.DeferralMinorPeriod.IsNull() && !c.DeferralMinorPeriod.IsUnknown() {
		deferrals["MinorPeriodInDays"] = map[string]interface{}{
			"Value":    c.DeferralMinorPeriod.ValueString(),
			"Included": true,
		}
	}
	if !c.DeferralSystemPeriod.IsNull() && !c.DeferralSystemPeriod.IsUnknown() {
		deferrals["SystemPeriodInDays"] = map[string]interface{}{
			"Value":    c.DeferralSystemPeriod.ValueString(),
			"Included": true,
		}
	}
	if len(deferrals) > 0 {
		config["Deferrals"] = deferrals
	}

	if !c.NotificationsEnabled.IsNull() && !c.NotificationsEnabled.IsUnknown() {
		config["Notifications"] = map[string]interface{}{
			"Enabled":  c.NotificationsEnabled.ValueBool(),
			"Included": true,
		}
	}

	rapidSecurityResponse := make(map[string]interface{})
	if !c.RapidSecurityResponseEnabled.IsNull() && !c.RapidSecurityResponseEnabled.IsUnknown() {
		rapidSecurityResponse["Enable"] = map[string]interface{}{
			"Enabled":  c.RapidSecurityResponseEnabled.ValueBool(),
			"Included": true,
		}
	}
	if !c.RapidSecurityResponseRollbackEnabled.IsNull() && !c.RapidSecurityResponseRollbackEnabled.IsUnknown() {
		rapidSecurityResponse["EnableRollback"] = map[string]interface{}{
			"Enabled":  c.RapidSecurityResponseRollbackEnabled.ValueBool(),
			"Included": true,
		}
	}
	if len(rapidSecurityResponse) > 0 {
		config["RapidSecurityResponse"] = rapidSecurityResponse
	}

	if !c.RecommendedCadence.IsNull() && !c.RecommendedCadence.IsUnknown() {
		config["RecommendedCadence"] = map[string]interface{}{
			"Value":    c.RecommendedCadence.ValueString(),
			"Included": true,
		}
	}

	return config, nil
}

// FromRawConfiguration populates the strongly-typed component from OpenAPI nested configuration
func (c *SoftwareUpdateSettingsComponent) FromRawConfiguration(rawConfig map[string]interface{}) error {
	extractOptionallyEnabled := func(key string) types.Bool {
		if obj, exists := rawConfig[key]; exists {
			if objMap, ok := obj.(map[string]interface{}); ok {
				if enabled, hasEnabled := objMap["Enabled"]; hasEnabled {
					if included, hasIncluded := objMap["Included"]; hasIncluded && included.(bool) {
						return types.BoolValue(enabled.(bool))
					}
				}
			}
		}
		return types.BoolNull()
	}

	extractValue := func(path ...string) interface{} {
		current := rawConfig
		for _, key := range path[:len(path)-1] {
			if next, exists := current[key]; exists {
				if nextMap, ok := next.(map[string]interface{}); ok {
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
			if objMap, ok := obj.(map[string]interface{}); ok {
				if value, hasValue := objMap["Value"]; hasValue {
					if included, hasIncluded := objMap["Included"]; hasIncluded && included.(bool) {
						return value
					}
				}
			}
		}
		return nil
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
		if betaMap, ok := beta.(map[string]interface{}); ok {
			if value, hasValue := betaMap["Value"]; hasValue {
				if included, hasIncluded := betaMap["Included"]; hasIncluded && included.(bool) {
					if valueMap, ok := value.(map[string]interface{}); ok {
						if enrollment, hasEnrollment := valueMap["ProgramEnrollment"]; hasEnrollment {
							c.BetaProgramEnrollment = types.StringValue(enrollment.(string))
						}

						if offerPrograms, hasOffer := valueMap["OfferPrograms"]; hasOffer {
							if programList, ok := offerPrograms.([]interface{}); ok {
								c.BetaOfferPrograms = make([]BetaProgramModel, len(programList))
								for i, program := range programList {
									if progMap, ok := program.(map[string]interface{}); ok {
										c.BetaOfferPrograms[i] = BetaProgramModel{
											Token:       types.StringValue(progMap["Token"].(string)),
											Description: types.StringValue(progMap["Description"].(string)),
										}
									}
								}
							}
						}

						if requireProgram, hasRequire := valueMap["RequireProgram"]; hasRequire {
							if progMap, ok := requireProgram.(map[string]interface{}); ok {
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

	if val := extractValue("Deferrals", "CombinedPeriodInDays"); val != nil {
		c.DeferralCombinedPeriod = types.StringValue(val.(string))
	} else {
		c.DeferralCombinedPeriod = types.StringNull()
	}

	if val := extractValue("Deferrals", "MajorPeriodInDays"); val != nil {
		c.DeferralMajorPeriod = types.StringValue(val.(string))
	} else {
		c.DeferralMajorPeriod = types.StringNull()
	}

	if val := extractValue("Deferrals", "MinorPeriodInDays"); val != nil {
		c.DeferralMinorPeriod = types.StringValue(val.(string))
	} else {
		c.DeferralMinorPeriod = types.StringNull()
	}

	if val := extractValue("Deferrals", "SystemPeriodInDays"); val != nil {
		c.DeferralSystemPeriod = types.StringValue(val.(string))
	} else {
		c.DeferralSystemPeriod = types.StringNull()
	}

	if rsr, exists := rawConfig["RapidSecurityResponse"]; exists {
		if rsrMap, ok := rsr.(map[string]interface{}); ok {
			if enable, hasEnable := rsrMap["Enable"]; hasEnable {
				if enableMap, ok := enable.(map[string]interface{}); ok {
					if enabled, hasEnabled := enableMap["Enabled"]; hasEnabled {
						if included, hasIncluded := enableMap["Included"]; hasIncluded && included.(bool) {
							c.RapidSecurityResponseEnabled = types.BoolValue(enabled.(bool))
						}
					}
				}
			}

			if rollback, hasRollback := rsrMap["EnableRollback"]; hasRollback {
				if rollbackMap, ok := rollback.(map[string]interface{}); ok {
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
