// Copyright 2025 Jamf Software LLC.

package components

import (
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SafariSettingsComponent represents a strongly-typed Safari settings component
type SafariSettingsComponent struct {
	AcceptCookies              types.String `tfsdk:"accept_cookies"`
	AllowDisablingFraudWarning types.Bool   `tfsdk:"allow_disabling_fraud_warning"`
	AllowHistoryClearing       types.Bool   `tfsdk:"allow_history_clearing"`
	AllowJavaScript            types.Bool   `tfsdk:"allow_javascript"`
	AllowPrivateBrowsing       types.Bool   `tfsdk:"allow_private_browsing"`
	AllowPopups                types.Bool   `tfsdk:"allow_popups"`
	AllowSummary               types.Bool   `tfsdk:"allow_summary"`
	NewTabStartPageType        types.String `tfsdk:"new_tab_start_page_type"`
	NewTabStartPageHomepageURL types.String `tfsdk:"new_tab_start_page_homepage_url"`
	NewTabStartPageExtensionID types.String `tfsdk:"new_tab_start_page_extension_id"`
}

// GetIdentifier returns the component identifier for Safari settings
func (c *SafariSettingsComponent) GetIdentifier() string {
	return "com.jamf.ddm.safari-settings"
}

// SafariSettingsComponentSchema returns the Terraform schema for Safari settings component
func SafariSettingsComponentSchema() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"accept_cookies": schema.StringAttribute{
				Description: "The policy Safari uses for managing cookies. Valid values: Never, CurrentWebsite, VisitedWebsites, Always.",
				Optional:    true,
			},
			"allow_disabling_fraud_warning": schema.BoolAttribute{
				Description: "If false, the system forces fraud warnings on in Safari.",
				Optional:    true,
			},
			"allow_history_clearing": schema.BoolAttribute{
				Description: "If false, the system disables clearing history in Safari.",
				Optional:    true,
			},
			"allow_javascript": schema.BoolAttribute{
				Description: "If false, the system disables JavaScript in Safari.",
				Optional:    true,
			},
			"allow_private_browsing": schema.BoolAttribute{
				Description: "If false, the system disables private browsing in Safari.",
				Optional:    true,
			},
			"allow_popups": schema.BoolAttribute{
				Description: "If false, the system disables popups in Safari.",
				Optional:    true,
			},
			"allow_summary": schema.BoolAttribute{
				Description: "If false, the system disables summarization of content in Safari.",
				Optional:    true,
			},
			"new_tab_start_page_type": schema.StringAttribute{
				Description: "Sets the start page type in Safari. Valid values: Start, Home, Extension.",
				Optional:    true,
			},
			"new_tab_start_page_homepage_url": schema.StringAttribute{
				Description: "The URL of the homepage which needs to start with https:// or http://. Required when page type is 'Home'.",
				Optional:    true,
			},
			"new_tab_start_page_extension_id": schema.StringAttribute{
				Description: "The composed identifier of the extension that provides the start page. Required when page type is 'Extension'. Format: com.example.extension (ABC1234567).",
				Optional:    true,
			},
		},
	}
}

// ToRawConfiguration converts the typed component to raw configuration matching OpenAPI SafariSettingsConfiguration schema
func (c *SafariSettingsComponent) ToRawConfiguration() (map[string]interface{}, error) {
	config := make(map[string]interface{})

	if !c.AcceptCookies.IsNull() && !c.AcceptCookies.IsUnknown() {
		config["AcceptCookies"] = map[string]interface{}{
			"Value":    c.AcceptCookies.ValueString(),
			"Included": true,
		}
	}

	if !c.AllowDisablingFraudWarning.IsNull() && !c.AllowDisablingFraudWarning.IsUnknown() {
		config["AllowDisablingFraudWarning"] = map[string]interface{}{
			"Value":    c.AllowDisablingFraudWarning.ValueBool(),
			"Included": true,
		}
	}

	if !c.AllowHistoryClearing.IsNull() && !c.AllowHistoryClearing.IsUnknown() {
		config["AllowHistoryClearing"] = map[string]interface{}{
			"Value":    c.AllowHistoryClearing.ValueBool(),
			"Included": true,
		}
	}

	if !c.AllowJavaScript.IsNull() && !c.AllowJavaScript.IsUnknown() {
		config["AllowJavaScript"] = map[string]interface{}{
			"Value":    c.AllowJavaScript.ValueBool(),
			"Included": true,
		}
	}

	if !c.AllowPrivateBrowsing.IsNull() && !c.AllowPrivateBrowsing.IsUnknown() {
		config["AllowPrivateBrowsing"] = map[string]interface{}{
			"Value":    c.AllowPrivateBrowsing.ValueBool(),
			"Included": true,
		}
	}

	if !c.AllowPopups.IsNull() && !c.AllowPopups.IsUnknown() {
		config["AllowPopups"] = map[string]interface{}{
			"Value":    c.AllowPopups.ValueBool(),
			"Included": true,
		}
	}

	if !c.AllowSummary.IsNull() && !c.AllowSummary.IsUnknown() {
		config["AllowSummary"] = map[string]interface{}{
			"Value":    c.AllowSummary.ValueBool(),
			"Included": true,
		}
	}

	if (!c.NewTabStartPageType.IsNull() && !c.NewTabStartPageType.IsUnknown()) ||
		(!c.NewTabStartPageHomepageURL.IsNull() && !c.NewTabStartPageHomepageURL.IsUnknown()) ||
		(!c.NewTabStartPageExtensionID.IsNull() && !c.NewTabStartPageExtensionID.IsUnknown()) {

		newTabStartPage := map[string]interface{}{
			"Included": true,
		}

		if !c.NewTabStartPageType.IsNull() && !c.NewTabStartPageType.IsUnknown() {
			newTabStartPage["PageType"] = c.NewTabStartPageType.ValueString()
		}

		if !c.NewTabStartPageHomepageURL.IsNull() && !c.NewTabStartPageHomepageURL.IsUnknown() {
			newTabStartPage["HomepageURL"] = c.NewTabStartPageHomepageURL.ValueString()
		}

		if !c.NewTabStartPageExtensionID.IsNull() && !c.NewTabStartPageExtensionID.IsUnknown() {
			newTabStartPage["ExtensionIdentifier"] = c.NewTabStartPageExtensionID.ValueString()
		}

		config["NewTabStartPage"] = newTabStartPage
	}

	return config, nil
}

// FromRawConfiguration populates the typed component from raw configuration data
func (c *SafariSettingsComponent) FromRawConfiguration(raw map[string]interface{}) error {
	if acceptCookiesRaw, exists := raw["AcceptCookies"]; exists {
		if acceptCookiesMap, ok := acceptCookiesRaw.(map[string]interface{}); ok {
			if value, exists := acceptCookiesMap["Value"]; exists {
				if valueStr, ok := value.(string); ok {
					c.AcceptCookies = types.StringValue(valueStr)
				}
			}
		}
	}

	if allowDisablingFraudWarningRaw, exists := raw["AllowDisablingFraudWarning"]; exists {
		if allowDisablingFraudWarningMap, ok := allowDisablingFraudWarningRaw.(map[string]interface{}); ok {
			if value, exists := allowDisablingFraudWarningMap["Value"]; exists {
				if valueBool, ok := value.(bool); ok {
					c.AllowDisablingFraudWarning = types.BoolValue(valueBool)
				}
			}
		}
	}

	if allowHistoryClearingRaw, exists := raw["AllowHistoryClearing"]; exists {
		if allowHistoryClearingMap, ok := allowHistoryClearingRaw.(map[string]interface{}); ok {
			if value, exists := allowHistoryClearingMap["Value"]; exists {
				if valueBool, ok := value.(bool); ok {
					c.AllowHistoryClearing = types.BoolValue(valueBool)
				}
			}
		}
	}

	if allowJavaScriptRaw, exists := raw["AllowJavaScript"]; exists {
		if allowJavaScriptMap, ok := allowJavaScriptRaw.(map[string]interface{}); ok {
			if value, exists := allowJavaScriptMap["Value"]; exists {
				if valueBool, ok := value.(bool); ok {
					c.AllowJavaScript = types.BoolValue(valueBool)
				}
			}
		}
	}

	if allowPrivateBrowsingRaw, exists := raw["AllowPrivateBrowsing"]; exists {
		if allowPrivateBrowsingMap, ok := allowPrivateBrowsingRaw.(map[string]interface{}); ok {
			if value, exists := allowPrivateBrowsingMap["Value"]; exists {
				if valueBool, ok := value.(bool); ok {
					c.AllowPrivateBrowsing = types.BoolValue(valueBool)
				}
			}
		}
	}

	if allowPopupsRaw, exists := raw["AllowPopups"]; exists {
		if allowPopupsMap, ok := allowPopupsRaw.(map[string]interface{}); ok {
			if value, exists := allowPopupsMap["Value"]; exists {
				if valueBool, ok := value.(bool); ok {
					c.AllowPopups = types.BoolValue(valueBool)
				}
			}
		}
	}

	if allowSummaryRaw, exists := raw["AllowSummary"]; exists {
		if allowSummaryMap, ok := allowSummaryRaw.(map[string]interface{}); ok {
			if value, exists := allowSummaryMap["Value"]; exists {
				if valueBool, ok := value.(bool); ok {
					c.AllowSummary = types.BoolValue(valueBool)
				}
			}
		}
	}

	if newTabStartPageRaw, exists := raw["NewTabStartPage"]; exists {
		if newTabStartPageMap, ok := newTabStartPageRaw.(map[string]interface{}); ok {
			if pageType, exists := newTabStartPageMap["PageType"]; exists {
				if pageTypeStr, ok := pageType.(string); ok {
					c.NewTabStartPageType = types.StringValue(pageTypeStr)
				}
			}
			if homepageURL, exists := newTabStartPageMap["HomepageURL"]; exists {
				if homepageURLStr, ok := homepageURL.(string); ok {
					c.NewTabStartPageHomepageURL = types.StringValue(homepageURLStr)
				}
			}
			if extensionID, exists := newTabStartPageMap["ExtensionIdentifier"]; exists {
				if extensionIDStr, ok := extensionID.(string); ok {
					c.NewTabStartPageExtensionID = types.StringValue(extensionIDStr)
				}
			}
		}
	}

	return nil
}

// ToClientComponent converts the typed component to the format expected by the Blueprint API client
func (c *SafariSettingsComponent) ToClientComponent() (*BlueprintComponentData, error) {
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
