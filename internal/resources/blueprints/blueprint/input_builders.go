// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package blueprint

import (
	"context"
	"encoding/json"
	"maps"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/blueprints/blueprint/components"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// collectAllComponents gathers components from both raw and strongly-typed sources.
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

			jsonObj := make(map[string]any)
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

			component.Configuration = json.RawMessage(jsonBytes)
		}
		allComponents = append(allComponents, component)
	}

	r.collectStronglyTypedComponents(&allComponents, &diags, data)

	return allComponents, diags
}

// collectStronglyTypedComponents processes all strongly-typed components using a scalable approach.
func (r *BlueprintResource) collectStronglyTypedComponents(allComponents *[]client.BlueprintComponentV1, diags *diag.Diagnostics, data *BlueprintResourceModel) {
	if data.AudioAccessorySettings != nil {
		r.collectSingleComponent(allComponents, diags, data.AudioAccessorySettings, "audio accessory settings")
	}

	if data.CustomDeclarations != nil {
		r.collectSingleComponent(allComponents, diags, data.CustomDeclarations, "custom declarations")
	}

	if data.DiskManagementSettings != nil {
		r.collectSingleComponent(allComponents, diags, data.DiskManagementSettings, "disk management settings")
	}

	if data.MathSettings != nil {
		r.collectSingleComponent(allComponents, diags, data.MathSettings, "math settings")
	}

	if data.PasscodePolicy != nil {
		r.collectSingleComponent(allComponents, diags, data.PasscodePolicy, "passcode policy")
	}

	if data.SafariBookmarks != nil {
		r.collectSingleComponent(allComponents, diags, data.SafariBookmarks, "safari bookmarks")
	}

	if data.SafariExtensions != nil {
		r.collectSingleComponent(allComponents, diags, data.SafariExtensions, "safari extensions")
	}

	if data.SafariSettings != nil {
		r.collectSingleComponent(allComponents, diags, data.SafariSettings, "safari settings")
	}

	if data.ServiceBackgroundTasks != nil {
		r.collectSingleComponent(allComponents, diags, data.ServiceBackgroundTasks, "service background tasks")
	}

	if data.ServiceConfigurationFiles != nil {
		r.collectSingleComponent(allComponents, diags, data.ServiceConfigurationFiles, "service configuration files")
	}

	if data.SoftwareUpdate != nil {
		r.collectSingleComponent(allComponents, diags, data.SoftwareUpdate, "software update")
	}

	if data.SoftwareUpdateSettings != nil {
		r.collectSingleComponent(allComponents, diags, data.SoftwareUpdateSettings, "software update settings")
	}

	if !data.LegacyPayloads.IsNull() && !data.LegacyPayloads.IsUnknown() {
		r.collectLegacyPayloads(allComponents, diags, data.LegacyPayloads, data.Name.ValueString())
	}
}

// collectSingleComponent is a helper function that can collect any type of strongly-typed component.
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

// collectLegacyPayloads builds the API component from a dynamic legacy payloads value.
func (r *BlueprintResource) collectLegacyPayloads(allComponents *[]client.BlueprintComponentV1, diags *diag.Diagnostics, legacyPayloads types.Dynamic, blueprintName string) {
	raw, err := helpers.TerraformDynamicToJSON(legacyPayloads)
	if err != nil {
		diags.AddError("Error reading legacy payloads", "Could not convert legacy payloads to JSON: "+err.Error())
		return
	}

	items, ok := raw.([]any)
	if !ok {
		diags.AddError("Invalid legacy_payloads", "Expected a list of objects, got a non-list value.")
		return
	}

	payloadArray := make([]map[string]any, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			diags.AddError("Invalid legacy_payloads entry", "Each legacy payload must be an object.")
			return
		}

		payloadType, _ := obj["payload_type"].(string)
		if payloadType == "" {
			diags.AddError("Missing payload_type", "Each legacy payload must include a payload_type key.")
			return
		}

		entry := map[string]any{
			"payloadType":       payloadType,
			"payloadIdentifier": generatePayloadIdentifier(payloadType),
		}

		if settings, exists := obj["settings"]; exists {
			if settingsMap, ok := settings.(map[string]any); ok {
				maps.Copy(entry, settingsMap)
			}
		}

		payloadArray = append(payloadArray, entry)
	}

	config := map[string]any{
		"payloadDisplayName": blueprintName,
		"payloadContent":     payloadArray,
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		diags.AddError("Error encoding legacy payloads configuration", "Could not encode configuration to JSON: "+err.Error())
		return
	}

	*allComponents = append(*allComponents, client.BlueprintComponentV1{
		Identifier:    "com.jamf.ddm-configuration-profile",
		Configuration: json.RawMessage(configJSON),
	})
}
