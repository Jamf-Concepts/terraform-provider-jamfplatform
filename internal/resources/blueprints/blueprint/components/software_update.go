// Copyright 2025 Jamf Software LLC.

package components

import (
	"encoding/json"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SoftwareUpdateComponent represents a strongly-typed software update enforcement component
type SoftwareUpdateComponent struct {
	TargetOSVersion     types.String `tfsdk:"target_os_version"`
	TargetLocalDateTime types.String `tfsdk:"target_local_date_time"`
	DetailsURLValue     types.String `tfsdk:"details_url_value"`
}

// SoftwareUpdateComponentSchema returns the Terraform schema for software update component
func SoftwareUpdateComponentSchema() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"target_os_version": schema.StringAttribute{
				Description: "Target OS version. Format: major.minor[.patch]",
				Required:    true,
				Validators: []validator.String{stringvalidator.RegexMatches(
					regexp.MustCompile(`^\d+\.\d+(\.\d+)?$`),
					"Value must be a valid semantic version (e.g., 10.15.7)",
				)},
			},
			"target_local_date_time": schema.StringAttribute{
				Description: "Local time of the device until which update must be performed. Format: RFC3339 date-time.",
				Required:    true,
				Validators: []validator.String{stringvalidator.RegexMatches(
					regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}$`),
					"Value must be a valid RFC3339 date-time (e.g., 2023-10-05T14:48:00)",
				)},
			},
			"details_url_value": schema.StringAttribute{
				Description: "URL of a web page with the details about the enforced update.",
				Optional:    true,
			},
		},
	}
}

// ToRawConfiguration converts the strongly-typed component to raw key-value configuration
func (c *SoftwareUpdateComponent) ToRawConfiguration() (map[string]interface{}, error) {
	config := make(map[string]interface{})

	if !c.TargetOSVersion.IsNull() && !c.TargetOSVersion.IsUnknown() {
		config["targetOSVersion"] = c.TargetOSVersion.ValueString()
	}

	if !c.TargetLocalDateTime.IsNull() && !c.TargetLocalDateTime.IsUnknown() {
		config["targetLocalDateTime"] = c.TargetLocalDateTime.ValueString()
	}

	detailsURL := map[string]interface{}{
		"Included": false,
		"Value":    "",
	}

	if !c.DetailsURLValue.IsNull() && !c.DetailsURLValue.IsUnknown() && c.DetailsURLValue.ValueString() != "" {
		detailsURL["Included"] = true
		detailsURL["Value"] = c.DetailsURLValue.ValueString()
	}

	config["detailsURL"] = detailsURL

	return config, nil
}

// FromRawConfiguration populates the strongly-typed component from raw configuration
func (c *SoftwareUpdateComponent) FromRawConfiguration(rawConfig map[string]interface{}) error {
	if targetOSVersion, exists := rawConfig["targetOSVersion"]; exists {
		if targetOSVersionStr, ok := targetOSVersion.(string); ok {
			c.TargetOSVersion = types.StringValue(targetOSVersionStr)
		}
	}

	if targetLocalDateTime, exists := rawConfig["targetLocalDateTime"]; exists {
		if targetLocalDateTimeStr, ok := targetLocalDateTime.(string); ok {
			c.TargetLocalDateTime = types.StringValue(targetLocalDateTimeStr)
		}
	}

	if detailsURL, exists := rawConfig["detailsURL"]; exists {
		if detailsURLMap, ok := detailsURL.(map[string]interface{}); ok {
			if value, exists := detailsURLMap["Value"]; exists {
				if valueStr, ok := value.(string); ok && valueStr != "" {
					c.DetailsURLValue = types.StringValue(valueStr)
				}
			}
		}
	}

	return nil
}

// GetIdentifier returns the component identifier for software update
func (c *SoftwareUpdateComponent) GetIdentifier() string {
	return "com.jamf.ddm.sw-updates"
}

// ToClientComponent converts the strongly-typed component to a client.BlueprintComponent
func (c *SoftwareUpdateComponent) ToClientComponent() (*BlueprintComponentData, error) {
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
