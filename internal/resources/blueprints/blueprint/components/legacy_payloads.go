// Copyright 2025 Jamf Software LLC.

package components

import (
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// LegacyPayloadsComponent represents a strongly-typed legacy configuration profile component
type LegacyPayloadsComponent struct {
	PayloadContent types.String `tfsdk:"payload_content"`
}

// LegacyPayloadsComponentSchema returns the Terraform schema for legacy payloads component
func LegacyPayloadsComponentSchema() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"payload_content": schema.StringAttribute{
				Description: "JSON-encoded array of payload objects. Each payload must have payloadType and payloadIdentifier fields. The payload display name will automatically use the blueprint name.",
				Required:    true,
			},
		},
	}
}

// ToRawConfiguration converts the strongly-typed component to raw key-value configuration
func (c *LegacyPayloadsComponent) ToRawConfiguration() (map[string]interface{}, error) {
	config := make(map[string]interface{})
	if !c.PayloadContent.IsNull() && !c.PayloadContent.IsUnknown() {
		var payloadArray []interface{}
		if err := json.Unmarshal([]byte(c.PayloadContent.ValueString()), &payloadArray); err != nil {
			return nil, err
		}
		config["payloadContent"] = payloadArray
	}

	return config, nil
}

// FromRawConfiguration populates the strongly-typed component from raw configuration
func (c *LegacyPayloadsComponent) FromRawConfiguration(rawConfig map[string]interface{}) error {
	if payloadContent, exists := rawConfig["payloadContent"]; exists {
		payloadJSON, err := json.Marshal(payloadContent)
		if err != nil {
			return err
		}
		c.PayloadContent = types.StringValue(string(payloadJSON))
	}

	return nil
}

// GetIdentifier returns the component identifier for legacy payloads
func (c *LegacyPayloadsComponent) GetIdentifier() string {
	return "com.jamf.ddm-configuration-profile"
}

// ToClientComponent converts the strongly-typed component to a client.BlueprintComponent
func (c *LegacyPayloadsComponent) ToClientComponent() (*BlueprintComponentData, error) {
	return c.ToClientComponentWithName("")
}

// ToClientComponentWithName converts the component to a client.BlueprintComponent with the specified payload display name
func (c *LegacyPayloadsComponent) ToClientComponentWithName(payloadDisplayName string) (*BlueprintComponentData, error) {
	config, err := c.ToRawConfiguration()
	if err != nil {
		return nil, err
	}

	if payloadDisplayName != "" {
		config["payloadDisplayName"] = payloadDisplayName
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
