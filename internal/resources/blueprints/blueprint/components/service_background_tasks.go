// Copyright 2025 Jamf Software LLC.

package components

import (
	"encoding/json"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ServiceBackgroundTasksComponent represents a strongly-typed service background tasks component
type ServiceBackgroundTasksComponent struct {
	BackgroundTasks []ServiceBackgroundTaskModel `tfsdk:"background_tasks"`
}

// ServiceBackgroundTaskModel represents a background task configuration
type ServiceBackgroundTaskModel struct {
	TaskType                 types.String       `tfsdk:"task_type"`
	TaskDescription          types.String       `tfsdk:"task_description"`
	ExecutableAssetReference *DataAssetRefModel `tfsdk:"executable_asset_reference"`
	LaunchdConfigurations    []LaunchdItemModel `tfsdk:"launchd_configurations"`
}

// DataAssetRefModel represents a data asset reference
type DataAssetRefModel struct {
	DataURL     types.String `tfsdk:"data_url"`
	HashSHA256  types.String `tfsdk:"hash_sha_256"`
	ContentType types.String `tfsdk:"content_type"`
}

// LaunchdItemModel represents a launchd configuration item
type LaunchdItemModel struct {
	Context            types.String       `tfsdk:"context"`
	FileAssetReference *DataAssetRefModel `tfsdk:"file_asset_reference"`
}

// ServiceBackgroundTasksComponentSchema returns the Terraform schema for service background tasks component
func ServiceBackgroundTasksComponentSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"background_tasks": schema.SetNestedAttribute{
			MarkdownDescription: "Set of background tasks.",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"task_type": schema.StringAttribute{
						MarkdownDescription: "Task type identifier.",
						Required:            true,
					},
					"task_description": schema.StringAttribute{
						MarkdownDescription: "Task description.",
						Optional:            true,
					},
					"executable_asset_reference": schema.SingleNestedAttribute{
						MarkdownDescription: "Reference to the executable asset.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"data_url": schema.StringAttribute{
								MarkdownDescription: "URL that hosts the executable data.",
								Required:            true,
							},
							"hash_sha_256": schema.StringAttribute{
								MarkdownDescription: "SHA-256 hash of the data.",
								Optional:            true,
							},
							"content_type": schema.StringAttribute{
								MarkdownDescription: "Media type of the data. Always `application/zip` for executable assets.",
								Computed:            true,
							},
						},
					},
					"launchd_configurations": schema.SetNestedAttribute{
						MarkdownDescription: "Launchd configuration items.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"context": schema.StringAttribute{
									MarkdownDescription: "Launchd context. Valid values are `daemon`, `agent`.",
									Required:            true,
									Validators:          []validator.String{stringvalidator.OneOf("daemon", "agent")},
								},
								"file_asset_reference": schema.SingleNestedAttribute{
									MarkdownDescription: "Reference to the configuration file asset.",
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
											MarkdownDescription: "Media type of the data.",
											Optional:            true,
										},
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

// ToRawConfiguration converts the typed component to raw configuration matching OpenAPI ServicesBackgroundTasksConfiguration schema
func (c *ServiceBackgroundTasksComponent) ToRawConfiguration() (map[string]any, error) {
	config := make(map[string]any)

	if len(c.BackgroundTasks) > 0 {
		backgroundTasks := make([]any, 0, len(c.BackgroundTasks))

		for _, task := range c.BackgroundTasks {
			taskMap := make(map[string]any)

			if helpers.IsConfiguredValue(task.TaskType) {
				taskMap["TaskType"] = task.TaskType.ValueString()
			}

			if helpers.IsConfiguredValue(task.TaskDescription) {
				taskMap["TaskDescription"] = task.TaskDescription.ValueString()
			}

			if task.ExecutableAssetReference != nil {
				execRef := make(map[string]any)

				reference := make(map[string]any)
				if helpers.IsConfiguredValue(task.ExecutableAssetReference.DataURL) {
					reference["DataURL"] = task.ExecutableAssetReference.DataURL.ValueString()
				}
				if helpers.IsConfiguredValue(task.ExecutableAssetReference.HashSHA256) {
					reference["Hash-SHA-256"] = task.ExecutableAssetReference.HashSHA256.ValueString()
				}
				reference["ContentType"] = "application/zip"

				execRef["Reference"] = reference

				taskMap["ExecutableAssetReference"] = execRef
			}

			if len(task.LaunchdConfigurations) > 0 {
				launchdConfigs := make([]any, 0, len(task.LaunchdConfigurations))

				for _, launchd := range task.LaunchdConfigurations {
					launchdMap := make(map[string]any)

					if helpers.IsConfiguredValue(launchd.Context) {
						launchdMap["Context"] = launchd.Context.ValueString()
					}

					if launchd.FileAssetReference != nil {
						fileRef := make(map[string]any)

						reference := make(map[string]any)
						if helpers.IsConfiguredValue(launchd.FileAssetReference.DataURL) {
							reference["DataURL"] = launchd.FileAssetReference.DataURL.ValueString()
						}
						if helpers.IsConfiguredValue(launchd.FileAssetReference.HashSHA256) {
							reference["Hash-SHA-256"] = launchd.FileAssetReference.HashSHA256.ValueString()
						}
						if helpers.IsConfiguredValue(launchd.FileAssetReference.ContentType) {
							reference["ContentType"] = launchd.FileAssetReference.ContentType.ValueString()
						}

						fileRef["Reference"] = reference

						launchdMap["FileAssetReference"] = fileRef
					}

					launchdConfigs = append(launchdConfigs, launchdMap)
				}

				taskMap["LaunchdConfigurations"] = launchdConfigs
			}

			backgroundTasks = append(backgroundTasks, taskMap)
		}

		config["backgroundTasks"] = backgroundTasks
	}

	return config, nil
}

// FromRawConfiguration populates the typed component from raw configuration data
func (c *ServiceBackgroundTasksComponent) FromRawConfiguration(raw map[string]any) error {
	if backgroundTasksRaw, exists := raw["backgroundTasks"]; exists {
		if backgroundTasksSlice, ok := backgroundTasksRaw.([]any); ok {
			backgroundTasks := make([]ServiceBackgroundTaskModel, 0, len(backgroundTasksSlice))

			for _, taskRaw := range backgroundTasksSlice {
				if taskMap, ok := taskRaw.(map[string]any); ok {
					task := ServiceBackgroundTaskModel{}

					if taskType, exists := taskMap["TaskType"]; exists {
						if taskTypeStr, ok := taskType.(string); ok {
							task.TaskType = types.StringValue(taskTypeStr)
						}
					}

					if taskDescription, exists := taskMap["TaskDescription"]; exists {
						if taskDescriptionStr, ok := taskDescription.(string); ok {
							task.TaskDescription = types.StringValue(taskDescriptionStr)
						}
					}

					if execRefRaw, exists := taskMap["ExecutableAssetReference"]; exists {
						if execRefMap, ok := execRefRaw.(map[string]any); ok {
							if refRaw, exists := execRefMap["Reference"]; exists {
								if refMap, ok := refRaw.(map[string]any); ok {
									execRef := &DataAssetRefModel{}

									if dataURL, exists := refMap["DataURL"]; exists {
										if dataURLStr, ok := dataURL.(string); ok {
											execRef.DataURL = types.StringValue(dataURLStr)
										}
									}

									if hashSHA256, exists := refMap["Hash-SHA-256"]; exists {
										if hashStr, ok := hashSHA256.(string); ok {
											execRef.HashSHA256 = types.StringValue(hashStr)
										}
									}

									execRef.ContentType = types.StringValue("application/zip")

									task.ExecutableAssetReference = execRef
								}
							}
						}
					}

					if launchdConfigsRaw, exists := taskMap["LaunchdConfigurations"]; exists {
						if launchdConfigsSlice, ok := launchdConfigsRaw.([]any); ok {
							launchdConfigs := make([]LaunchdItemModel, 0, len(launchdConfigsSlice))

							for _, launchdRaw := range launchdConfigsSlice {
								if launchdMap, ok := launchdRaw.(map[string]any); ok {
									launchd := LaunchdItemModel{}

									if context, exists := launchdMap["Context"]; exists {
										if contextStr, ok := context.(string); ok {
											launchd.Context = types.StringValue(contextStr)
										}
									}

									if fileRefRaw, exists := launchdMap["FileAssetReference"]; exists {
										if fileRefMap, ok := fileRefRaw.(map[string]any); ok {
											if refRaw, exists := fileRefMap["Reference"]; exists {
												if refMap, ok := refRaw.(map[string]any); ok {
													fileRef := &DataAssetRefModel{}

													if dataURL, exists := refMap["DataURL"]; exists {
														if dataURLStr, ok := dataURL.(string); ok {
															fileRef.DataURL = types.StringValue(dataURLStr)
														}
													}

													if hashSHA256, exists := refMap["Hash-SHA-256"]; exists {
														if hashStr, ok := hashSHA256.(string); ok {
															fileRef.HashSHA256 = types.StringValue(hashStr)
														}
													}

													if contentType, exists := refMap["ContentType"]; exists {
														if contentTypeStr, ok := contentType.(string); ok {
															fileRef.ContentType = types.StringValue(contentTypeStr)
														}
													}

													launchd.FileAssetReference = fileRef
												}
											}
										}
									}

									launchdConfigs = append(launchdConfigs, launchd)
								}
							}

							task.LaunchdConfigurations = launchdConfigs
						}
					}

					backgroundTasks = append(backgroundTasks, task)
				}
			}

			c.BackgroundTasks = backgroundTasks
		}
	}

	return nil
}

// GetIdentifier returns the component identifier for service background tasks
func (c *ServiceBackgroundTasksComponent) GetIdentifier() string {
	return "com.jamf.ddm.service-background-tasks"
}

// ToClientComponent converts the typed component to the format expected by the Blueprint API client
func (c *ServiceBackgroundTasksComponent) ToClientComponent() (*BlueprintComponentData, error) {
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
