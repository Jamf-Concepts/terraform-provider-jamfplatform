// Copyright 2025 Jamf Software LLC.

package blueprint

import (
	"context"
	"encoding/json"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/blueprints/blueprint/components"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

			normalizedConfig := normalizeJSON(string(jsonBytes))
			component.Configuration = json.RawMessage(normalizedConfig)
		}
		allComponents = append(allComponents, component)
	}

	r.collectStronglyTypedComponents(&allComponents, &diags, data)

	return allComponents, diags
}

// collectStronglyTypedComponents processes all strongly-typed components using a scalable approach.
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

// collectLegacyPayloadsString is a special helper for legacy payloads from string attribute.
func (r *BlueprintResource) collectLegacyPayloadsString(allComponents *[]client.BlueprintComponentV1, diags *diag.Diagnostics, payloadContent string, blueprintName string) {
	var payloadArray []any
	if err := json.Unmarshal([]byte(payloadContent), &payloadArray); err != nil {
		diags.AddError(
			"Error parsing legacy payloads JSON",
			"Could not parse payload_content as JSON array: "+err.Error(),
		)
		return
	}

	config := map[string]any{
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
