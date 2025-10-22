// Copyright 2025 Jamf Software LLC.

package components

import (
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AudioAccessorySettingsComponent represents a strongly-typed audio accessory settings component
type AudioAccessorySettingsComponent struct {
	TemporaryPairingDisabled types.Bool   `tfsdk:"temporary_pairing_disabled"`
	UnpairingTimePolicy      types.String `tfsdk:"unpairing_time_policy"`
	UnpairingTimeHour        types.Int64  `tfsdk:"unpairing_time_hour"`
}

// GetIdentifier returns the component identifier for audio accessory settings
func (c *AudioAccessorySettingsComponent) GetIdentifier() string {
	return "com.jamf.ddm.audio-accessory-settings"
}

// AudioAccessorySettingsComponentSchema returns the Terraform schema for audio accessory settings component
func AudioAccessorySettingsComponentSchema() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"temporary_pairing_disabled": schema.BoolAttribute{
				Description: "If true, temporary pairing of audio accessories is disabled.",
				Required:    true,
			},
			"unpairing_time_policy": schema.StringAttribute{
				Description: "Device's unpairing policy. Valid values: None, Hour. When set to 'Hour', unpairing_time_hour must also be provided.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("None"),
				Validators: []validator.String{
					stringvalidator.OneOf("None", "Hour"),
				},
			},
			"unpairing_time_hour": schema.Int64Attribute{
				Description: "The local time hour (24-hour clock) when the device automatically unpairs temporarily paired audio accessories. Required when policy is 'Hour'. Range: 0-23.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Validators: []validator.Int64{
					int64validator.Between(0, 23),
				},
			},
		},
	}
}

// ToRawConfiguration converts the typed component to raw configuration matching the actual API format
func (c *AudioAccessorySettingsComponent) ToRawConfiguration() (map[string]interface{}, error) {
	config := make(map[string]interface{})

	hasTemporaryPairing := (!c.TemporaryPairingDisabled.IsNull() && !c.TemporaryPairingDisabled.IsUnknown()) ||
		(!c.UnpairingTimePolicy.IsNull() && !c.UnpairingTimePolicy.IsUnknown()) ||
		(!c.UnpairingTimeHour.IsNull() && !c.UnpairingTimeHour.IsUnknown())

	if hasTemporaryPairing {
		temporaryPairing := make(map[string]interface{})
		temporaryPairing["Included"] = true

		if !c.TemporaryPairingDisabled.IsNull() && !c.TemporaryPairingDisabled.IsUnknown() {
			temporaryPairing["Disabled"] = c.TemporaryPairingDisabled.ValueBool()
		}

		hasUnpairingSettings := (!c.UnpairingTimePolicy.IsNull() && !c.UnpairingTimePolicy.IsUnknown()) ||
			(!c.UnpairingTimeHour.IsNull() && !c.UnpairingTimeHour.IsUnknown())

		if hasUnpairingSettings {
			configuration := make(map[string]interface{})
			unpairingTime := make(map[string]interface{})

			if !c.UnpairingTimePolicy.IsNull() && !c.UnpairingTimePolicy.IsUnknown() {
				unpairingTime["Policy"] = c.UnpairingTimePolicy.ValueString()
			}

			if !c.UnpairingTimeHour.IsNull() && !c.UnpairingTimeHour.IsUnknown() {
				unpairingTime["Hour"] = int(c.UnpairingTimeHour.ValueInt64())
			}

			configuration["UnpairingTime"] = unpairingTime
			temporaryPairing["Configuration"] = configuration
		}

		config["TemporaryPairing"] = temporaryPairing
	}

	return config, nil
}

// FromRawConfiguration populates the typed component from raw configuration data
func (c *AudioAccessorySettingsComponent) FromRawConfiguration(raw map[string]interface{}) error {
	c.TemporaryPairingDisabled = types.BoolNull()
	c.UnpairingTimePolicy = types.StringNull()
	c.UnpairingTimeHour = types.Int64Null()

	if temporaryPairingRaw, exists := raw["TemporaryPairing"]; exists {
		if temporaryPairing, ok := temporaryPairingRaw.(map[string]interface{}); ok {
			if included, hasIncluded := temporaryPairing["Included"]; hasIncluded && included.(bool) {

				if disabled, exists := temporaryPairing["Disabled"]; exists {
					switch v := disabled.(type) {
					case bool:
						c.TemporaryPairingDisabled = types.BoolValue(v)
					case string:
						switch v {
						case "true":
							c.TemporaryPairingDisabled = types.BoolValue(true)
						case "false":
							c.TemporaryPairingDisabled = types.BoolValue(false)
						}
					}
				}

				if configRaw, exists := temporaryPairing["Configuration"]; exists {
					if config, ok := configRaw.(map[string]interface{}); ok {
						if unpairingTimeRaw, exists := config["UnpairingTime"]; exists {
							if unpairingTime, ok := unpairingTimeRaw.(map[string]interface{}); ok {

								if policy, exists := unpairingTime["Policy"]; exists {
									if policyStr, ok := policy.(string); ok {
										c.UnpairingTimePolicy = types.StringValue(policyStr)
									}
								}

								if hour, exists := unpairingTime["Hour"]; exists {
									switch v := hour.(type) {
									case int:
										c.UnpairingTimeHour = types.Int64Value(int64(v))
									case int64:
										c.UnpairingTimeHour = types.Int64Value(v)
									case float64:
										c.UnpairingTimeHour = types.Int64Value(int64(v))
									}
								}
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
func (c *AudioAccessorySettingsComponent) ToClientComponent() (*BlueprintComponentData, error) {
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
