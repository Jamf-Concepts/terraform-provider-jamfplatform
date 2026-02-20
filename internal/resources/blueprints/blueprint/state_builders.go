// Copyright 2025 Jamf Software LLC.

package blueprint

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/blueprints/blueprint/components"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// updateModelFromAPIResponse updates the Terraform model with data from the API response.
func updateModelFromAPIResponse(model *BlueprintResourceModel, blueprint *client.BlueprintDetailV1) {
	stateRawIdentifiers := make(map[string]struct{}, len(model.Components))
	for _, comp := range model.Components {
		identifier := comp.Identifier.ValueString()
		if identifier != "" {
			stateRawIdentifiers[identifier] = struct{}{}
		}
	}

	model.ID = types.StringValue(blueprint.ID)
	model.Name = types.StringValue(blueprint.Name)

	if !helpers.IsConfiguredValue(model.Description) && blueprint.Description == "" {
		model.Description = types.StringNull()
	} else {
		model.Description = types.StringValue(blueprint.Description)
	}

	model.Created = types.StringValue(blueprint.Created)
	model.Updated = types.StringValue(blueprint.Updated)
	model.DeploymentState = types.StringValue(blueprint.DeploymentState.State)
	model.Deployed = types.BoolValue(strings.EqualFold(blueprint.DeploymentState.State, "DEPLOYED"))

	deviceGroupsSet, _ := types.SetValueFrom(context.Background(), types.StringType, blueprint.Scope.DeviceGroups)
	model.DeviceGroups = deviceGroupsSet

	apiComponentsByID := make(map[string]client.BlueprintComponentV1)
	var rawComponents []ComponentModel

	if len(blueprint.Steps) > 0 {
		step := blueprint.Steps[0]
		rawComponents = make([]ComponentModel, 0, len(step.Components))

		for _, comp := range step.Components {
			apiComponentsByID[comp.Identifier] = comp

			_, handledAsRaw := stateRawIdentifiers[comp.Identifier]
			if _, isTyped := stronglyTypedComponentIdentifiers[comp.Identifier]; isTyped && !handledAsRaw {
				continue
			}

			configMap := make(map[string]string)
			if comp.Configuration != nil {
				var jsonObj map[string]any
				if err := json.Unmarshal(comp.Configuration, &jsonObj); err == nil {
					flattenJSON(jsonObj, "", configMap)
				}
			}
			configMapValue, _ := types.MapValueFrom(context.Background(), types.StringType, configMap)

			rawComponents = append(rawComponents, ComponentModel{
				Identifier:    types.StringValue(comp.Identifier),
				Configuration: configMapValue,
			})
		}
	}

	model.Components = rawComponents
	updateStronglyTypedComponentsFromAPI(model, apiComponentsByID, stateRawIdentifiers)
	model.Timeouts = helpers.EnsureResourceTimeouts(model.Timeouts, blueprintTimeoutAttributeTypes)
}

// updateStronglyTypedComponentsFromAPI updates all strongly-typed components from API response.
func updateStronglyTypedComponentsFromAPI(model *BlueprintResourceModel, apiComponentsByID map[string]client.BlueprintComponentV1, rawIdentifiers map[string]struct{}) {
	model.AudioAccessorySettings = buildTypedComponentSlice[components.AudioAccessorySettingsComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.audio-accessory-settings", func(cfg map[string]any, target *components.AudioAccessorySettingsComponent) error {
		return target.FromRawConfiguration(cfg)
	})

	model.CustomDeclarations = buildTypedComponentSlice[components.CustomDeclarationsComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.custom-declarations", func(cfg map[string]any, target *components.CustomDeclarationsComponent) error {
		return target.FromRawConfiguration(cfg)
	})

	model.DiskManagementSettings = buildTypedComponentSlice[components.DiskManagementPolicyComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.disk-management", func(cfg map[string]any, target *components.DiskManagementPolicyComponent) error {
		return target.FromRawConfiguration(cfg)
	})

	model.MathSettings = buildTypedComponentSlice[components.MathSettingsComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.math-settings", func(cfg map[string]any, target *components.MathSettingsComponent) error {
		return target.FromRawConfiguration(cfg)
	})

	model.PasscodePolicy = buildTypedComponentSlice[components.PasscodePolicyComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.passcode-settings", func(cfg map[string]any, target *components.PasscodePolicyComponent) error {
		return target.FromRawConfiguration(cfg)
	})

	model.SafariBookmarks = buildTypedComponentSlice[components.SafariBookmarksComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.safari-bookmarks", func(cfg map[string]any, target *components.SafariBookmarksComponent) error {
		return target.FromRawConfiguration(cfg)
	})

	model.SafariExtensions = buildTypedComponentSlice[components.SafariExtensionsComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.safari-extensions", func(cfg map[string]any, target *components.SafariExtensionsComponent) error {
		return target.FromRawConfiguration(cfg)
	})

	model.SafariSettings = buildTypedComponentSlice[components.SafariSettingsComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.safari-settings", func(cfg map[string]any, target *components.SafariSettingsComponent) error {
		return target.FromRawConfiguration(cfg)
	})

	model.ServiceBackgroundTasks = buildTypedComponentSlice[components.ServiceBackgroundTasksComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.service-background-tasks", func(cfg map[string]any, target *components.ServiceBackgroundTasksComponent) error {
		return target.FromRawConfiguration(cfg)
	})

	model.ServiceConfigurationFiles = buildTypedComponentSlice[components.ServiceConfigurationFilesComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.service-configuration-files", func(cfg map[string]any, target *components.ServiceConfigurationFilesComponent) error {
		return target.FromRawConfiguration(cfg)
	})

	model.SoftwareUpdate = buildTypedComponentSlice[components.SoftwareUpdateComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.sw-updates", func(cfg map[string]any, target *components.SoftwareUpdateComponent) error {
		return target.FromRawConfiguration(cfg)
	})

	model.SoftwareUpdateSettings = buildTypedComponentSlice[components.SoftwareUpdateSettingsComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.software-update-settings", func(cfg map[string]any, target *components.SoftwareUpdateSettingsComponent) error {
		return target.FromRawConfiguration(cfg)
	})

	updateLegacyPayloadsFromAPI(model, apiComponentsByID, rawIdentifiers)
}

// buildTypedComponentSlice is a generic helper to build strongly-typed component slices.
func buildTypedComponentSlice[T any](apiComponentsByID map[string]client.BlueprintComponentV1, rawIdentifiers map[string]struct{}, identifier string, populate func(map[string]any, *T) error) []T {
	if _, handledAsRaw := rawIdentifiers[identifier]; handledAsRaw {
		return nil
	}

	config, ok := parseComponentConfiguration(apiComponentsByID, identifier)
	if !ok {
		return nil
	}

	var component T
	if err := populate(config, &component); err != nil {
		return nil
	}

	return []T{component}
}

// parseComponentConfiguration extracts and parses the configuration of a component by its identifier.
func parseComponentConfiguration(apiComponentsByID map[string]client.BlueprintComponentV1, identifier string) (map[string]any, bool) {
	apiComp, exists := apiComponentsByID[identifier]
	if !exists || apiComp.Configuration == nil {
		return nil, false
	}

	var jsonObj map[string]any
	if err := json.Unmarshal(apiComp.Configuration, &jsonObj); err != nil {
		return nil, false
	}

	return jsonObj, true
}

// updateLegacyPayloadsFromAPI handles the special case of legacy payloads component.
func updateLegacyPayloadsFromAPI(model *BlueprintResourceModel, apiComponentsByID map[string]client.BlueprintComponentV1, rawIdentifiers map[string]struct{}) {
	if _, handledAsRaw := rawIdentifiers["com.jamf.ddm-configuration-profile"]; handledAsRaw {
		return
	}

	config, ok := parseComponentConfiguration(apiComponentsByID, "com.jamf.ddm-configuration-profile")
	if !ok {
		model.LegacyPayloads = types.StringNull()
		return
	}

	payloadContent, exists := config["payloadContent"]
	if !exists {
		model.LegacyPayloads = types.StringNull()
		return
	}

	payloadJSON, err := json.Marshal(payloadContent)
	if err != nil {
		model.LegacyPayloads = types.StringNull()
		return
	}

	model.LegacyPayloads = types.StringValue(string(payloadJSON))
}
