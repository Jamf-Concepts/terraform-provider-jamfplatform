// Copyright 2025 Jamf Software LLC.

package blueprint

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/blueprints/blueprint/components"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// BlueprintResource implements the Terraform resource for Jamf Blueprint.
type BlueprintResource struct {
	client *client.Client
}

// BlueprintResourceModel represents the Terraform resource model for a Jamf Blueprint.
type BlueprintResourceModel struct {
	ID                        types.String                                    `tfsdk:"id"`
	Name                      types.String                                    `tfsdk:"name"`
	Description               types.String                                    `tfsdk:"description"`
	DeviceGroups              types.Set                                       `tfsdk:"device_groups"`
	Components                []ComponentModel                                `tfsdk:"raw_component"`
	AudioAccessorySettings    []components.AudioAccessorySettingsComponent    `tfsdk:"audio_accessory_settings"`
	DiskManagementSettings    []components.DiskManagementPolicyComponent      `tfsdk:"disk_management_settings"`
	MathSettings              []components.MathSettingsComponent              `tfsdk:"math_settings"`
	PasscodePolicy            []components.PasscodePolicyComponent            `tfsdk:"passcode_policy"`
	SafariBookmarks           []components.SafariBookmarksComponent           `tfsdk:"safari_bookmarks"`
	SafariExtensions          []components.SafariExtensionsComponent          `tfsdk:"safari_extensions"`
	SafariSettings            []components.SafariSettingsComponent            `tfsdk:"safari_settings"`
	ServiceBackgroundTasks    []components.ServiceBackgroundTasksComponent    `tfsdk:"service_background_tasks"`
	ServiceConfigurationFiles []components.ServiceConfigurationFilesComponent `tfsdk:"service_configuration_files"`
	SoftwareUpdate            []components.SoftwareUpdateComponent            `tfsdk:"software_update"`
	SoftwareUpdateSettings    []components.SoftwareUpdateSettingsComponent    `tfsdk:"software_update_settings"`
	LegacyPayloads            types.String                                    `tfsdk:"legacy_payloads"`
	Created                   types.String                                    `tfsdk:"created"`
	Updated                   types.String                                    `tfsdk:"updated"`
	DeploymentState           types.String                                    `tfsdk:"deployment_state"`
}

// BlueprintDataSource implements the Terraform data source for Jamf Blueprint.
type BlueprintDataSource struct {
	client *client.Client
}

// BlueprintDataSourceModel defines the data structure for the blueprint data source.
type BlueprintDataSourceModel struct {
	ID              types.String     `tfsdk:"id"`
	Name            types.String     `tfsdk:"name"`
	BlueprintID     types.String     `tfsdk:"blueprint_id"`
	Description     types.String     `tfsdk:"description"`
	Created         types.String     `tfsdk:"created"`
	Updated         types.String     `tfsdk:"updated"`
	DeploymentState types.String     `tfsdk:"deployment_state"`
	DeviceGroups    types.Set        `tfsdk:"device_groups"`
	Components      []ComponentModel `tfsdk:"component"`
}

// ComponentModel defines the data structure for a blueprint component.
type ComponentModel struct {
	Identifier    types.String `tfsdk:"identifier"`
	Configuration types.Map    `tfsdk:"configuration"`
}
