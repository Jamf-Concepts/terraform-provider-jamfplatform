// Copyright 2025 Jamf Software LLC.

package components

import (
	"encoding/json"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SoftwareUpdateComponent represents a strongly-typed software update enforcement component
type SoftwareUpdateComponent struct {
	EnforcementType     types.String `tfsdk:"enforcement_type"`
	DeploymentTime      types.String `tfsdk:"deployment_time"`
	EnforceAfterDays    types.Int64  `tfsdk:"enforce_after_days"`
	TargetOSVersion     types.String `tfsdk:"target_os_version"`
	TargetLocalDateTime types.String `tfsdk:"target_local_date_time"`
	DetailsURLValue     types.String `tfsdk:"details_url_value"`
}

// SoftwareUpdateComponentSchema returns the Terraform schema for software update component
func SoftwareUpdateComponentSchema() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"enforcement_type": schema.StringAttribute{
				Description: "Type of enforcement. Automatically set to 'AUTOMATIC' when deployment_time or enforce_after_days is specified, or 'MANUAL' when target_os_version or target_local_date_time is specified.",
				Computed:    true,
			},
			"deployment_time": schema.StringAttribute{
				Description: "For automatic enforcement. Local device time to install the update. Format: HH:mm (24-hour). Cannot be used with target_os_version or target_local_date_time.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^(?:[01]\d|2[0-3]):[0-5]\d$`),
						"Value must be in HH:mm format (e.g., 14:30)",
					),
					stringvalidator.AlsoRequires(
						path.MatchRelative().AtParent().AtName("enforce_after_days"),
					),
					stringvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("target_os_version"),
						path.MatchRelative().AtParent().AtName("target_local_date_time"),
					),
				},
			},
			"enforce_after_days": schema.Int64Attribute{
				Description: "For automatic enforcement. Days after release to enforce the update. Maximum is 30. Cannot be used with target_os_version or target_local_date_time.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 30),
					int64validator.AlsoRequires(
						path.MatchRelative().AtParent().AtName("deployment_time"),
					),
					int64validator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("target_os_version"),
						path.MatchRelative().AtParent().AtName("target_local_date_time"),
					),
				},
			},
			"target_os_version": schema.StringAttribute{
				Description: "For manual enforcement. Target OS version. Format: major.minor[.patch]. Cannot be used with deployment_time or enforce_after_days.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^\d+\.\d+(\.\d+)?$`),
						"Value must be a valid semantic version (e.g., 10.15.7)",
					),
					stringvalidator.AlsoRequires(
						path.MatchRelative().AtParent().AtName("target_local_date_time"),
					),
					stringvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("deployment_time"),
						path.MatchRelative().AtParent().AtName("enforce_after_days"),
					),
				},
			},
			"target_local_date_time": schema.StringAttribute{
				Description: "For manual enforcement. Local device date and time to enforce the software update. Format: RFC3339 date-time. Cannot be used with deployment_time or enforce_after_days.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}$`),
						"Value must be a valid RFC3339 date-time (e.g., 2023-10-05T14:48:00)",
					),
					stringvalidator.AlsoRequires(
						path.MatchRelative().AtParent().AtName("target_os_version"),
					),
					stringvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("deployment_time"),
						path.MatchRelative().AtParent().AtName("enforce_after_days"),
					),
				},
			},
			"details_url_value": schema.StringAttribute{
				Description: "URL of a web page with the details about the software update.",
				Optional:    true,
			},
		},
	}
}

// ToRawConfiguration converts the strongly-typed component to raw key-value configuration
func (c *SoftwareUpdateComponent) ToRawConfiguration() (map[string]interface{}, error) {
	config := make(map[string]interface{})

	if (!c.DeploymentTime.IsNull() && !c.DeploymentTime.IsUnknown()) ||
		(!c.EnforceAfterDays.IsNull() && !c.EnforceAfterDays.IsUnknown()) {
		config["enforcementType"] = "AUTOMATIC"
	}

	if !c.DeploymentTime.IsNull() && !c.DeploymentTime.IsUnknown() {
		config["deploymentTime"] = c.DeploymentTime.ValueString()
	}

	if !c.EnforceAfterDays.IsNull() && !c.EnforceAfterDays.IsUnknown() {
		config["enforceAfterDays"] = c.EnforceAfterDays.ValueInt64()
	}

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
	if enforcementType, exists := rawConfig["enforcementType"]; exists {
		if enforcementTypeStr, ok := enforcementType.(string); ok {
			c.EnforcementType = types.StringValue(enforcementTypeStr)
		}
	}

	if deploymentTime, exists := rawConfig["deploymentTime"]; exists {
		if deploymentTimeStr, ok := deploymentTime.(string); ok {
			c.DeploymentTime = types.StringValue(deploymentTimeStr)
		}
	}

	if enforceAfterDays, exists := rawConfig["enforceAfterDays"]; exists {
		if enforceAfterDaysFloat, ok := enforceAfterDays.(float64); ok {
			c.EnforceAfterDays = types.Int64Value(int64(enforceAfterDaysFloat))
		}
	}

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

	if c.EnforcementType.IsNull() || c.EnforcementType.IsUnknown() {
		if !c.TargetOSVersion.IsNull() || !c.TargetLocalDateTime.IsNull() {
			c.EnforcementType = types.StringValue("MANUAL")
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
