// Copyright 2025 Jamf Software LLC.

package blueprint

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/blueprints/blueprint/components"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Constants for blueprint deployment states and component identifiers
const (
	blueprintDeploymentStateDeployed    = "DEPLOYED"
	blueprintDeploymentStateNotDeployed = "NOT_DEPLOYED"
)

// stronglyTypedComponentIdentifiers lists all component identifiers that have strongly-typed representations
var stronglyTypedComponentIdentifiers = map[string]struct{}{
	"com.jamf.ddm.audio-accessory-settings":    {},
	"com.jamf.ddm.custom-declarations":         {},
	"com.jamf.ddm.disk-management":             {},
	"com.jamf.ddm.math-settings":               {},
	"com.jamf.ddm.passcode-settings":           {},
	"com.jamf.ddm.safari-bookmarks":            {},
	"com.jamf.ddm.safari-extensions":           {},
	"com.jamf.ddm.safari-settings":             {},
	"com.jamf.ddm.service-background-tasks":    {},
	"com.jamf.ddm.service-configuration-files": {},
	"com.jamf.ddm.sw-updates":                  {},
	"com.jamf.ddm.software-update-settings":    {},
	"com.jamf.ddm-configuration-profile":       {},
}

// blueprintTimeoutAttributeTypes defines the attribute types for blueprint timeouts
var blueprintTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

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
				var jsonObj map[string]interface{}
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

		if helpers.IsConfiguredValue(comp.Configuration) {
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

	for i := range data.CustomDeclarations {
		r.collectSingleComponent(allComponents, diags, &data.CustomDeclarations[i], "custom declarations")
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

	if helpers.IsConfiguredValue(data.LegacyPayloads) {
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
func updateStronglyTypedComponentsFromAPI(model *BlueprintResourceModel, apiComponentsByID map[string]client.BlueprintComponentV1, rawIdentifiers map[string]struct{}) {
	model.AudioAccessorySettings = buildTypedComponentSlice[components.AudioAccessorySettingsComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.audio-accessory-settings", func(cfg map[string]interface{}, target *components.AudioAccessorySettingsComponent) error {
		return target.FromRawConfiguration(cfg)
	})

	model.CustomDeclarations = buildTypedComponentSlice[components.CustomDeclarationsComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.custom-declarations", func(cfg map[string]interface{}, target *components.CustomDeclarationsComponent) error {
		return target.FromRawConfiguration(cfg)
	})

	model.DiskManagementSettings = buildTypedComponentSlice[components.DiskManagementPolicyComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.disk-management", func(cfg map[string]interface{}, target *components.DiskManagementPolicyComponent) error {
		return target.FromRawConfiguration(cfg)
	})

	model.MathSettings = buildTypedComponentSlice[components.MathSettingsComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.math-settings", func(cfg map[string]interface{}, target *components.MathSettingsComponent) error {
		return target.FromRawConfiguration(cfg)
	})

	model.PasscodePolicy = buildTypedComponentSlice[components.PasscodePolicyComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.passcode-settings", func(cfg map[string]interface{}, target *components.PasscodePolicyComponent) error {
		return target.FromRawConfiguration(cfg)
	})

	model.SafariBookmarks = buildTypedComponentSlice[components.SafariBookmarksComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.safari-bookmarks", func(cfg map[string]interface{}, target *components.SafariBookmarksComponent) error {
		return target.FromRawConfiguration(cfg)
	})

	model.SafariExtensions = buildTypedComponentSlice[components.SafariExtensionsComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.safari-extensions", func(cfg map[string]interface{}, target *components.SafariExtensionsComponent) error {
		return target.FromRawConfiguration(cfg)
	})

	model.SafariSettings = buildTypedComponentSlice[components.SafariSettingsComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.safari-settings", func(cfg map[string]interface{}, target *components.SafariSettingsComponent) error {
		return target.FromRawConfiguration(cfg)
	})

	model.ServiceBackgroundTasks = buildTypedComponentSlice[components.ServiceBackgroundTasksComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.service-background-tasks", func(cfg map[string]interface{}, target *components.ServiceBackgroundTasksComponent) error {
		return target.FromRawConfiguration(cfg)
	})

	model.ServiceConfigurationFiles = buildTypedComponentSlice[components.ServiceConfigurationFilesComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.service-configuration-files", func(cfg map[string]interface{}, target *components.ServiceConfigurationFilesComponent) error {
		return target.FromRawConfiguration(cfg)
	})

	model.SoftwareUpdate = buildTypedComponentSlice[components.SoftwareUpdateComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.sw-updates", func(cfg map[string]interface{}, target *components.SoftwareUpdateComponent) error {
		return target.FromRawConfiguration(cfg)
	})

	model.SoftwareUpdateSettings = buildTypedComponentSlice[components.SoftwareUpdateSettingsComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.software-update-settings", func(cfg map[string]interface{}, target *components.SoftwareUpdateSettingsComponent) error {
		return target.FromRawConfiguration(cfg)
	})

	updateLegacyPayloadsFromAPI(model, apiComponentsByID, rawIdentifiers)
}

// buildTypedComponentSlice is a generic helper to build strongly-typed component slices
func buildTypedComponentSlice[T any](apiComponentsByID map[string]client.BlueprintComponentV1, rawIdentifiers map[string]struct{}, identifier string, populate func(map[string]interface{}, *T) error) []T {
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

// parseComponentConfiguration extracts and parses the configuration of a component by its identifier
func parseComponentConfiguration(apiComponentsByID map[string]client.BlueprintComponentV1, identifier string) (map[string]interface{}, bool) {
	apiComp, exists := apiComponentsByID[identifier]
	if !exists || apiComp.Configuration == nil {
		return nil, false
	}

	var jsonObj map[string]interface{}
	if err := json.Unmarshal(apiComp.Configuration, &jsonObj); err != nil {
		return nil, false
	}

	return jsonObj, true
}

// updateLegacyPayloadsFromAPI handles the special case of legacy payloads component
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

// desiredDeployedValue returns the desired deployed state based on the provided types.Bool value
func desiredDeployedValue(v types.Bool) bool {
	if !helpers.IsConfiguredValue(v) {
		return true
	}
	return v.ValueBool()
}

// reconcileBlueprintDeployment ensures the blueprint's deployment state matches the desired state
func (r *BlueprintResource) reconcileBlueprintDeployment(ctx context.Context, blueprintID string, desiredDeployed bool) (*client.BlueprintDetailV1, error) {
	blueprint, err := r.client.GetBlueprintByIDV1(ctx, blueprintID)
	if err != nil {
		return nil, err
	}

	if desiredDeployed {
		if !strings.EqualFold(blueprint.DeploymentState.State, blueprintDeploymentStateDeployed) {
			if err := r.client.DeployBlueprintV1(ctx, blueprintID); err != nil {
				return blueprint, err
			}
			return r.client.GetBlueprintByIDV1(ctx, blueprintID)
		}
		return blueprint, nil
	}

	if strings.EqualFold(blueprint.DeploymentState.State, blueprintDeploymentStateNotDeployed) {
		return blueprint, nil
	}

	if err := r.client.UndeployBlueprintV1(ctx, blueprintID); err != nil {
		return blueprint, err
	}

	return r.client.GetBlueprintByIDV1(ctx, blueprintID)
}
