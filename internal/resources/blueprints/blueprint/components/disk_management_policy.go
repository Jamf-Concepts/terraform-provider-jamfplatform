// Copyright 2025 Jamf Software LLC.

package components

import (
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DiskManagementPolicyComponent represents a strongly-typed disk management policy component
type DiskManagementPolicyComponent struct {
	ExternalStorage types.String `tfsdk:"external_storage"`
	NetworkStorage  types.String `tfsdk:"network_storage"`
}

// GetIdentifier returns the component identifier for disk management policy
func (c *DiskManagementPolicyComponent) GetIdentifier() string {
	return "com.jamf.ddm.disk-management"
}

// DiskManagementPolicyComponentSchema returns the Terraform schema for disk management policy component
func DiskManagementPolicyComponentSchema() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"external_storage": schema.StringAttribute{
				Description: "Storage mode for external storage. Valid values: Allowed, Disallowed, ReadOnly.",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.OneOf("Allowed", "Disallowed", "ReadOnly")},
			},
			"network_storage": schema.StringAttribute{
				Description: "Storage mode for network storage. Valid values: Allowed, Disallowed, ReadOnly.",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.OneOf("Allowed", "Disallowed", "ReadOnly")},
			},
		},
	}
}

// ToRawConfiguration converts the typed component to raw configuration matching OpenAPI DiskManagementPolicyConfiguration schema
func (c *DiskManagementPolicyComponent) ToRawConfiguration() (map[string]interface{}, error) {
	config := make(map[string]interface{})

	config["version"] = int32(2)

	if (!c.ExternalStorage.IsNull() && !c.ExternalStorage.IsUnknown()) ||
		(!c.NetworkStorage.IsNull() && !c.NetworkStorage.IsUnknown()) {

		restrictions := make(map[string]interface{})

		if !c.ExternalStorage.IsNull() && !c.ExternalStorage.IsUnknown() {
			restrictions["ExternalStorage"] = map[string]interface{}{
				"Value":    c.ExternalStorage.ValueString(),
				"Included": true,
			}
		}

		if !c.NetworkStorage.IsNull() && !c.NetworkStorage.IsUnknown() {
			restrictions["NetworkStorage"] = map[string]interface{}{
				"Value":    c.NetworkStorage.ValueString(),
				"Included": true,
			}
		}

		config["Restrictions"] = restrictions
	}

	return config, nil
}

// FromRawConfiguration populates the typed component from raw configuration data
func (c *DiskManagementPolicyComponent) FromRawConfiguration(raw map[string]interface{}) error {
	if restrictionsRaw, exists := raw["Restrictions"]; exists {
		if restrictionsMap, ok := restrictionsRaw.(map[string]interface{}); ok {
			if externalStorageRaw, exists := restrictionsMap["ExternalStorage"]; exists {
				if externalStorageStr, ok := externalStorageRaw.(string); ok {
					c.ExternalStorage = types.StringValue(externalStorageStr)
				} else if externalStorageMap, ok := externalStorageRaw.(map[string]interface{}); ok {
					if value, exists := externalStorageMap["Value"]; exists {
						if valueStr, ok := value.(string); ok {
							c.ExternalStorage = types.StringValue(valueStr)
						}
					}
				}
			}

			if networkStorageRaw, exists := restrictionsMap["NetworkStorage"]; exists {
				if networkStorageStr, ok := networkStorageRaw.(string); ok {
					c.NetworkStorage = types.StringValue(networkStorageStr)
				} else if networkStorageMap, ok := networkStorageRaw.(map[string]interface{}); ok {
					if value, exists := networkStorageMap["Value"]; exists {
						if valueStr, ok := value.(string); ok {
							c.NetworkStorage = types.StringValue(valueStr)
						}
					}
				}
			}
		}
	}

	return nil
}

// ToClientComponent converts the typed component to the format expected by the Blueprint API client
func (c *DiskManagementPolicyComponent) ToClientComponent() (*BlueprintComponentData, error) {
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
