// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package blueprint

import (
	"context"
	"encoding/json"
	"maps"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/blueprints"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/blueprints/blueprint/components"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// flatStepName is the single step name used when a blueprint is authored with the deprecated flat
// top-level component attributes (no component_blocks).
const flatStepName = "Declaration group"

// buildSteps converts the model into the ordered SDK steps for a create/update request. In block
// mode (component_blocks set) it emits one step per block, preserving order, per-block name, and
// per-block activation condition. In the deprecated flat mode it emits the single "Declaration
// group" step carrying every top-level component and the top-level activation condition.
func (r *BlueprintResource) buildSteps(ctx context.Context, data *BlueprintResourceModel) ([]blueprints.BlueprintStep, diag.Diagnostics) {
	var diags diag.Diagnostics
	blueprintName := data.Name.ValueString()

	if len(data.ComponentBlocks) > 0 {
		steps := make([]blueprints.BlueprintStep, 0, len(data.ComponentBlocks))
		for _, block := range data.ComponentBlocks {
			components, blockDiags := r.collectBlockComponents(ctx, block)
			r.collectBlockLegacyPayloads(&components, &blockDiags, block.LegacyPayloads, blueprintName)
			diags.Append(blockDiags...)
			if blockDiags.HasError() {
				continue
			}
			steps = append(steps, blueprints.BlueprintStep{
				Name:                block.Name.ValueStringPointer(),
				ActivationPredicate: block.ActivationConditions.ValueStringPointer(),
				Components:          components,
			})
		}
		return steps, diags
	}

	components, flatDiags := r.collectBlockComponents(ctx, data.flatComponentsAsBlock())
	if !data.LegacyPayloads.IsNull() && !data.LegacyPayloads.IsUnknown() {
		r.collectLegacyPayloads(&components, &flatDiags, data.LegacyPayloads, blueprintName)
	}
	diags.Append(flatDiags...)

	stepName := flatStepName
	steps := []blueprints.BlueprintStep{
		{
			Name:                &stepName,
			Components:          components,
			ActivationPredicate: data.ActivationConditions.ValueStringPointer(),
		},
	}
	return steps, diags
}

// collectBlockComponents gathers the raw and strongly-typed components of one block into SDK
// component values. Legacy payloads are collected separately by the caller because the flat
// (dynamic) and block (JSON-string) shapes differ. The flat top-level authoring style reuses this
// by passing data.flatComponentsAsBlock(); each entry in component_blocks passes its own carrier.
func (r *BlueprintResource) collectBlockComponents(ctx context.Context, block ComponentBlockModel) ([]blueprints.Component, diag.Diagnostics) {
	var allComponents []blueprints.Component
	var diags diag.Diagnostics

	for _, comp := range block.Components {
		component := blueprints.Component{
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

	r.collectStronglyTypedComponents(&allComponents, &diags, block)

	return allComponents, diags
}

// collectStronglyTypedComponents processes all strongly-typed components of a block.
func (r *BlueprintResource) collectStronglyTypedComponents(allComponents *[]blueprints.Component, diags *diag.Diagnostics, block ComponentBlockModel) {
	if block.AudioAccessorySettings != nil {
		r.collectSingleComponent(allComponents, diags, block.AudioAccessorySettings, "audio accessory settings")
	}

	if block.CustomDeclarations != nil {
		r.collectSingleComponent(allComponents, diags, block.CustomDeclarations, "custom declarations")
	}

	if block.DiskManagementSettings != nil {
		r.collectSingleComponent(allComponents, diags, block.DiskManagementSettings, "disk management settings")
	}

	if block.MathSettings != nil {
		r.collectSingleComponent(allComponents, diags, block.MathSettings, "math settings")
	}

	if block.PasscodePolicy != nil {
		r.collectSingleComponent(allComponents, diags, block.PasscodePolicy, "passcode policy")
	}

	if block.SafariBookmarks != nil {
		r.collectSingleComponent(allComponents, diags, block.SafariBookmarks, "safari bookmarks")
	}

	if block.SafariExtensions != nil {
		r.collectSingleComponent(allComponents, diags, block.SafariExtensions, "safari extensions")
	}

	if block.SafariSettings != nil {
		r.collectSingleComponent(allComponents, diags, block.SafariSettings, "safari settings")
	}

	if block.ServiceBackgroundTasks != nil {
		r.collectSingleComponent(allComponents, diags, block.ServiceBackgroundTasks, "service background tasks")
	}

	if block.ServiceConfigurationFiles != nil {
		r.collectSingleComponent(allComponents, diags, block.ServiceConfigurationFiles, "service configuration files")
	}

	if block.SoftwareUpdate != nil {
		r.collectSingleComponent(allComponents, diags, block.SoftwareUpdate, "software update")
	}

	if block.SoftwareUpdateSettings != nil {
		r.collectSingleComponent(allComponents, diags, block.SoftwareUpdateSettings, "software update settings")
	}
}

// collectSingleComponent is a helper function that can collect any type of strongly-typed component.
func (r *BlueprintResource) collectSingleComponent(allComponents *[]blueprints.Component, diags *diag.Diagnostics, comp components.ComponentConverter, componentName string) {
	clientComp, err := comp.ToClientComponent()
	if err != nil {
		diags.AddError("Failed to build "+componentName+" component", err.Error())
		return
	}
	*allComponents = append(*allComponents, *clientComp)
}

// legacyPayloadEntry is one legacy payload flattened to its payload type and settings map, the
// common shape both the flat (dynamic) and block (JSON-string) legacy collectors reduce to.
type legacyPayloadEntry struct {
	PayloadType string
	Settings    map[string]any
}

// collectLegacyPayloads builds the legacy configuration profile component from the deprecated
// dynamic top-level legacy_payloads value.
func (r *BlueprintResource) collectLegacyPayloads(allComponents *[]blueprints.Component, diags *diag.Diagnostics, legacyPayloads types.Dynamic, blueprintName string) {
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

	entries := make([]legacyPayloadEntry, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			diags.AddError("Invalid legacy_payloads entry", "Each legacy payload must be an object.")
			return
		}

		payloadType, _ := obj["payload_type"].(string)
		entry := legacyPayloadEntry{PayloadType: payloadType}
		if settings, exists := obj["settings"]; exists {
			if settingsMap, ok := settings.(map[string]any); ok {
				entry.Settings = settingsMap
			}
		}
		entries = append(entries, entry)
	}

	r.appendLegacyConfigProfile(allComponents, diags, entries, blueprintName)
}

// collectBlockLegacyPayloads builds the legacy configuration profile component from a block's
// legacy_payloads list, whose settings arrive as JSON object strings.
func (r *BlueprintResource) collectBlockLegacyPayloads(allComponents *[]blueprints.Component, diags *diag.Diagnostics, payloads []BlockLegacyPayloadModel, blueprintName string) {
	if len(payloads) == 0 {
		return
	}

	entries := make([]legacyPayloadEntry, 0, len(payloads))
	for _, payload := range payloads {
		entry := legacyPayloadEntry{PayloadType: payload.PayloadType.ValueString()}
		if helpers.IsConfiguredValue(payload.Settings) && payload.Settings.ValueString() != "" {
			var settingsMap map[string]any
			if err := json.Unmarshal([]byte(payload.Settings.ValueString()), &settingsMap); err != nil {
				diags.AddError(
					"Invalid legacy payload settings",
					"settings for payload_type "+entry.PayloadType+" must be a JSON object string (use jsonencode): "+err.Error(),
				)
				return
			}
			entry.Settings = settingsMap
		}
		entries = append(entries, entry)
	}

	r.appendLegacyConfigProfile(allComponents, diags, entries, blueprintName)
}

// appendLegacyConfigProfile assembles the shared com.jamf.ddm-configuration-profile component from
// the flattened legacy payload entries and appends it. It rejects a missing payload type or a
// duplicate payload type.
func (r *BlueprintResource) appendLegacyConfigProfile(allComponents *[]blueprints.Component, diags *diag.Diagnostics, entries []legacyPayloadEntry, blueprintName string) {
	seenPayloadTypes := make(map[string]bool, len(entries))
	payloadArray := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		if entry.PayloadType == "" {
			diags.AddError("Missing payload_type", "Each legacy payload must include a payload_type key.")
			return
		}

		if seenPayloadTypes[entry.PayloadType] {
			diags.AddError(
				"Duplicate payload_type",
				"Legacy payloads must not contain duplicate payload types. Found duplicate: "+entry.PayloadType,
			)
			return
		}
		seenPayloadTypes[entry.PayloadType] = true

		payload := map[string]any{
			"payloadType":       entry.PayloadType,
			"payloadIdentifier": generatePayloadIdentifier(entry.PayloadType),
		}
		maps.Copy(payload, entry.Settings)
		payloadArray = append(payloadArray, payload)
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

	*allComponents = append(*allComponents, blueprints.Component{
		Identifier:    "com.jamf.ddm-configuration-profile",
		Configuration: json.RawMessage(configJSON),
	})
}
