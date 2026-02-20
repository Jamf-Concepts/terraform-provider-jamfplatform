// Copyright 2025 Jamf Software LLC.

package components

import (
	"encoding/json"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ServiceConfigurationFilesComponent represents a strongly-typed service configuration files component
type ServiceConfigurationFilesComponent struct {
	ServiceConfigFiles []ServiceConfigFileModel `tfsdk:"service_config_files"`
}

// ServiceConfigDataAssetRefModel represents a data asset reference for service configuration files
type ServiceConfigDataAssetRefModel struct {
	DataURL     types.String `tfsdk:"data_url"`
	HashSHA256  types.String `tfsdk:"hash_sha_256"`
	ContentType types.String `tfsdk:"content_type"`
}

// ServiceConfigFileModel represents a service configuration file
type ServiceConfigFileModel struct {
	ServiceType        types.String                    `tfsdk:"service_type"`
	DataAssetReference *ServiceConfigDataAssetRefModel `tfsdk:"data_asset_reference"`
}

// ServiceConfigurationFilesComponentSchema returns the Terraform schema for service configuration files component
func ServiceConfigurationFilesComponentSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"service_config_files": schema.SetNestedAttribute{
			MarkdownDescription: "Set of service configuration files.",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"service_type": schema.StringAttribute{
						MarkdownDescription: "The identifier of the system service with managed configuration files.",
						Required:            true,
					},
					"data_asset_reference": schema.SingleNestedAttribute{
						MarkdownDescription: "Reference to the configuration data asset.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"data_url": schema.StringAttribute{
								MarkdownDescription: "URL that hosts the configuration data.",
								Required:            true,
							},
							"hash_sha_256": schema.StringAttribute{
								MarkdownDescription: "SHA-256 hash of the data.",
								Optional:            true,
							},
							"content_type": schema.StringAttribute{
								MarkdownDescription: "Media type of the data. Always `application/zip` for service configuration files.",
								Computed:            true,
							},
						},
					},
				},
			},
		},
	}
}

// ToRawConfiguration converts the typed component to raw configuration matching OpenAPI ServicesConfigurationFilesConfiguration schema
func (c *ServiceConfigurationFilesComponent) ToRawConfiguration() (map[string]any, error) {
	config := make(map[string]any)

	if len(c.ServiceConfigFiles) > 0 {
		serviceConfigFiles := make([]any, 0, len(c.ServiceConfigFiles))

		for _, configFile := range c.ServiceConfigFiles {
			configFileMap := make(map[string]any)

			if helpers.IsConfiguredValue(configFile.ServiceType) {
				configFileMap["ServiceType"] = configFile.ServiceType.ValueString()
			}

			if configFile.DataAssetReference != nil {
				dataRef := make(map[string]any)

				reference := make(map[string]any)
				if helpers.IsConfiguredValue(configFile.DataAssetReference.DataURL) {
					reference["DataURL"] = configFile.DataAssetReference.DataURL.ValueString()
				}
				if helpers.IsConfiguredValue(configFile.DataAssetReference.HashSHA256) {
					reference["Hash-SHA-256"] = configFile.DataAssetReference.HashSHA256.ValueString()
				}
				reference["ContentType"] = "application/zip"

				dataRef["Reference"] = reference

				configFileMap["DataAssetReference"] = dataRef
			}

			serviceConfigFiles = append(serviceConfigFiles, configFileMap)
		}

		config["serviceConfigFiles"] = serviceConfigFiles
	}

	return config, nil
}

// FromRawConfiguration populates the typed component from raw configuration data
func (c *ServiceConfigurationFilesComponent) FromRawConfiguration(raw map[string]any) error {
	if serviceConfigFilesRaw, exists := raw["serviceConfigFiles"]; exists {
		if serviceConfigFilesSlice, ok := serviceConfigFilesRaw.([]any); ok {
			serviceConfigFiles := make([]ServiceConfigFileModel, 0, len(serviceConfigFilesSlice))

			for _, configFileRaw := range serviceConfigFilesSlice {
				if configFileMap, ok := configFileRaw.(map[string]any); ok {
					configFile := ServiceConfigFileModel{}

					if serviceType, exists := configFileMap["ServiceType"]; exists {
						if serviceTypeStr, ok := serviceType.(string); ok {
							configFile.ServiceType = types.StringValue(serviceTypeStr)
						}
					}

					if dataAssetRefRaw, exists := configFileMap["DataAssetReference"]; exists {
						if dataAssetRefMap, ok := dataAssetRefRaw.(map[string]any); ok {
							if refRaw, exists := dataAssetRefMap["Reference"]; exists {
								if refMap, ok := refRaw.(map[string]any); ok {
									dataAssetRef := &ServiceConfigDataAssetRefModel{}

									if dataURL, exists := refMap["DataURL"]; exists {
										if dataURLStr, ok := dataURL.(string); ok {
											dataAssetRef.DataURL = types.StringValue(dataURLStr)
										}
									}

									if hashSHA256, exists := refMap["Hash-SHA-256"]; exists {
										if hashStr, ok := hashSHA256.(string); ok {
											dataAssetRef.HashSHA256 = types.StringValue(hashStr)
										}
									}

									dataAssetRef.ContentType = types.StringValue("application/zip")

									configFile.DataAssetReference = dataAssetRef
								}
							}
						}
					}

					serviceConfigFiles = append(serviceConfigFiles, configFile)
				}
			}

			c.ServiceConfigFiles = serviceConfigFiles
		}
	}

	return nil
}

// GetIdentifier returns the component identifier for service configuration files
func (c *ServiceConfigurationFilesComponent) GetIdentifier() string {
	return "com.jamf.ddm.service-configuration-files"
}

// ToClientComponent converts the typed component to the format expected by the Blueprint API client
func (c *ServiceConfigurationFilesComponent) ToClientComponent() (*BlueprintComponentData, error) {
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
