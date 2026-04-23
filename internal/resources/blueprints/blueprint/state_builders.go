// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package blueprint

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/blueprints"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/blueprints/blueprint/components"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// updateModelFromAPIResponse updates the Terraform model with data from the API response.
func updateModelFromAPIResponse(ctx context.Context, model *BlueprintResourceModel, blueprint *blueprints.BlueprintDetail) {
	stateRawIdentifiers := make(map[string]struct{}, len(model.Components))
	for _, comp := range model.Components {
		identifier := comp.Identifier.ValueString()
		if identifier != "" {
			stateRawIdentifiers[identifier] = struct{}{}
		}
	}

	model.ID = types.StringValue(blueprint.ID)
	model.Name = types.StringValue(blueprint.Name)

	model.Description = helpers.ReconcileOptionalStringPointer(blueprint.Description, model.Description)

	model.Created = types.StringValue(blueprint.Created.Format(time.RFC3339))
	model.Updated = types.StringValue(blueprint.Updated.Format(time.RFC3339))
	if blueprint.DeploymentState != nil {
		model.DeploymentState = types.StringValue(blueprint.DeploymentState.State)
		model.Deployed = types.BoolValue(strings.EqualFold(blueprint.DeploymentState.State, "DEPLOYED"))
	} else {
		model.DeploymentState = types.StringValue("")
		model.Deployed = types.BoolValue(false)
	}

	deviceGroupsSet, _ := types.SetValueFrom(context.Background(), types.StringType, scopeDeviceGroups(blueprint.Scope))
	model.DeviceGroups = deviceGroupsSet

	apiComponentsByID := make(map[string]blueprints.Component)
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

	if len(rawComponents) > 0 {
		model.Components = rawComponents
	} else {
		model.Components = nil
	}
	updateStronglyTypedComponentsFromAPI(ctx, model, apiComponentsByID, stateRawIdentifiers)
	model.Timeouts = helpers.EnsureResourceTimeouts(model.Timeouts, blueprintTimeoutAttributeTypes)
}

// updateStronglyTypedComponentsFromAPI updates all strongly-typed components from API response.
func updateStronglyTypedComponentsFromAPI(ctx context.Context, model *BlueprintResourceModel, apiComponentsByID map[string]blueprints.Component, rawIdentifiers map[string]struct{}) {
	model.AudioAccessorySettings = buildTypedComponent[components.AudioAccessorySettingsComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.audio-accessory-settings", func(raw json.RawMessage, target *components.AudioAccessorySettingsComponent) error {
		return target.FromRawConfiguration(raw)
	})

	model.CustomDeclarations = buildTypedComponent[components.CustomDeclarationsComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.custom-declarations", func(raw json.RawMessage, target *components.CustomDeclarationsComponent) error {
		return target.FromRawConfiguration(raw)
	})

	model.DiskManagementSettings = buildTypedComponent[components.DiskManagementPolicyComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.disk-management", func(raw json.RawMessage, target *components.DiskManagementPolicyComponent) error {
		return target.FromRawConfiguration(raw)
	})

	model.MathSettings = buildTypedComponent[components.MathSettingsComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.math-settings", func(raw json.RawMessage, target *components.MathSettingsComponent) error {
		return target.FromRawConfiguration(raw)
	})

	model.PasscodePolicy = buildTypedComponent[components.PasscodePolicyComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.passcode-settings", func(raw json.RawMessage, target *components.PasscodePolicyComponent) error {
		return target.FromRawConfiguration(raw)
	})

	model.SafariBookmarks = buildTypedComponent[components.SafariBookmarksComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.safari-bookmarks", func(raw json.RawMessage, target *components.SafariBookmarksComponent) error {
		return target.FromRawConfiguration(raw)
	})

	model.SafariExtensions = buildTypedComponent[components.SafariExtensionsComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.safari-extensions", func(raw json.RawMessage, target *components.SafariExtensionsComponent) error {
		return target.FromRawConfiguration(raw)
	})

	model.SafariSettings = buildTypedComponent[components.SafariSettingsComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.safari-settings", func(raw json.RawMessage, target *components.SafariSettingsComponent) error {
		return target.FromRawConfiguration(raw)
	})

	model.ServiceBackgroundTasks = buildTypedComponent[components.ServiceBackgroundTasksComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.service-background-tasks", func(raw json.RawMessage, target *components.ServiceBackgroundTasksComponent) error {
		return target.FromRawConfiguration(raw)
	})

	model.ServiceConfigurationFiles = buildTypedComponent[components.ServiceConfigurationFilesComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.service-configuration-files", func(raw json.RawMessage, target *components.ServiceConfigurationFilesComponent) error {
		return target.FromRawConfiguration(raw)
	})

	model.SoftwareUpdate = buildTypedComponent[components.SoftwareUpdateComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.sw-updates", func(raw json.RawMessage, target *components.SoftwareUpdateComponent) error {
		return target.FromRawConfiguration(raw)
	})

	model.SoftwareUpdateSettings = buildTypedComponent[components.SoftwareUpdateSettingsComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.software-update-settings", func(raw json.RawMessage, target *components.SoftwareUpdateSettingsComponent) error {
		return target.FromRawConfiguration(raw)
	})

	updateLegacyPayloadsFromAPI(ctx, model, apiComponentsByID, rawIdentifiers)
}

// buildTypedComponent is a generic helper to build a strongly-typed singleton component pointer.
func buildTypedComponent[T any](apiComponentsByID map[string]blueprints.Component, rawIdentifiers map[string]struct{}, identifier string, populate func(json.RawMessage, *T) error) *T {
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

	return &component
}

// parseComponentConfiguration returns the raw JSON configuration of a component by its identifier.
func parseComponentConfiguration(apiComponentsByID map[string]blueprints.Component, identifier string) (json.RawMessage, bool) {
	apiComp, exists := apiComponentsByID[identifier]
	if !exists || apiComp.Configuration == nil {
		return nil, false
	}
	return apiComp.Configuration, true
}

// updateLegacyPayloadsFromAPI handles the special case of legacy payloads component.
func updateLegacyPayloadsFromAPI(ctx context.Context, model *BlueprintResourceModel, apiComponentsByID map[string]blueprints.Component, rawIdentifiers map[string]struct{}) {
	if _, handledAsRaw := rawIdentifiers["com.jamf.ddm-configuration-profile"]; handledAsRaw {
		return
	}

	rawJSON, ok := parseComponentConfiguration(apiComponentsByID, "com.jamf.ddm-configuration-profile")
	if !ok {
		model.LegacyPayloads = types.DynamicNull()
		return
	}

	var config map[string]any
	if err := json.Unmarshal(rawJSON, &config); err != nil {
		model.LegacyPayloads = types.DynamicNull()
		return
	}

	payloadContent, exists := config["payloadContent"]
	if !exists {
		model.LegacyPayloads = types.DynamicNull()
		return
	}

	payloadArray, ok := payloadContent.([]any)
	if !ok {
		model.LegacyPayloads = types.DynamicNull()
		return
	}

	var resultItems []any
	for _, item := range payloadArray {
		payloadMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		payloadType, _ := payloadMap["payloadType"].(string)

		settingsMap := make(map[string]any, len(payloadMap))
		for k, v := range payloadMap {
			if k == "payloadType" || k == "payloadIdentifier" {
				continue
			}
			settingsMap[k] = v
		}

		entry := map[string]any{
			"payload_type": payloadType,
		}
		if len(settingsMap) > 0 {
			entry["settings"] = settingsMap
		}

		resultItems = append(resultItems, entry)
	}

	if len(resultItems) > 0 {
		dynVal, err := helpers.JSONToTerraformDynamic(resultItems)
		if err != nil {
			tflog.Warn(ctx, "Failed to convert legacy payloads to dynamic", map[string]any{
				"error": err.Error(),
			})
			model.LegacyPayloads = types.DynamicNull()
			return
		}
		model.LegacyPayloads = dynVal
	} else {
		model.LegacyPayloads = types.DynamicNull()
	}
}
