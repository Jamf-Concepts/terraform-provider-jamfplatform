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

	restrictions := map[string]interface{}{
		"ExternalStorage": setStringField(c.ExternalStorage, "Allowed"),
		"NetworkStorage":  setStringField(c.NetworkStorage, "Allowed"),
	}
	config["Restrictions"] = restrictions

	return config, nil
}

// FromRawConfiguration populates the typed component from raw configuration data
func (c *DiskManagementPolicyComponent) FromRawConfiguration(raw map[string]interface{}) error {
	extractValue := func(path ...string) interface{} {
		current := raw
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

	if val := extractValue("Restrictions", "ExternalStorage"); val != nil {
		c.ExternalStorage = types.StringValue(val.(string))
	} else {
		c.ExternalStorage = types.StringNull()
	}

	if val := extractValue("Restrictions", "NetworkStorage"); val != nil {
		c.NetworkStorage = types.StringValue(val.(string))
	} else {
		c.NetworkStorage = types.StringNull()
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
