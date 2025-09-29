// Copyright 2025 Jamf Software LLC.

package components

import (
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SafariExtensionsComponent represents a strongly-typed Safari extensions component
type SafariExtensionsComponent struct {
	ManagedExtensions []ManagedExtensionModel `tfsdk:"managed_extensions"`
}

// ManagedExtensionModel represents a managed Safari extension configuration
type ManagedExtensionModel struct {
	ExtensionID     types.String                  `tfsdk:"extension_id"`
	State           types.String                  `tfsdk:"state"`
	PrivateBrowsing types.String                  `tfsdk:"private_browsing"`
	AllowedDomains  []ManagedExtensionDomainModel `tfsdk:"allowed_domains"`
	DeniedDomains   []ManagedExtensionDomainModel `tfsdk:"denied_domains"`
}

// ManagedExtensionDomainModel represents a domain configuration for managed extensions
type ManagedExtensionDomainModel struct {
	Domain types.String `tfsdk:"domain"`
}

// GetIdentifier returns the component identifier for Safari extensions
func (c *SafariExtensionsComponent) GetIdentifier() string {
	return "com.jamf.ddm.safari-extensions"
}

// SafariExtensionsComponentSchema returns the Terraform schema for Safari extensions component
func SafariExtensionsComponentSchema() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Blocks: map[string]schema.Block{
			"managed_extensions": schema.ListNestedBlock{
				Description: "List of managed Safari extensions.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"extension_id": schema.StringAttribute{
							Description: "The extension identifier (bundle ID).",
							Required:    true,
						},
						"state": schema.StringAttribute{
							Description: "Extension state. Valid values: Allowed, AlwaysOn, AlwaysOff.",
							Optional:    true,
						},
						"private_browsing": schema.StringAttribute{
							Description: "Private browsing state. Valid values: Allowed, AlwaysOn, AlwaysOff.",
							Optional:    true,
						},
					},
					Blocks: map[string]schema.Block{
						"allowed_domains": schema.ListNestedBlock{
							Description: "List of allowed domains for this extension.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"domain": schema.StringAttribute{
										Description: "Domain name.",
										Required:    true,
									},
								},
							},
						},
						"denied_domains": schema.ListNestedBlock{
							Description: "List of denied domains for this extension.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"domain": schema.StringAttribute{
										Description: "Domain name.",
										Required:    true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// ToRawConfiguration converts the typed component to raw configuration matching OpenAPI SafariExtensionsConfiguration schema
func (c *SafariExtensionsComponent) ToRawConfiguration() (map[string]interface{}, error) {
	config := make(map[string]interface{})

	if len(c.ManagedExtensions) > 0 {
		managedExtensions := make(map[string]interface{})

		for _, ext := range c.ManagedExtensions {
			if ext.ExtensionID.IsNull() || ext.ExtensionID.IsUnknown() {
				continue
			}

			extConfig := make(map[string]interface{})

			if !ext.State.IsNull() && !ext.State.IsUnknown() {
				extConfig["State"] = ext.State.ValueString()
			}

			if !ext.PrivateBrowsing.IsNull() && !ext.PrivateBrowsing.IsUnknown() {
				extConfig["PrivateBrowsing"] = ext.PrivateBrowsing.ValueString()
			}

			if len(ext.AllowedDomains) > 0 {
				allowedDomains := make([]interface{}, 0, len(ext.AllowedDomains))
				for _, domain := range ext.AllowedDomains {
					if !domain.Domain.IsNull() && !domain.Domain.IsUnknown() {
						allowedDomains = append(allowedDomains, map[string]interface{}{
							"Domain": domain.Domain.ValueString(),
						})
					}
				}
				if len(allowedDomains) > 0 {
					extConfig["AllowedDomains"] = allowedDomains
				}
			}

			if len(ext.DeniedDomains) > 0 {
				deniedDomains := make([]interface{}, 0, len(ext.DeniedDomains))
				for _, domain := range ext.DeniedDomains {
					if !domain.Domain.IsNull() && !domain.Domain.IsUnknown() {
						deniedDomains = append(deniedDomains, map[string]interface{}{
							"Domain": domain.Domain.ValueString(),
						})
					}
				}
				if len(deniedDomains) > 0 {
					extConfig["DeniedDomains"] = deniedDomains
				}
			}

			managedExtensions[ext.ExtensionID.ValueString()] = extConfig
		}

		config["ManagedExtensions"] = managedExtensions
	}

	return config, nil
}

// FromRawConfiguration populates the typed component from raw configuration data
func (c *SafariExtensionsComponent) FromRawConfiguration(raw map[string]interface{}) error {
	if managedExtensionsRaw, exists := raw["ManagedExtensions"]; exists {
		if managedExtensionsMap, ok := managedExtensionsRaw.(map[string]interface{}); ok {
			extensions := make([]ManagedExtensionModel, 0, len(managedExtensionsMap))

			for extensionID, extConfigRaw := range managedExtensionsMap {
				ext := ManagedExtensionModel{
					ExtensionID: types.StringValue(extensionID),
				}

				if extConfigMap, ok := extConfigRaw.(map[string]interface{}); ok {
					if state, exists := extConfigMap["State"]; exists {
						if stateStr, ok := state.(string); ok {
							ext.State = types.StringValue(stateStr)
						}
					}

					if privateBrowsing, exists := extConfigMap["PrivateBrowsing"]; exists {
						if privateBrowsingStr, ok := privateBrowsing.(string); ok {
							ext.PrivateBrowsing = types.StringValue(privateBrowsingStr)
						}
					}

					if allowedDomainsRaw, exists := extConfigMap["AllowedDomains"]; exists {
						if allowedDomainsSlice, ok := allowedDomainsRaw.([]interface{}); ok {
							allowedDomains := make([]ManagedExtensionDomainModel, 0, len(allowedDomainsSlice))
							for _, domainRaw := range allowedDomainsSlice {
								if domainMap, ok := domainRaw.(map[string]interface{}); ok {
									if domain, exists := domainMap["Domain"]; exists {
										if domainStr, ok := domain.(string); ok {
											allowedDomains = append(allowedDomains, ManagedExtensionDomainModel{
												Domain: types.StringValue(domainStr),
											})
										}
									}
								}
							}
							ext.AllowedDomains = allowedDomains
						}
					}

					if deniedDomainsRaw, exists := extConfigMap["DeniedDomains"]; exists {
						if deniedDomainsSlice, ok := deniedDomainsRaw.([]interface{}); ok {
							deniedDomains := make([]ManagedExtensionDomainModel, 0, len(deniedDomainsSlice))
							for _, domainRaw := range deniedDomainsSlice {
								if domainMap, ok := domainRaw.(map[string]interface{}); ok {
									if domain, exists := domainMap["Domain"]; exists {
										if domainStr, ok := domain.(string); ok {
											deniedDomains = append(deniedDomains, ManagedExtensionDomainModel{
												Domain: types.StringValue(domainStr),
											})
										}
									}
								}
							}
							ext.DeniedDomains = deniedDomains
						}
					}
				}

				extensions = append(extensions, ext)
			}

			c.ManagedExtensions = extensions
		}
	}

	return nil
}

// ToClientComponent converts the typed component to the format expected by the Blueprint API client
func (c *SafariExtensionsComponent) ToClientComponent() (*BlueprintComponentData, error) {
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
