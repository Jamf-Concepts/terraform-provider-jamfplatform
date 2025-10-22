// Copyright 2025 Jamf Software LLC.

package blueprint

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/blueprints/blueprint/components"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// updateModelFromAPIResponse updates the Terraform model with data from the API response.
func updateModelFromAPIResponse(model *BlueprintResourceModel, blueprint *client.BlueprintDetailV1) {
	model.ID = types.StringValue(blueprint.ID)
	model.Name = types.StringValue(blueprint.Name)

	if model.Description.IsNull() && blueprint.Description == "" {
		model.Description = types.StringNull()
	} else {
		model.Description = types.StringValue(blueprint.Description)
	}

	model.Created = types.StringValue(blueprint.Created)
	model.Updated = types.StringValue(blueprint.Updated)
	model.DeploymentState = types.StringValue(blueprint.DeploymentState.State)

	deviceGroupsSet, _ := types.SetValueFrom(context.Background(), types.StringType, blueprint.Scope.DeviceGroups)
	model.DeviceGroups = deviceGroupsSet

	if len(blueprint.Steps) > 0 {
		step := blueprint.Steps[0]

		apiComponentsByID := make(map[string]client.BlueprintComponentV1)
		for _, comp := range step.Components {
			apiComponentsByID[comp.Identifier] = comp
		}

		components := make([]ComponentModel, len(model.Components))
		for i, modelComp := range model.Components {
			identifier := modelComp.Identifier.ValueString()

			if apiComp, exists := apiComponentsByID[identifier]; exists {
				configMap := make(map[string]string)
				if apiComp.Configuration != nil {
					var jsonObj map[string]interface{}
					if err := json.Unmarshal(apiComp.Configuration, &jsonObj); err == nil {
						flattenJSON(jsonObj, "", configMap)
					}
				}

				configMapValue, _ := types.MapValueFrom(context.Background(), types.StringType, configMap)
				components[i] = ComponentModel{
					Identifier:    types.StringValue(apiComp.Identifier),
					Configuration: configMapValue,
				}
			} else {
				components[i] = modelComp
			}
		}
		model.Components = components
		updateStronglyTypedComponentsFromAPI(model, apiComponentsByID)

	} else {
		model.Components = []ComponentModel{}
		model.AudioAccessorySettings = []components.AudioAccessorySettingsComponent{}
		model.DiskManagementSettings = []components.DiskManagementPolicyComponent{}
	}
}

// normalizeJSON takes a JSON string and returns it with sorted keys to ensure consistent comparison
func normalizeJSON(jsonStr string) string {
	if jsonStr == "" {
		return ""
	}

	var obj interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return jsonStr
	}

	normalized, err := json.Marshal(obj)
	if err != nil {
		return jsonStr
	}

	return string(normalized)
}

// setNestedValue sets a value in a nested map structure using underscore notation
func setNestedValue(obj map[string]interface{}, key string, value string) {
	parts := strings.Split(key, "_")
	current := obj

	for i := 0; i < len(parts)-1; i++ {
		if current[parts[i]] == nil {
			current[parts[i]] = make(map[string]interface{})
		}
		if nested, ok := current[parts[i]].(map[string]interface{}); ok {
			current = nested
		} else {
			current[parts[i]] = make(map[string]interface{})
			current = current[parts[i]].(map[string]interface{})
		}
	}

	finalKey := parts[len(parts)-1]
	if value == "" {
		current[finalKey] = nil
	} else if value == "true" {
		current[finalKey] = true
	} else if value == "false" {
		current[finalKey] = false
	} else if num, err := strconv.Atoi(value); err == nil {
		current[finalKey] = num
	} else {
		if strings.HasPrefix(value, "[") || strings.HasPrefix(value, "{") {
			var jsonValue interface{}
			if err := json.Unmarshal([]byte(value), &jsonValue); err == nil {
				current[finalKey] = jsonValue
				return
			}
		}
		current[finalKey] = value
	}
}

// flattenJSON flattens a nested JSON object into a flat map with underscore notation keys
func flattenJSON(obj map[string]interface{}, prefix string, result map[string]string) {
	for key, value := range obj {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "_" + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			flattenJSON(v, fullKey, result)
		case nil:
			result[fullKey] = ""
		case bool:
			if v {
				result[fullKey] = "true"
			} else {
				result[fullKey] = "false"
			}
		case float64:
			result[fullKey] = strconv.FormatFloat(v, 'f', -1, 64)
		case int:
			result[fullKey] = strconv.Itoa(v)
		case string:
			result[fullKey] = v
		default:
			if jsonBytes, err := json.Marshal(v); err == nil {
				result[fullKey] = string(jsonBytes)
			} else {
				result[fullKey] = ""
			}
		}
	}
}

// isNotFoundError checks if the error is a 404 not found error
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errorStr := err.Error()
	return strings.Contains(errorStr, "status 404") ||
		strings.Contains(errorStr, "was not found") ||
		strings.Contains(errorStr, "NOT_FOUND")
}

// isServerError checks if the error is a server error (500)
func isServerError(err error) bool {
	if err == nil {
		return false
	}
	errorStr := err.Error()
	return strings.Contains(errorStr, "status 500") ||
		strings.Contains(errorStr, "Internal Server Error")
}

// collectAllComponents gathers components from both raw and strongly-typed sources
func (r *BlueprintResource) collectAllComponents(ctx context.Context, data *BlueprintResourceModel) ([]client.BlueprintComponentV1, diag.Diagnostics) {
	var allComponents []client.BlueprintComponentV1
	var diags diag.Diagnostics

	for _, comp := range data.Components {
		component := client.BlueprintComponentV1{
			Identifier: comp.Identifier.ValueString(),
		}

		if !comp.Configuration.IsNull() && !comp.Configuration.IsUnknown() {
			configMap := make(map[string]string)
			configDiags := comp.Configuration.ElementsAs(ctx, &configMap, false)
			if configDiags.HasError() {
				diags.Append(configDiags...)
				continue
			}

			jsonObj := make(map[string]interface{})
			for key, value := range configMap {
				setNestedValue(jsonObj, key, value)
			}

			jsonBytes, err := json.Marshal(jsonObj)
			if err != nil {
				diags.AddError(
					"Error encoding component configuration",
					"Could not encode component configuration to JSON: "+err.Error(),
				)
				continue
			}

			normalizedConfig := normalizeJSON(string(jsonBytes))
			component.Configuration = json.RawMessage(normalizedConfig)
		}
		allComponents = append(allComponents, component)
	}

	r.collectStronglyTypedComponents(&allComponents, &diags, data)

	return allComponents, diags
}

// collectStronglyTypedComponents processes all strongly-typed components using a scalable approach
func (r *BlueprintResource) collectStronglyTypedComponents(allComponents *[]client.BlueprintComponentV1, diags *diag.Diagnostics, data *BlueprintResourceModel) {
	for i := range data.AudioAccessorySettings {
		r.collectSingleComponent(allComponents, diags, &data.AudioAccessorySettings[i], "audio accessory settings")
	}

	for i := range data.DiskManagementSettings {
		r.collectSingleComponent(allComponents, diags, &data.DiskManagementSettings[i], "disk management settings")
	}

	for i := range data.MathSettings {
		r.collectSingleComponent(allComponents, diags, &data.MathSettings[i], "math settings")
	}

	for i := range data.PasscodePolicy {
		r.collectSingleComponent(allComponents, diags, &data.PasscodePolicy[i], "passcode policy")
	}

	for i := range data.SafariBookmarks {
		r.collectSingleComponent(allComponents, diags, &data.SafariBookmarks[i], "safari bookmarks")
	}

	for i := range data.SafariExtensions {
		r.collectSingleComponent(allComponents, diags, &data.SafariExtensions[i], "safari extensions")
	}

	for i := range data.SafariSettings {
		r.collectSingleComponent(allComponents, diags, &data.SafariSettings[i], "safari settings")
	}

	for i := range data.ServiceBackgroundTasks {
		r.collectSingleComponent(allComponents, diags, &data.ServiceBackgroundTasks[i], "service background tasks")
	}

	for i := range data.ServiceConfigurationFiles {
		r.collectSingleComponent(allComponents, diags, &data.ServiceConfigurationFiles[i], "service configuration files")
	}

	for i := range data.SoftwareUpdate {
		r.collectSingleComponent(allComponents, diags, &data.SoftwareUpdate[i], "software update")
	}

	for i := range data.SoftwareUpdateSettings {
		r.collectSingleComponent(allComponents, diags, &data.SoftwareUpdateSettings[i], "software update settings")
	}

	if !data.LegacyPayloads.IsNull() && !data.LegacyPayloads.IsUnknown() {
		r.collectLegacyPayloadsString(allComponents, diags, data.LegacyPayloads.ValueString(), data.Name.ValueString())
	}
}

// collectSingleComponent is a helper function that can collect any type of strongly-typed component
func (r *BlueprintResource) collectSingleComponent(allComponents *[]client.BlueprintComponentV1, diags *diag.Diagnostics, comp components.ComponentConverter, componentName string) {
	clientComp, err := comp.ToClientComponent()
	if err != nil {
		diags.AddError(
			"Error converting "+componentName+" component",
			"Could not convert "+componentName+" to client format: "+err.Error(),
		)
		return
	}
	*allComponents = append(*allComponents, client.BlueprintComponentV1{
		Identifier:    clientComp.Identifier,
		Configuration: clientComp.Configuration,
	})
}

// collectLegacyPayloadsString is a special helper for legacy payloads from string attribute
func (r *BlueprintResource) collectLegacyPayloadsString(allComponents *[]client.BlueprintComponentV1, diags *diag.Diagnostics, payloadContent string, blueprintName string) {
	var payloadArray []interface{}
	if err := json.Unmarshal([]byte(payloadContent), &payloadArray); err != nil {
		diags.AddError(
			"Error parsing legacy payloads JSON",
			"Could not parse payload_content as JSON array: "+err.Error(),
		)
		return
	}

	config := map[string]interface{}{
		"payloadDisplayName": blueprintName,
		"payloadContent":     payloadArray,
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		diags.AddError(
			"Error encoding legacy payloads configuration",
			"Could not encode configuration to JSON: "+err.Error(),
		)
		return
	}

	*allComponents = append(*allComponents, client.BlueprintComponentV1{
		Identifier:    "com.jamf.ddm-configuration-profile",
		Configuration: json.RawMessage(configJSON),
	})
}

// updateStronglyTypedComponentsFromAPI updates all strongly-typed components from API response
func updateStronglyTypedComponentsFromAPI(model *BlueprintResourceModel, apiComponentsByID map[string]client.BlueprintComponentV1) {
	updateComponentsFromAPI("com.jamf.ddm.audio-accessory-settings", apiComponentsByID, func(jsonObj map[string]interface{}) {
		for i := range model.AudioAccessorySettings {
			_ = model.AudioAccessorySettings[i].FromRawConfiguration(jsonObj)
		}
	})

	updateComponentsFromAPI("com.jamf.ddm.disk-management", apiComponentsByID, func(jsonObj map[string]interface{}) {
		for i := range model.DiskManagementSettings {
			_ = model.DiskManagementSettings[i].FromRawConfiguration(jsonObj)
		}
	})

	updateComponentsFromAPI("com.jamf.ddm.math-settings", apiComponentsByID, func(jsonObj map[string]interface{}) {
		for i := range model.MathSettings {
			_ = model.MathSettings[i].FromRawConfiguration(jsonObj)
		}
	})

	updateComponentsFromAPI("com.jamf.ddm.passcode-settings", apiComponentsByID, func(jsonObj map[string]interface{}) {
		for i := range model.PasscodePolicy {
			_ = model.PasscodePolicy[i].FromRawConfiguration(jsonObj)
		}
	})

	updateComponentsFromAPI("com.jamf.ddm.safari-bookmarks", apiComponentsByID, func(jsonObj map[string]interface{}) {
		for i := range model.SafariBookmarks {
			_ = model.SafariBookmarks[i].FromRawConfiguration(jsonObj)
		}
	})

	updateComponentsFromAPI("com.jamf.ddm.safari-extensions", apiComponentsByID, func(jsonObj map[string]interface{}) {
		for i := range model.SafariExtensions {
			_ = model.SafariExtensions[i].FromRawConfiguration(jsonObj)
		}
	})

	updateComponentsFromAPI("com.jamf.ddm.safari-settings", apiComponentsByID, func(jsonObj map[string]interface{}) {
		for i := range model.SafariSettings {
			_ = model.SafariSettings[i].FromRawConfiguration(jsonObj)
		}
	})

	updateComponentsFromAPI("com.jamf.ddm.service-background-tasks", apiComponentsByID, func(jsonObj map[string]interface{}) {
		for i := range model.ServiceBackgroundTasks {
			_ = model.ServiceBackgroundTasks[i].FromRawConfiguration(jsonObj)
		}
	})

	updateComponentsFromAPI("com.jamf.ddm.service-configuration-files", apiComponentsByID, func(jsonObj map[string]interface{}) {
		for i := range model.ServiceConfigurationFiles {
			_ = model.ServiceConfigurationFiles[i].FromRawConfiguration(jsonObj)
		}
	})

	updateComponentsFromAPI("com.jamf.ddm.sw-updates", apiComponentsByID, func(jsonObj map[string]interface{}) {
		for i := range model.SoftwareUpdate {
			_ = model.SoftwareUpdate[i].FromRawConfiguration(jsonObj)
		}
	})

	updateComponentsFromAPI("com.jamf.ddm.software-update-settings", apiComponentsByID, func(jsonObj map[string]interface{}) {
		for i := range model.SoftwareUpdateSettings {
			_ = model.SoftwareUpdateSettings[i].FromRawConfiguration(jsonObj)
		}
	})

	updateComponentsFromAPI("com.jamf.ddm-configuration-profile", apiComponentsByID, func(jsonObj map[string]interface{}) {
		if !model.LegacyPayloads.IsNull() {
			if payloadContent, exists := jsonObj["payloadContent"]; exists {
				payloadJSON, err := json.Marshal(payloadContent)
				if err == nil {
					model.LegacyPayloads = types.StringValue(string(payloadJSON))
				}
			}
		}
	})
}

// updateComponentsFromAPI is a generic helper that updates components of any type from API response
func updateComponentsFromAPI(identifier string, apiComponentsByID map[string]client.BlueprintComponentV1, updateFunc func(map[string]interface{})) {
	if apiComp, exists := apiComponentsByID[identifier]; exists && apiComp.Configuration != nil {
		var jsonObj map[string]interface{}
		if err := json.Unmarshal(apiComp.Configuration, &jsonObj); err == nil {
			updateFunc(jsonObj)
		}
	}
}
