// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"encoding/json"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/blueprints"
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/bpcomponents/declarations"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ServiceBackgroundTasksComponent represents a strongly-typed service background tasks component.
type ServiceBackgroundTasksComponent struct {
	BackgroundTasks []ServiceBackgroundTaskModel `tfsdk:"background_tasks"`
}

// ServiceBackgroundTaskModel represents a background task configuration.
type ServiceBackgroundTaskModel struct {
	TaskType                 types.String       `tfsdk:"task_type"`
	TaskDescription          types.String       `tfsdk:"task_description"`
	ExecutableAssetReference *DataAssetRefModel `tfsdk:"executable_asset_reference"`
	LaunchdConfigurations    []LaunchdItemModel `tfsdk:"launchd_configurations"`
}

// DataAssetRefModel represents a data asset reference.
type DataAssetRefModel struct {
	DataURL     types.String `tfsdk:"data_url"`
	HashSHA256  types.String `tfsdk:"hash_sha_256"`
	ContentType types.String `tfsdk:"content_type"`
}

// LaunchdItemModel represents a launchd configuration item.
type LaunchdItemModel struct {
	Context            types.String       `tfsdk:"context"`
	FileAssetReference *DataAssetRefModel `tfsdk:"file_asset_reference"`
}

// ServiceBackgroundTasksComponentSchema returns the Terraform schema for service background tasks component.
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
									Required:            true,
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

// buildDataAssetReference converts a DataAssetRefModel to a declarations.DataAssetReference.
func buildDataAssetReference(ref *DataAssetRefModel, defaultContentType string) declarations.DataAssetReference {
	r := declarations.AssetDataReference{
		DataURL:     ref.DataURL.ValueString(),
		ContentType: &defaultContentType,
	}
	if helpers.IsConfiguredValue(ref.HashSHA256) {
		h := ref.HashSHA256.ValueString()
		r.HashSHA256 = &h
	}
	if helpers.IsConfiguredValue(ref.ContentType) {
		ct := ref.ContentType.ValueString()
		r.ContentType = &ct
	}
	return declarations.DataAssetReference{Reference: r}
}

// ToRawConfiguration converts the typed component to raw JSON configuration.
func (c *ServiceBackgroundTasksComponent) ToRawConfiguration() (json.RawMessage, error) {
	tasks := make([]declarations.ServiceBackgroundTasksConfiguration, 0, len(c.BackgroundTasks))

	for _, task := range c.BackgroundTasks {
		t := declarations.ServiceBackgroundTasksConfiguration{}

		if helpers.IsConfiguredValue(task.TaskType) {
			t.TaskType = task.TaskType.ValueString()
		}
		if helpers.IsConfiguredValue(task.TaskDescription) {
			desc := task.TaskDescription.ValueString()
			t.TaskDescription = &desc
		}

		if task.ExecutableAssetReference != nil {
			ref := buildDataAssetReference(task.ExecutableAssetReference, "application/zip")
			t.ExecutableAssetReference = &ref
		}

		if len(task.LaunchdConfigurations) > 0 {
			launchdItems := make([]declarations.LaunchdItem, 0, len(task.LaunchdConfigurations))
			for _, lc := range task.LaunchdConfigurations {
				item := declarations.LaunchdItem{}
				if helpers.IsConfiguredValue(lc.Context) {
					item.Context = lc.Context.ValueString()
				}
				if lc.FileAssetReference != nil {
					item.FileAssetReference = buildDataAssetReference(lc.FileAssetReference, "")
				}
				launchdItems = append(launchdItems, item)
			}
			t.LaunchdConfigurations = &launchdItems
		}

		tasks = append(tasks, t)
	}

	cfg := declarations.ServicesBackgroundTasksConfiguration{BackgroundTasks: tasks}
	return json.Marshal(cfg)
}

// dataAssetRefFromSDK converts a declarations.DataAssetReference to a DataAssetRefModel.
func dataAssetRefFromSDK(ref declarations.DataAssetReference, defaultContentType string) *DataAssetRefModel {
	m := &DataAssetRefModel{
		DataURL: types.StringValue(ref.Reference.DataURL),
	}
	if ref.Reference.HashSHA256 != nil {
		m.HashSHA256 = types.StringValue(*ref.Reference.HashSHA256)
	}
	if ref.Reference.ContentType != nil {
		m.ContentType = types.StringValue(*ref.Reference.ContentType)
	} else {
		m.ContentType = types.StringValue(defaultContentType)
	}
	return m
}

// FromRawConfiguration populates the typed component from raw JSON configuration.
func (c *ServiceBackgroundTasksComponent) FromRawConfiguration(raw json.RawMessage) error {
	var cfg declarations.ServicesBackgroundTasksConfiguration
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return err
	}

	tasks := make([]ServiceBackgroundTaskModel, 0, len(cfg.BackgroundTasks))
	for _, t := range cfg.BackgroundTasks {
		task := ServiceBackgroundTaskModel{
			TaskType: types.StringValue(t.TaskType),
		}

		if t.TaskDescription != nil {
			task.TaskDescription = types.StringValue(*t.TaskDescription)
		}

		if t.ExecutableAssetReference != nil {
			task.ExecutableAssetReference = dataAssetRefFromSDK(*t.ExecutableAssetReference, "application/zip")
		}

		if t.LaunchdConfigurations != nil {
			launchdConfigs := make([]LaunchdItemModel, 0, len(*t.LaunchdConfigurations))
			for _, lc := range *t.LaunchdConfigurations {
				item := LaunchdItemModel{
					Context:            types.StringValue(lc.Context),
					FileAssetReference: dataAssetRefFromSDK(lc.FileAssetReference, ""),
				}
				launchdConfigs = append(launchdConfigs, item)
			}
			task.LaunchdConfigurations = launchdConfigs
		}

		tasks = append(tasks, task)
	}

	c.BackgroundTasks = tasks
	return nil
}

// GetIdentifier returns the component identifier for service background tasks.
func (c *ServiceBackgroundTasksComponent) GetIdentifier() string {
	return "com.jamf.ddm.service-background-tasks"
}

// ToClientComponent converts the typed component to the format expected by the Blueprint API client.
func (c *ServiceBackgroundTasksComponent) ToClientComponent() (*blueprints.Component, error) {
	cfg, err := c.ToRawConfiguration()
	if err != nil {
		return nil, err
	}
	return &blueprints.Component{
		Identifier:    c.GetIdentifier(),
		Configuration: cfg,
	}, nil
}
