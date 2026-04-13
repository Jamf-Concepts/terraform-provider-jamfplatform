// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package blueprint

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/blueprints/blueprint/components"
	resourceTimeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.ResourceWithUpgradeState = &BlueprintResource{}

// UpgradeState returns state upgraders for migrating between schema versions.
func (r *BlueprintResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: blueprintSchemaV0(),
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var old blueprintResourceModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &old)...)
				if resp.Diagnostics.HasError() {
					return
				}

				upgraded := BlueprintResourceModel{
					ID:                        old.ID,
					Name:                      old.Name,
					Description:               old.Description,
					Deployed:                  old.Deployed,
					DeviceGroups:              old.DeviceGroups,
					Components:                old.Components,
					AudioAccessorySettings:    sliceToPointer(old.AudioAccessorySettings),
					CustomDeclarations:        sliceToPointer(old.CustomDeclarations),
					DiskManagementSettings:    sliceToPointer(old.DiskManagementSettings),
					MathSettings:              sliceToPointer(old.MathSettings),
					PasscodePolicy:            sliceToPointer(old.PasscodePolicy),
					SafariBookmarks:           sliceToPointer(old.SafariBookmarks),
					SafariExtensions:          sliceToPointer(old.SafariExtensions),
					SafariSettings:            sliceToPointer(old.SafariSettings),
					ServiceBackgroundTasks:    sliceToPointer(old.ServiceBackgroundTasks),
					ServiceConfigurationFiles: sliceToPointer(old.ServiceConfigurationFiles),
					SoftwareUpdate:            sliceToPointer(old.SoftwareUpdate),
					SoftwareUpdateSettings:    upgradeSoftwareUpdateSettingsFromSlice(old.SoftwareUpdateSettings),
					LegacyPayloads:            upgradeLegacyPayloadsFromString(old.LegacyPayloads),
					Created:                   old.Created,
					Updated:                   old.Updated,
					DeploymentState:           old.DeploymentState,
					Timeouts:                  resourceTimeouts.Value{Object: old.Timeouts},
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
			},
		},
		1: {
			PriorSchema: blueprintSchemaV1(),
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var old blueprintResourceModelV1
				resp.Diagnostics.Append(req.State.Get(ctx, &old)...)
				if resp.Diagnostics.HasError() {
					return
				}

				upgraded := BlueprintResourceModel{
					ID:                        old.ID,
					Name:                      old.Name,
					Description:               old.Description,
					Deployed:                  old.Deployed,
					DeviceGroups:              old.DeviceGroups,
					Components:                old.Components,
					AudioAccessorySettings:    old.AudioAccessorySettings,
					CustomDeclarations:        old.CustomDeclarations,
					DiskManagementSettings:    old.DiskManagementSettings,
					MathSettings:              old.MathSettings,
					PasscodePolicy:            old.PasscodePolicy,
					SafariBookmarks:           old.SafariBookmarks,
					SafariExtensions:          old.SafariExtensions,
					SafariSettings:            old.SafariSettings,
					ServiceBackgroundTasks:    old.ServiceBackgroundTasks,
					ServiceConfigurationFiles: old.ServiceConfigurationFiles,
					SoftwareUpdate:            old.SoftwareUpdate,
					SoftwareUpdateSettings:    upgradeSoftwareUpdateSettings(old.SoftwareUpdateSettings),
					LegacyPayloads:            upgradeLegacyPayloadsFromString(old.LegacyPayloads),
					Created:                   old.Created,
					Updated:                   old.Updated,
					DeploymentState:           old.DeploymentState,
					Timeouts:                  resourceTimeouts.Value{Object: old.Timeouts},
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
			},
		},
		2: {
			PriorSchema: blueprintSchemaV2(),
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var old blueprintResourceModelV2
				resp.Diagnostics.Append(req.State.Get(ctx, &old)...)
				if resp.Diagnostics.HasError() {
					return
				}

				upgraded := BlueprintResourceModel{
					ID:                        old.ID,
					Name:                      old.Name,
					Description:               old.Description,
					Deployed:                  old.Deployed,
					DeviceGroups:              old.DeviceGroups,
					Components:                old.Components,
					AudioAccessorySettings:    old.AudioAccessorySettings,
					CustomDeclarations:        old.CustomDeclarations,
					DiskManagementSettings:    old.DiskManagementSettings,
					MathSettings:              old.MathSettings,
					PasscodePolicy:            old.PasscodePolicy,
					SafariBookmarks:           old.SafariBookmarks,
					SafariExtensions:          old.SafariExtensions,
					SafariSettings:            old.SafariSettings,
					ServiceBackgroundTasks:    old.ServiceBackgroundTasks,
					ServiceConfigurationFiles: old.ServiceConfigurationFiles,
					SoftwareUpdate:            old.SoftwareUpdate,
					SoftwareUpdateSettings:    upgradeSoftwareUpdateSettings(old.SoftwareUpdateSettings),
					LegacyPayloads:            old.LegacyPayloads,
					Created:                   old.Created,
					Updated:                   old.Updated,
					DeploymentState:           old.DeploymentState,
					Timeouts:                  old.Timeouts,
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
			},
		},
	}
}

// sliceToPointer converts a slice from a ListNestedBlock to a pointer for SingleNestedAttribute.
func sliceToPointer[T any](s []T) *T {
	if len(s) > 0 {
		return &s[0]
	}
	return nil
}

// upgradeLegacyPayloadsFromString converts the v0/v1 JSON string format to the v2 dynamic format.
func upgradeLegacyPayloadsFromString(legacyPayloads types.String) types.Dynamic {
	if legacyPayloads.IsNull() || legacyPayloads.IsUnknown() || legacyPayloads.ValueString() == "" {
		return types.DynamicNull()
	}

	var payloadArray []map[string]any
	if err := json.Unmarshal([]byte(legacyPayloads.ValueString()), &payloadArray); err != nil {
		return types.DynamicNull()
	}

	if len(payloadArray) == 0 {
		return types.DynamicNull()
	}

	var resultItems []any
	for _, payload := range payloadArray {
		payloadType, _ := payload["payloadType"].(string)

		settingsMap := make(map[string]any, len(payload))
		for k, v := range payload {
			if k == "payloadType" || k == "payloadIdentifier" {
				continue
			}
			settingsMap[k] = v
		}

		entry := map[string]any{
			"payload_type": payloadType,
		}
		if len(settingsMap) > 0 {
			entry["settings"] = settingsMap
		}

		resultItems = append(resultItems, entry)
	}

	dynVal, err := helpers.JSONToTerraformDynamic(resultItems)
	if err != nil {
		return types.DynamicNull()
	}

	return dynVal
}

// softwareUpdateSettingsComponentPrior is the v0-v2 software update settings model with string deferral fields.
type softwareUpdateSettingsComponentPrior struct {
	AllowStandardUserOSUpdates           types.Bool                      `tfsdk:"allow_standard_user_os_updates"`
	AutomaticDownload                    types.String                    `tfsdk:"automatic_download"`
	AutomaticInstallOSUpdates            types.String                    `tfsdk:"automatic_install_os_updates"`
	AutomaticInstallSecurityUpdate       types.String                    `tfsdk:"automatic_install_security_updates"`
	BetaProgramEnrollment                types.String                    `tfsdk:"beta_program_enrollment"`
	BetaOfferPrograms                    []components.BetaProgramModel   `tfsdk:"beta_offer_programs"`
	BetaRequireProgramToken              types.String                    `tfsdk:"beta_require_program_token"`
	BetaRequireProgramDescription        types.String                    `tfsdk:"beta_require_program_description"`
	DeferralCombinedPeriod               types.String                    `tfsdk:"deferral_combined_period_days"`
	DeferralMajorPeriod                  types.String                    `tfsdk:"deferral_major_period_days"`
	DeferralMinorPeriod                  types.String                    `tfsdk:"deferral_minor_period_days"`
	DeferralSystemPeriod                 types.String                    `tfsdk:"deferral_system_period_days"`
	NotificationsEnabled                 types.Bool                      `tfsdk:"notifications_enabled"`
	RapidSecurityResponseEnabled         types.Bool                      `tfsdk:"rapid_security_response_enabled"`
	RapidSecurityResponseRollbackEnabled types.Bool                      `tfsdk:"rapid_security_response_rollback_enabled"`
	RecommendedCadence                   types.String                    `tfsdk:"recommended_cadence"`
}

// upgradeSoftwareUpdateSettings converts v0-v2 string deferral fields to v3 int64 fields.
func upgradeSoftwareUpdateSettings(old *softwareUpdateSettingsComponentPrior) *components.SoftwareUpdateSettingsComponent {
	if old == nil {
		return nil
	}
	return &components.SoftwareUpdateSettingsComponent{
		AllowStandardUserOSUpdates:           old.AllowStandardUserOSUpdates,
		AutomaticDownload:                    old.AutomaticDownload,
		AutomaticInstallOSUpdates:            old.AutomaticInstallOSUpdates,
		AutomaticInstallSecurityUpdate:       old.AutomaticInstallSecurityUpdate,
		BetaProgramEnrollment:                old.BetaProgramEnrollment,
		BetaOfferPrograms:                    old.BetaOfferPrograms,
		BetaRequireProgramToken:              old.BetaRequireProgramToken,
		BetaRequireProgramDescription:        old.BetaRequireProgramDescription,
		DeferralCombinedPeriod:               stringToInt64(old.DeferralCombinedPeriod),
		DeferralMajorPeriod:                  stringToInt64(old.DeferralMajorPeriod),
		DeferralMinorPeriod:                  stringToInt64(old.DeferralMinorPeriod),
		DeferralSystemPeriod:                 stringToInt64(old.DeferralSystemPeriod),
		NotificationsEnabled:                 old.NotificationsEnabled,
		RapidSecurityResponseEnabled:         old.RapidSecurityResponseEnabled,
		RapidSecurityResponseRollbackEnabled: old.RapidSecurityResponseRollbackEnabled,
		RecommendedCadence:                   old.RecommendedCadence,
	}
}

// upgradeSoftwareUpdateSettingsFromSlice converts a v0 slice to the current pointer type with deferral migration.
func upgradeSoftwareUpdateSettingsFromSlice(old []softwareUpdateSettingsComponentPrior) *components.SoftwareUpdateSettingsComponent {
	if len(old) == 0 {
		return nil
	}
	return upgradeSoftwareUpdateSettings(&old[0])
}

// stringToInt64 converts a types.String to types.Int64, returning null if the string is empty or not parseable.
func stringToInt64(s types.String) types.Int64 {
	if s.IsNull() || s.IsUnknown() || s.ValueString() == "" {
		return types.Int64Null()
	}
	val, err := strconv.ParseInt(s.ValueString(), 10, 64)
	if err != nil {
		return types.Int64Null()
	}
	return types.Int64Value(val)
}

// blueprintResourceModelV0 is the v0 resource model where typed components were ListNestedBlocks.
type blueprintResourceModelV0 struct {
	ID                        types.String                                    `tfsdk:"id"`
	Name                      types.String                                    `tfsdk:"name"`
	Description               types.String                                    `tfsdk:"description"`
	Deployed                  types.Bool                                      `tfsdk:"deployed"`
	DeviceGroups              types.Set                                       `tfsdk:"device_groups"`
	Components                []ComponentModel                                `tfsdk:"raw_component"`
	AudioAccessorySettings    []components.AudioAccessorySettingsComponent    `tfsdk:"audio_accessory_settings"`
	CustomDeclarations        []components.CustomDeclarationsComponent        `tfsdk:"custom_declarations"`
	DiskManagementSettings    []components.DiskManagementPolicyComponent      `tfsdk:"disk_management_settings"`
	MathSettings              []components.MathSettingsComponent              `tfsdk:"math_settings"`
	PasscodePolicy            []components.PasscodePolicyComponent            `tfsdk:"passcode_policy"`
	SafariBookmarks           []components.SafariBookmarksComponent           `tfsdk:"safari_bookmarks"`
	SafariExtensions          []components.SafariExtensionsComponent          `tfsdk:"safari_extensions"`
	SafariSettings            []components.SafariSettingsComponent            `tfsdk:"safari_settings"`
	ServiceBackgroundTasks    []components.ServiceBackgroundTasksComponent    `tfsdk:"service_background_tasks"`
	ServiceConfigurationFiles []components.ServiceConfigurationFilesComponent `tfsdk:"service_configuration_files"`
	SoftwareUpdate            []components.SoftwareUpdateComponent            `tfsdk:"software_update"`
	SoftwareUpdateSettings    []softwareUpdateSettingsComponentPrior          `tfsdk:"software_update_settings"`
	LegacyPayloads            types.String                                    `tfsdk:"legacy_payloads"`
	Created                   types.String                                    `tfsdk:"created"`
	Updated                   types.String                                    `tfsdk:"updated"`
	DeploymentState           types.String                                    `tfsdk:"deployment_state"`
	Timeouts                  types.Object                                    `tfsdk:"timeouts"`
}

// blueprintResourceModelV1 is the v1 resource model where legacy_payloads was a JSON string.
type blueprintResourceModelV1 struct {
	ID                        types.String                                   `tfsdk:"id"`
	Name                      types.String                                   `tfsdk:"name"`
	Description               types.String                                   `tfsdk:"description"`
	Deployed                  types.Bool                                     `tfsdk:"deployed"`
	DeviceGroups              types.Set                                      `tfsdk:"device_groups"`
	Components                []ComponentModel                               `tfsdk:"raw_component"`
	AudioAccessorySettings    *components.AudioAccessorySettingsComponent    `tfsdk:"audio_accessory_settings"`
	CustomDeclarations        *components.CustomDeclarationsComponent        `tfsdk:"custom_declarations"`
	DiskManagementSettings    *components.DiskManagementPolicyComponent      `tfsdk:"disk_management_settings"`
	MathSettings              *components.MathSettingsComponent              `tfsdk:"math_settings"`
	PasscodePolicy            *components.PasscodePolicyComponent            `tfsdk:"passcode_policy"`
	SafariBookmarks           *components.SafariBookmarksComponent           `tfsdk:"safari_bookmarks"`
	SafariExtensions          *components.SafariExtensionsComponent          `tfsdk:"safari_extensions"`
	SafariSettings            *components.SafariSettingsComponent            `tfsdk:"safari_settings"`
	ServiceBackgroundTasks    *components.ServiceBackgroundTasksComponent    `tfsdk:"service_background_tasks"`
	ServiceConfigurationFiles *components.ServiceConfigurationFilesComponent `tfsdk:"service_configuration_files"`
	SoftwareUpdate            *components.SoftwareUpdateComponent            `tfsdk:"software_update"`
	SoftwareUpdateSettings    *softwareUpdateSettingsComponentPrior          `tfsdk:"software_update_settings"`
	LegacyPayloads            types.String                                   `tfsdk:"legacy_payloads"`
	Created                   types.String                                   `tfsdk:"created"`
	Updated                   types.String                                   `tfsdk:"updated"`
	DeploymentState           types.String                                   `tfsdk:"deployment_state"`
	Timeouts                  types.Object                                   `tfsdk:"timeouts"`
}

// blueprintResourceModelV2 is the v2 resource model where deferral fields were strings.
type blueprintResourceModelV2 struct {
	ID                        types.String                                   `tfsdk:"id"`
	Name                      types.String                                   `tfsdk:"name"`
	Description               types.String                                   `tfsdk:"description"`
	Deployed                  types.Bool                                     `tfsdk:"deployed"`
	DeviceGroups              types.Set                                      `tfsdk:"device_groups"`
	Components                []ComponentModel                               `tfsdk:"raw_component"`
	AudioAccessorySettings    *components.AudioAccessorySettingsComponent    `tfsdk:"audio_accessory_settings"`
	CustomDeclarations        *components.CustomDeclarationsComponent        `tfsdk:"custom_declarations"`
	DiskManagementSettings    *components.DiskManagementPolicyComponent      `tfsdk:"disk_management_settings"`
	MathSettings              *components.MathSettingsComponent              `tfsdk:"math_settings"`
	PasscodePolicy            *components.PasscodePolicyComponent            `tfsdk:"passcode_policy"`
	SafariBookmarks           *components.SafariBookmarksComponent           `tfsdk:"safari_bookmarks"`
	SafariExtensions          *components.SafariExtensionsComponent          `tfsdk:"safari_extensions"`
	SafariSettings            *components.SafariSettingsComponent            `tfsdk:"safari_settings"`
	ServiceBackgroundTasks    *components.ServiceBackgroundTasksComponent    `tfsdk:"service_background_tasks"`
	ServiceConfigurationFiles *components.ServiceConfigurationFilesComponent `tfsdk:"service_configuration_files"`
	SoftwareUpdate            *components.SoftwareUpdateComponent            `tfsdk:"software_update"`
	SoftwareUpdateSettings    *softwareUpdateSettingsComponentPrior          `tfsdk:"software_update_settings"`
	LegacyPayloads            types.Dynamic                                  `tfsdk:"legacy_payloads"`
	Created                   types.String                                   `tfsdk:"created"`
	Updated                   types.String                                   `tfsdk:"updated"`
	DeploymentState           types.String                                   `tfsdk:"deployment_state"`
	Timeouts                  resourceTimeouts.Value                         `tfsdk:"timeouts"`
}

// blueprintSchemaV0 returns the v0 schema where components were blocks instead of nested attributes.
func blueprintSchemaV0() *schema.Schema {
	return &schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":               schema.StringAttribute{Computed: true},
			"name":             schema.StringAttribute{Required: true},
			"description":      schema.StringAttribute{Optional: true},
			"deployed":         schema.BoolAttribute{Required: true},
			"device_groups":    schema.SetAttribute{Required: true, ElementType: types.StringType},
			"legacy_payloads":  schema.StringAttribute{Optional: true},
			"created":          schema.StringAttribute{Computed: true},
			"updated":          schema.StringAttribute{Computed: true},
			"deployment_state": schema.StringAttribute{Computed: true},
			"timeouts":         schema.ObjectAttribute{Optional: true, AttributeTypes: blueprintTimeoutAttributeTypes},
		},
		Blocks: map[string]schema.Block{
			"raw_component": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"identifier":    schema.StringAttribute{Required: true},
						"configuration": schema.MapAttribute{Optional: true, ElementType: types.StringType},
					},
				},
			},
			"audio_accessory_settings": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"temporary_pairing_disabled": schema.BoolAttribute{Required: true},
						"unpairing_time_policy":      schema.StringAttribute{Optional: true, Computed: true},
						"unpairing_time_hour":        schema.Int64Attribute{Optional: true, Computed: true},
					},
				},
			},
			"custom_declarations": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"declaration": schema.ListNestedBlock{
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"channel": schema.StringAttribute{Required: true},
									"kind":    schema.StringAttribute{Required: true},
									"payload": schema.StringAttribute{Required: true},
									"type":    schema.StringAttribute{Required: true},
								},
							},
						},
					},
				},
			},
			"disk_management_settings": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"external_storage": schema.StringAttribute{Optional: true},
						"network_storage":  schema.StringAttribute{Optional: true},
					},
				},
			},
			"math_settings": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"calculator_basic_mode_add_square_root":  schema.BoolAttribute{Optional: true},
						"calculator_scientific_mode_enabled":     schema.BoolAttribute{Optional: true},
						"calculator_programmer_mode_enabled":     schema.BoolAttribute{Optional: true},
						"calculator_math_notes_mode_enabled":     schema.BoolAttribute{Optional: true},
						"calculator_input_modes_unit_conversion": schema.BoolAttribute{Optional: true},
						"calculator_input_modes_rpn":             schema.BoolAttribute{Optional: true},
						"system_behavior_keyboard_suggestions":   schema.BoolAttribute{Optional: true},
						"system_behavior_math_notes":             schema.BoolAttribute{Optional: true},
					},
				},
			},
			"passcode_policy": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"change_at_next_auth":              schema.BoolAttribute{Optional: true},
						"failed_attempts_reset_in_minutes": schema.Int64Attribute{Optional: true},
						"maximum_failed_attempts":          schema.Int64Attribute{Optional: true},
						"maximum_grace_period_in_minutes":  schema.Int64Attribute{Optional: true},
						"maximum_inactivity_in_minutes":    schema.Int64Attribute{Optional: true},
						"maximum_passcode_age_in_days":     schema.Int64Attribute{Optional: true},
						"minimum_complex_characters":       schema.Int64Attribute{Optional: true},
						"minimum_length":                   schema.Int64Attribute{Optional: true},
						"passcode_reuse_limit":             schema.Int64Attribute{Optional: true},
						"require_alphanumeric_passcode":    schema.BoolAttribute{Optional: true},
						"require_complex_passcode":         schema.BoolAttribute{Optional: true},
						"require_passcode":                 schema.BoolAttribute{Optional: true},
					},
				},
			},
			"safari_bookmarks": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"managed_bookmarks": schema.ListNestedBlock{
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"group_identifier": schema.StringAttribute{Required: true},
									"title":            schema.StringAttribute{Required: true},
								},
								Blocks: map[string]schema.Block{
									"bookmarks": schema.ListNestedBlock{
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"type":  schema.StringAttribute{Optional: true},
												"title": schema.StringAttribute{Required: true},
												"url":   schema.StringAttribute{Optional: true},
											},
											Blocks: map[string]schema.Block{
												"folder": schema.ListNestedBlock{
													NestedObject: schema.NestedBlockObject{
														Attributes: map[string]schema.Attribute{
															"title": schema.StringAttribute{Required: true},
															"url":   schema.StringAttribute{Required: true},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"safari_extensions": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"managed_extensions": schema.ListNestedBlock{
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"extension_id":     schema.StringAttribute{Required: true},
									"state":            schema.StringAttribute{Optional: true},
									"private_browsing": schema.StringAttribute{Optional: true},
								},
								Blocks: map[string]schema.Block{
									"allowed_domains": schema.ListNestedBlock{
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"domain": schema.StringAttribute{Required: true},
											},
										},
									},
									"denied_domains": schema.ListNestedBlock{
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"domain": schema.StringAttribute{Required: true},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"safari_settings": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"accept_cookies":                  schema.StringAttribute{Optional: true},
						"allow_disabling_fraud_warning":   schema.BoolAttribute{Optional: true},
						"allow_history_clearing":          schema.BoolAttribute{Optional: true},
						"allow_javascript":                schema.BoolAttribute{Optional: true},
						"allow_private_browsing":          schema.BoolAttribute{Optional: true},
						"allow_popups":                    schema.BoolAttribute{Optional: true},
						"allow_summary":                   schema.BoolAttribute{Optional: true},
						"new_tab_start_page_type":         schema.StringAttribute{Optional: true},
						"new_tab_start_page_homepage_url": schema.StringAttribute{Optional: true},
						"new_tab_start_page_extension_id": schema.StringAttribute{Optional: true},
					},
				},
			},
			"service_background_tasks": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"background_tasks": schema.ListNestedBlock{
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"task_type":        schema.StringAttribute{Required: true},
									"task_description": schema.StringAttribute{Optional: true},
								},
								Blocks: map[string]schema.Block{
									"executable_asset_reference": schema.SingleNestedBlock{
										Attributes: map[string]schema.Attribute{
											"data_url":     schema.StringAttribute{Required: true},
											"hash_sha_256": schema.StringAttribute{Optional: true},
											"content_type": schema.StringAttribute{Computed: true},
										},
									},
									"launchd_configurations": schema.ListNestedBlock{
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"context": schema.StringAttribute{Required: true},
											},
											Blocks: map[string]schema.Block{
												"file_asset_reference": schema.SingleNestedBlock{
													Attributes: map[string]schema.Attribute{
														"data_url":     schema.StringAttribute{Required: true},
														"hash_sha_256": schema.StringAttribute{Optional: true},
														"content_type": schema.StringAttribute{Optional: true},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"service_configuration_files": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"service_config_files": schema.ListNestedBlock{
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"service_type": schema.StringAttribute{Required: true},
								},
								Blocks: map[string]schema.Block{
									"data_asset_reference": schema.SingleNestedBlock{
										Attributes: map[string]schema.Attribute{
											"data_url":     schema.StringAttribute{Required: true},
											"hash_sha_256": schema.StringAttribute{Optional: true},
											"content_type": schema.StringAttribute{Computed: true},
										},
									},
								},
							},
						},
					},
				},
			},
			"software_update": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"enforcement_type":       schema.StringAttribute{Computed: true},
						"deployment_time":        schema.StringAttribute{Optional: true},
						"enforce_after_days":     schema.Int64Attribute{Optional: true},
						"ignore_major_versions":  schema.BoolAttribute{Optional: true},
						"target_os_version":      schema.StringAttribute{Optional: true},
						"target_local_date_time": schema.StringAttribute{Optional: true},
						"details_url_value":      schema.StringAttribute{Optional: true},
					},
				},
			},
			"software_update_settings": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"allow_standard_user_os_updates":           schema.BoolAttribute{Optional: true},
						"automatic_download":                       schema.StringAttribute{Optional: true},
						"automatic_install_os_updates":             schema.StringAttribute{Optional: true},
						"automatic_install_security_updates":       schema.StringAttribute{Optional: true},
						"beta_program_enrollment":                  schema.StringAttribute{Optional: true},
						"beta_require_program_token":               schema.StringAttribute{Optional: true},
						"beta_require_program_description":         schema.StringAttribute{Optional: true},
						"deferral_combined_period_days":            schema.StringAttribute{Optional: true},
						"deferral_major_period_days":               schema.StringAttribute{Optional: true},
						"deferral_minor_period_days":               schema.StringAttribute{Optional: true},
						"deferral_system_period_days":              schema.StringAttribute{Optional: true},
						"notifications_enabled":                    schema.BoolAttribute{Optional: true},
						"rapid_security_response_enabled":          schema.BoolAttribute{Optional: true},
						"rapid_security_response_rollback_enabled": schema.BoolAttribute{Optional: true},
						"recommended_cadence":                      schema.StringAttribute{Optional: true},
					},
					Blocks: map[string]schema.Block{
						"beta_offer_programs": schema.ListNestedBlock{
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"token":       schema.StringAttribute{Required: true},
									"description": schema.StringAttribute{Required: true},
								},
							},
						},
					},
				},
			},
		},
	}
}

// blueprintSchemaV1 returns the v1 schema where legacy_payloads was a JSON string attribute.
func blueprintSchemaV1() *schema.Schema {
	return &schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":               schema.StringAttribute{Computed: true},
			"name":             schema.StringAttribute{Required: true},
			"description":      schema.StringAttribute{Optional: true},
			"deployed":         schema.BoolAttribute{Required: true},
			"device_groups":    schema.SetAttribute{Required: true, ElementType: types.StringType},
			"legacy_payloads":  schema.StringAttribute{Optional: true},
			"created":          schema.StringAttribute{Computed: true},
			"updated":          schema.StringAttribute{Computed: true},
			"deployment_state": schema.StringAttribute{Computed: true},
			"raw_component": schema.SetNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"identifier":    schema.StringAttribute{Required: true},
						"configuration": schema.MapAttribute{Optional: true, ElementType: types.StringType},
					},
				},
			},
			"audio_accessory_settings": schema.SingleNestedAttribute{
				Optional:   true,
				Attributes: components.AudioAccessorySettingsComponentSchema(),
			},
			"custom_declarations": schema.SingleNestedAttribute{
				Optional:   true,
				Attributes: components.CustomDeclarationsComponentSchema(),
			},
			"disk_management_settings": schema.SingleNestedAttribute{
				Optional:   true,
				Attributes: components.DiskManagementPolicyComponentSchema(),
			},
			"math_settings": schema.SingleNestedAttribute{
				Optional:   true,
				Attributes: components.MathSettingsComponentSchema(),
			},
			"passcode_policy": schema.SingleNestedAttribute{
				Optional:   true,
				Attributes: passcodePolicySchemaPrior(),
			},
			"safari_bookmarks": schema.SingleNestedAttribute{
				Optional:   true,
				Attributes: components.SafariBookmarksComponentSchema(),
			},
			"safari_extensions": schema.SingleNestedAttribute{
				Optional:   true,
				Attributes: components.SafariExtensionsComponentSchema(),
			},
			"safari_settings": schema.SingleNestedAttribute{
				Optional:   true,
				Attributes: components.SafariSettingsComponentSchema(),
			},
			"service_background_tasks": schema.SingleNestedAttribute{
				Optional:   true,
				Attributes: components.ServiceBackgroundTasksComponentSchema(),
			},
			"service_configuration_files": schema.SingleNestedAttribute{
				Optional:   true,
				Attributes: components.ServiceConfigurationFilesComponentSchema(),
			},
			"software_update": schema.SingleNestedAttribute{
				Optional:   true,
				Attributes: components.SoftwareUpdateComponentSchema(),
			},
			"software_update_settings": schema.SingleNestedAttribute{
				Optional:   true,
				Attributes: softwareUpdateSettingsSchemaV2(),
			},
			"timeouts": schema.ObjectAttribute{Optional: true, AttributeTypes: blueprintTimeoutAttributeTypes},
		},
	}
}

// blueprintSchemaV2 returns the v2 schema where deferral fields were strings.
func blueprintSchemaV2() *schema.Schema {
	return &schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":               schema.StringAttribute{Computed: true},
			"name":             schema.StringAttribute{Required: true},
			"description":      schema.StringAttribute{Optional: true},
			"deployed":         schema.BoolAttribute{Required: true},
			"device_groups":    schema.SetAttribute{Required: true, ElementType: types.StringType},
			"legacy_payloads":  schema.DynamicAttribute{Optional: true},
			"created":          schema.StringAttribute{Computed: true},
			"updated":          schema.StringAttribute{Computed: true},
			"deployment_state": schema.StringAttribute{Computed: true},
			"raw_component": schema.SetNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"identifier":    schema.StringAttribute{Required: true},
						"configuration": schema.MapAttribute{Optional: true, ElementType: types.StringType},
					},
				},
			},
			"audio_accessory_settings": schema.SingleNestedAttribute{
				Optional:   true,
				Attributes: components.AudioAccessorySettingsComponentSchema(),
			},
			"custom_declarations": schema.SingleNestedAttribute{
				Optional:   true,
				Attributes: components.CustomDeclarationsComponentSchema(),
			},
			"disk_management_settings": schema.SingleNestedAttribute{
				Optional:   true,
				Attributes: components.DiskManagementPolicyComponentSchema(),
			},
			"math_settings": schema.SingleNestedAttribute{
				Optional:   true,
				Attributes: components.MathSettingsComponentSchema(),
			},
			"passcode_policy": schema.SingleNestedAttribute{
				Optional:   true,
				Attributes: passcodePolicySchemaPrior(),
			},
			"safari_bookmarks": schema.SingleNestedAttribute{
				Optional:   true,
				Attributes: components.SafariBookmarksComponentSchema(),
			},
			"safari_extensions": schema.SingleNestedAttribute{
				Optional:   true,
				Attributes: components.SafariExtensionsComponentSchema(),
			},
			"safari_settings": schema.SingleNestedAttribute{
				Optional:   true,
				Attributes: components.SafariSettingsComponentSchema(),
			},
			"service_background_tasks": schema.SingleNestedAttribute{
				Optional:   true,
				Attributes: components.ServiceBackgroundTasksComponentSchema(),
			},
			"service_configuration_files": schema.SingleNestedAttribute{
				Optional:   true,
				Attributes: components.ServiceConfigurationFilesComponentSchema(),
			},
			"software_update": schema.SingleNestedAttribute{
				Optional:   true,
				Attributes: components.SoftwareUpdateComponentSchema(),
			},
			"software_update_settings": schema.SingleNestedAttribute{
				Optional:   true,
				Attributes: softwareUpdateSettingsSchemaV2(),
			},
			"timeouts": schema.ObjectAttribute{Optional: true, AttributeTypes: blueprintTimeoutAttributeTypes},
		},
	}
}

// passcodePolicySchemaPrior returns the prior passcode policy schema without custom regex fields.
func passcodePolicySchemaPrior() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"change_at_next_auth":              schema.BoolAttribute{Optional: true},
		"failed_attempts_reset_in_minutes": schema.Int64Attribute{Optional: true},
		"maximum_failed_attempts":          schema.Int64Attribute{Optional: true},
		"maximum_grace_period_in_minutes":  schema.Int64Attribute{Optional: true},
		"maximum_inactivity_in_minutes":    schema.Int64Attribute{Optional: true},
		"maximum_passcode_age_in_days":     schema.Int64Attribute{Optional: true},
		"minimum_complex_characters":       schema.Int64Attribute{Optional: true},
		"minimum_length":                   schema.Int64Attribute{Optional: true},
		"passcode_reuse_limit":             schema.Int64Attribute{Optional: true},
		"require_alphanumeric_passcode":    schema.BoolAttribute{Optional: true},
		"require_complex_passcode":         schema.BoolAttribute{Optional: true},
		"require_passcode":                 schema.BoolAttribute{Optional: true},
	}
}

// softwareUpdateSettingsSchemaV2 returns the prior schema for software update settings with string deferral attributes.
func softwareUpdateSettingsSchemaV2() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"allow_standard_user_os_updates":           schema.BoolAttribute{Optional: true},
		"automatic_download":                       schema.StringAttribute{Optional: true},
		"automatic_install_os_updates":             schema.StringAttribute{Optional: true},
		"automatic_install_security_updates":       schema.StringAttribute{Optional: true},
		"beta_program_enrollment":                  schema.StringAttribute{Optional: true},
		"beta_require_program_token":               schema.StringAttribute{Optional: true},
		"beta_require_program_description":         schema.StringAttribute{Optional: true},
		"deferral_combined_period_days":            schema.StringAttribute{Optional: true},
		"deferral_major_period_days":               schema.StringAttribute{Optional: true},
		"deferral_minor_period_days":               schema.StringAttribute{Optional: true},
		"deferral_system_period_days":              schema.StringAttribute{Optional: true},
		"notifications_enabled":                    schema.BoolAttribute{Optional: true},
		"rapid_security_response_enabled":          schema.BoolAttribute{Optional: true},
		"rapid_security_response_rollback_enabled": schema.BoolAttribute{Optional: true},
		"recommended_cadence":                      schema.StringAttribute{Optional: true},
		"beta_offer_programs": schema.SetNestedAttribute{
			Optional: true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"token":       schema.StringAttribute{Required: true},
					"description": schema.StringAttribute{Required: true},
				},
			},
		},
	}
}
