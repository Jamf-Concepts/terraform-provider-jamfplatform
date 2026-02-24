// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"encoding/json"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
func SafariSettingsComponentSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"accept_cookies": schema.StringAttribute{
			MarkdownDescription: "The policy Safari uses for managing cookies. Valid values are `Never`, `CurrentWebsite`, `VisitedWebsites`, `Always`.",
			Optional:            true,
		},
		"allow_disabling_fraud_warning": schema.BoolAttribute{
			MarkdownDescription: "If false, the system forces fraud warnings on in Safari.",
			Optional:            true,
		},
		"allow_history_clearing": schema.BoolAttribute{
			MarkdownDescription: "If false, the system disables clearing history in Safari.",
			Optional:            true,
		},
		"allow_javascript": schema.BoolAttribute{
			MarkdownDescription: "If false, the system disables JavaScript in Safari.",
			Optional:            true,
		},
		"allow_private_browsing": schema.BoolAttribute{
			MarkdownDescription: "If false, the system disables private browsing in Safari.",
			Optional:            true,
		},
		"allow_popups": schema.BoolAttribute{
			MarkdownDescription: "If false, the system disables popups in Safari.",
			Optional:            true,
		},
		"allow_summary": schema.BoolAttribute{
			MarkdownDescription: "If false, the system disables summarization of content in Safari.",
			Optional:            true,
		},
		"new_tab_start_page_type": schema.StringAttribute{
			MarkdownDescription: "Sets the start page type in Safari. Valid values are `Start`, `Home`, `Extension`.",
			Optional:            true,
			Validators:          []validator.String{stringvalidator.OneOf("Start", "Home", "Extension")},
		},
		"new_tab_start_page_homepage_url": schema.StringAttribute{
			MarkdownDescription: "The URL of the homepage which needs to start with `https://` or `http://`. Required when page type is `Home`.",
			Optional:            true,
		},
		"new_tab_start_page_extension_id": schema.StringAttribute{
			MarkdownDescription: "The composed identifier of the extension that provides the start page. Required when page type is `Extension`. Format: `com.example.extension (ABC1234567)`.",
			Optional:            true,
		},
	}
}

// ToRawConfiguration converts the typed component to raw configuration matching OpenAPI SafariSettingsConfiguration schema
func (c *SafariSettingsComponent) ToRawConfiguration() (map[string]any, error) {
	config := make(map[string]any)

	if helpers.IsConfiguredValue(c.AcceptCookies) {
		config["AcceptCookies"] = setStringField(c.AcceptCookies, "")
	}

	if helpers.IsConfiguredValue(c.AllowDisablingFraudWarning) {
		config["AllowDisablingFraudWarning"] = setBoolFieldWithKey(c.AllowDisablingFraudWarning, "Value", false)
	}

	if helpers.IsConfiguredValue(c.AllowHistoryClearing) {
		config["AllowHistoryClearing"] = setBoolFieldWithKey(c.AllowHistoryClearing, "Value", false)
	}

	if helpers.IsConfiguredValue(c.AllowJavaScript) {
		config["AllowJavaScript"] = setBoolFieldWithKey(c.AllowJavaScript, "Value", false)
	}

	if helpers.IsConfiguredValue(c.AllowPrivateBrowsing) {
		config["AllowPrivateBrowsing"] = setBoolFieldWithKey(c.AllowPrivateBrowsing, "Value", false)
	}

	if helpers.IsConfiguredValue(c.AllowPopups) {
		config["AllowPopups"] = setBoolFieldWithKey(c.AllowPopups, "Value", false)
	}

	if helpers.IsConfiguredValue(c.AllowSummary) {
		config["AllowSummary"] = setBoolFieldWithKey(c.AllowSummary, "Value", false)
	}

	if helpers.IsConfiguredValue(c.NewTabStartPageType) ||
		helpers.IsConfiguredValue(c.NewTabStartPageHomepageURL) ||
		helpers.IsConfiguredValue(c.NewTabStartPageExtensionID) {

		newTabStartPage := map[string]any{
			"Included": true,
		}

		if helpers.IsConfiguredValue(c.NewTabStartPageType) {
			newTabStartPage["PageType"] = c.NewTabStartPageType.ValueString()
		}

		if helpers.IsConfiguredValue(c.NewTabStartPageHomepageURL) {
			newTabStartPage["HomepageURL"] = c.NewTabStartPageHomepageURL.ValueString()
		}

		if helpers.IsConfiguredValue(c.NewTabStartPageExtensionID) {
			newTabStartPage["ExtensionIdentifier"] = c.NewTabStartPageExtensionID.ValueString()
		}

		config["NewTabStartPage"] = newTabStartPage
	}

	return config, nil
}

// FromRawConfiguration populates the typed component from raw configuration data
func (c *SafariSettingsComponent) FromRawConfiguration(raw map[string]any) error {
	if acceptCookiesRaw, exists := raw["AcceptCookies"]; exists {
		if acceptCookiesMap, ok := acceptCookiesRaw.(map[string]any); ok {
			if value, exists := acceptCookiesMap["Value"]; exists {
				if valueStr, ok := value.(string); ok {
					c.AcceptCookies = types.StringValue(valueStr)
				}
			}
		}
	}

	if allowDisablingFraudWarningRaw, exists := raw["AllowDisablingFraudWarning"]; exists {
		if allowDisablingFraudWarningMap, ok := allowDisablingFraudWarningRaw.(map[string]any); ok {
			if value, exists := allowDisablingFraudWarningMap["Value"]; exists {
				if valueBool, ok := value.(bool); ok {
					c.AllowDisablingFraudWarning = types.BoolValue(valueBool)
				}
			}
		}
	}

	if allowHistoryClearingRaw, exists := raw["AllowHistoryClearing"]; exists {
		if allowHistoryClearingMap, ok := allowHistoryClearingRaw.(map[string]any); ok {
			if value, exists := allowHistoryClearingMap["Value"]; exists {
				if valueBool, ok := value.(bool); ok {
					c.AllowHistoryClearing = types.BoolValue(valueBool)
				}
			}
		}
	}

	if allowJavaScriptRaw, exists := raw["AllowJavaScript"]; exists {
		if allowJavaScriptMap, ok := allowJavaScriptRaw.(map[string]any); ok {
			if value, exists := allowJavaScriptMap["Value"]; exists {
				if valueBool, ok := value.(bool); ok {
					c.AllowJavaScript = types.BoolValue(valueBool)
				}
			}
		}
	}

	if allowPrivateBrowsingRaw, exists := raw["AllowPrivateBrowsing"]; exists {
		if allowPrivateBrowsingMap, ok := allowPrivateBrowsingRaw.(map[string]any); ok {
			if value, exists := allowPrivateBrowsingMap["Value"]; exists {
				if valueBool, ok := value.(bool); ok {
					c.AllowPrivateBrowsing = types.BoolValue(valueBool)
				}
			}
		}
	}

	if allowPopupsRaw, exists := raw["AllowPopups"]; exists {
		if allowPopupsMap, ok := allowPopupsRaw.(map[string]any); ok {
			if value, exists := allowPopupsMap["Value"]; exists {
				if valueBool, ok := value.(bool); ok {
					c.AllowPopups = types.BoolValue(valueBool)
				}
			}
		}
	}

	if allowSummaryRaw, exists := raw["AllowSummary"]; exists {
		if allowSummaryMap, ok := allowSummaryRaw.(map[string]any); ok {
			if value, exists := allowSummaryMap["Value"]; exists {
				if valueBool, ok := value.(bool); ok {
					c.AllowSummary = types.BoolValue(valueBool)
				}
			}
		}
	}

	if newTabStartPageRaw, exists := raw["NewTabStartPage"]; exists {
		if newTabStartPageMap, ok := newTabStartPageRaw.(map[string]any); ok {
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
