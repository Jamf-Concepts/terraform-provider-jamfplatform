// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package blueprint

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/blueprints"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/blueprints/blueprint/components"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// updateModelFromAPIResponse updates the Terraform model with data from the API response. It
// selects between the deprecated flat top-level attributes and ordered component_blocks based on
// which the prior model used (a fresh/empty model — import, list resource — defaults to blocks).
func updateModelFromAPIResponse(ctx context.Context, model *BlueprintResourceModel, blueprint *blueprints.BlueprintDetail) diag.Diagnostics {
	var diags diag.Diagnostics

	blockMode := len(model.ComponentBlocks) > 0 || !model.hasFlatComponents()

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

	if blockMode {
		updateComponentBlocksFromAPI(ctx, model, blueprint)
	} else {
		diags.Append(updateFlatComponentsFromAPI(ctx, model, blueprint)...)
	}

	model.Timeouts = helpers.EnsureResourceTimeouts(model.Timeouts, blueprintTimeoutAttributeTypes)
	return diags
}

// updateComponentBlocksFromAPI populates model.ComponentBlocks from every wire step (block mode)
// and clears the deprecated flat attributes so state carries a single representation.
func updateComponentBlocksFromAPI(ctx context.Context, model *BlueprintResourceModel, blueprint *blueprints.BlueprintDetail) {
	prior := model.ComponentBlocks
	blocks := make([]ComponentBlockModel, 0, len(blueprint.Steps))

	for i, step := range blueprint.Steps {
		var priorRaw map[string]struct{}
		var priorLegacy []BlockLegacyPayloadModel
		priorName := types.StringNull()
		priorActivation := types.StringNull()
		if i < len(prior) {
			priorRaw = rawIdentifierSet(prior[i].Components)
			priorLegacy = prior[i].LegacyPayloads
			priorName = prior[i].Name
			priorActivation = prior[i].ActivationConditions
		}

		block, apiComponentsByID := mapStepComponents(ctx, step, priorRaw)
		block.Name = helpers.ReconcileOptionalStringPointer(step.Name, priorName)
		block.ActivationConditions = helpers.ReconcileOptionalStringPointer(step.ActivationPredicate, priorActivation)
		block.LegacyPayloads = flattenBlockLegacyPayloads(priorLegacy, apiComponentsByID, priorRaw)
		blocks = append(blocks, block)
	}

	if len(blocks) > 0 {
		model.ComponentBlocks = blocks
	} else {
		model.ComponentBlocks = nil
	}

	model.applyFlatComponentsFromBlock(ComponentBlockModel{})
	model.LegacyPayloads = types.DynamicNull()
	model.ActivationConditions = types.StringNull()
}

// updateFlatComponentsFromAPI populates the deprecated flat top-level attributes from the first
// step (flat mode). When the blueprint has more than one step it also emits a migration warning:
// the flat attributes cannot represent the extra blocks, so applying would collapse them.
func updateFlatComponentsFromAPI(ctx context.Context, model *BlueprintResourceModel, blueprint *blueprints.BlueprintDetail) diag.Diagnostics {
	var diags diag.Diagnostics

	priorRaw := rawIdentifierSet(model.Components)

	var step blueprints.BlueprintStep
	if len(blueprint.Steps) > 0 {
		step = blueprint.Steps[0]
	}

	block, apiComponentsByID := mapStepComponents(ctx, step, priorRaw)
	model.applyFlatComponentsFromBlock(block)
	model.LegacyPayloads = flattenFlatLegacyPayloads(model.LegacyPayloads, apiComponentsByID, priorRaw)
	model.ActivationConditions = helpers.ReconcileOptionalStringPointer(step.ActivationPredicate, model.ActivationConditions)

	if len(blueprint.Steps) > 1 {
		diags.AddWarning(
			"Blueprint has multiple component blocks",
			"This blueprint has multiple component blocks, but the configuration uses the deprecated top-level component attributes, which can only represent the first block. "+
				"Only the first block is reflected in state, and applying this configuration would remove the others. Migrate to `component_blocks` to manage every block.",
		)
	}

	return diags
}

// rawIdentifierSet collects the raw_component identifiers a prior model authored, so components
// that also have a strongly-typed representation stay in raw_component when the user chose it.
func rawIdentifierSet(components []ComponentModel) map[string]struct{} {
	identifiers := make(map[string]struct{}, len(components))
	for _, comp := range components {
		if identifier := comp.Identifier.ValueString(); identifier != "" {
			identifiers[identifier] = struct{}{}
		}
	}
	return identifiers
}

// mapStepComponents converts one wire step's raw and strongly-typed components into a
// ComponentBlockModel carrier, and returns the step's components keyed by identifier so the caller
// can flatten legacy payloads. It leaves Name, ActivationConditions, and LegacyPayloads unset — the
// caller reconciles those.
func mapStepComponents(ctx context.Context, step blueprints.BlueprintStep, priorRawIdentifiers map[string]struct{}) (ComponentBlockModel, map[string]blueprints.Component) {
	var block ComponentBlockModel

	apiComponentsByID := make(map[string]blueprints.Component)
	var rawComponents []ComponentModel

	for _, comp := range step.Components {
		apiComponentsByID[comp.Identifier] = comp

		_, handledAsRaw := priorRawIdentifiers[comp.Identifier]
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
		configMapValue, _ := types.MapValueFrom(ctx, types.StringType, configMap)

		rawComponents = append(rawComponents, ComponentModel{
			Identifier:    types.StringValue(comp.Identifier),
			Configuration: configMapValue,
		})
	}

	if len(rawComponents) > 0 {
		block.Components = rawComponents
	}

	updateStronglyTypedComponentsFromAPI(&block, apiComponentsByID, priorRawIdentifiers)
	return block, apiComponentsByID
}

// updateStronglyTypedComponentsFromAPI updates all strongly-typed components of a block from the
// API response.
func updateStronglyTypedComponentsFromAPI(block *ComponentBlockModel, apiComponentsByID map[string]blueprints.Component, rawIdentifiers map[string]struct{}) {
	block.AudioAccessorySettings = buildTypedComponent[components.AudioAccessorySettingsComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.audio-accessory-settings", func(raw json.RawMessage, target *components.AudioAccessorySettingsComponent) error {
		return target.FromRawConfiguration(raw)
	})

	block.CustomDeclarations = buildTypedComponent[components.CustomDeclarationsComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.custom-declarations", func(raw json.RawMessage, target *components.CustomDeclarationsComponent) error {
		return target.FromRawConfiguration(raw)
	})

	block.DiskManagementSettings = buildTypedComponent[components.DiskManagementPolicyComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.disk-management", func(raw json.RawMessage, target *components.DiskManagementPolicyComponent) error {
		return target.FromRawConfiguration(raw)
	})

	block.MathSettings = buildTypedComponent[components.MathSettingsComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.math-settings", func(raw json.RawMessage, target *components.MathSettingsComponent) error {
		return target.FromRawConfiguration(raw)
	})

	block.PasscodePolicy = buildTypedComponent[components.PasscodePolicyComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.passcode-settings", func(raw json.RawMessage, target *components.PasscodePolicyComponent) error {
		return target.FromRawConfiguration(raw)
	})

	block.SafariBookmarks = buildTypedComponent[components.SafariBookmarksComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.safari-bookmarks", func(raw json.RawMessage, target *components.SafariBookmarksComponent) error {
		return target.FromRawConfiguration(raw)
	})

	block.SafariExtensions = buildTypedComponent[components.SafariExtensionsComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.safari-extensions", func(raw json.RawMessage, target *components.SafariExtensionsComponent) error {
		return target.FromRawConfiguration(raw)
	})

	block.SafariSettings = buildTypedComponent[components.SafariSettingsComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.safari-settings", func(raw json.RawMessage, target *components.SafariSettingsComponent) error {
		return target.FromRawConfiguration(raw)
	})

	block.ServiceBackgroundTasks = buildTypedComponent[components.ServiceBackgroundTasksComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.service-background-tasks", func(raw json.RawMessage, target *components.ServiceBackgroundTasksComponent) error {
		return target.FromRawConfiguration(raw)
	})

	block.ServiceConfigurationFiles = buildTypedComponent[components.ServiceConfigurationFilesComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.service-configuration-files", func(raw json.RawMessage, target *components.ServiceConfigurationFilesComponent) error {
		return target.FromRawConfiguration(raw)
	})

	block.SoftwareUpdate = buildTypedComponent[components.SoftwareUpdateComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.sw-updates", func(raw json.RawMessage, target *components.SoftwareUpdateComponent) error {
		return target.FromRawConfiguration(raw)
	})

	block.SoftwareUpdateSettings = buildTypedComponent[components.SoftwareUpdateSettingsComponent](apiComponentsByID, rawIdentifiers, "com.jamf.ddm.software-update-settings", func(raw json.RawMessage, target *components.SoftwareUpdateSettingsComponent) error {
		return target.FromRawConfiguration(raw)
	})
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

// legacyPayloadItems extracts the legacy configuration profile's payloads from the wire as a list
// of `{payload_type, settings}` maps (settings present only when non-empty), or nil when the
// component is absent or malformed. The caller decides how to render them (dynamic or JSON string).
func legacyPayloadItems(apiComponentsByID map[string]blueprints.Component) []any {
	rawJSON, ok := parseComponentConfiguration(apiComponentsByID, "com.jamf.ddm-configuration-profile")
	if !ok {
		return nil
	}

	var config map[string]any
	if err := json.Unmarshal(rawJSON, &config); err != nil {
		return nil
	}

	payloadArray, ok := config["payloadContent"].([]any)
	if !ok {
		return nil
	}

	var items []any
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

		entry := map[string]any{"payload_type": payloadType}
		if len(settingsMap) > 0 {
			entry["settings"] = settingsMap
		}
		items = append(items, entry)
	}

	if len(items) == 0 {
		return nil
	}
	return items
}

// flattenFlatLegacyPayloads renders the wire legacy payloads into the deprecated top-level dynamic
// value. When the user manages the configuration profile as a raw_component it is left untouched;
// when the server value is semantically identical to the prior value that prior value is preserved
// (the dynamic null-typing reconcile — see dynamicPayloadsMatchJSON).
func flattenFlatLegacyPayloads(prior types.Dynamic, apiComponentsByID map[string]blueprints.Component, rawIdentifiers map[string]struct{}) types.Dynamic {
	if _, handledAsRaw := rawIdentifiers["com.jamf.ddm-configuration-profile"]; handledAsRaw {
		return prior
	}

	items := legacyPayloadItems(apiComponentsByID)
	if len(items) == 0 {
		return types.DynamicNull()
	}

	if dynamicPayloadsMatchJSON(prior, items) {
		return prior
	}

	dynVal, err := helpers.JSONToTerraformDynamic(items)
	if err != nil {
		return types.DynamicNull()
	}
	return dynVal
}

// flattenBlockLegacyPayloads renders the wire legacy payloads into a block's typed list, with each
// payload's settings as a canonical JSON string. When the user manages the configuration profile as
// a raw_component the prior value is left untouched. For each payload, the prior settings string is
// preserved when it is semantically identical to the server value, keeping diffs stable.
func flattenBlockLegacyPayloads(prior []BlockLegacyPayloadModel, apiComponentsByID map[string]blueprints.Component, rawIdentifiers map[string]struct{}) []BlockLegacyPayloadModel {
	if _, handledAsRaw := rawIdentifiers["com.jamf.ddm-configuration-profile"]; handledAsRaw {
		return prior
	}

	items := legacyPayloadItems(apiComponentsByID)
	if len(items) == 0 {
		return nil
	}

	priorByType := make(map[string]types.String, len(prior))
	for _, entry := range prior {
		priorByType[entry.PayloadType.ValueString()] = entry.Settings
	}

	result := make([]BlockLegacyPayloadModel, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		payloadType, _ := obj["payload_type"].(string)
		entry := BlockLegacyPayloadModel{PayloadType: types.StringValue(payloadType)}

		settings, hasSettings := obj["settings"].(map[string]any)
		switch {
		case !hasSettings:
			entry.Settings = types.StringNull()
		case settingsStringMatchesJSON(priorByType[payloadType], settings):
			entry.Settings = priorByType[payloadType]
		default:
			if encoded, err := json.Marshal(settings); err == nil {
				entry.Settings = types.StringValue(string(encoded))
			} else {
				entry.Settings = types.StringNull()
			}
		}
		result = append(result, entry)
	}
	return result
}

// settingsStringMatchesJSON reports whether a prior settings JSON string is semantically identical
// to a server-derived settings map, comparing canonical JSON encodings (sorted keys, float64
// numbers). It keeps a block's settings string stable when the server echoes an equivalent value.
func settingsStringMatchesJSON(prior types.String, settings map[string]any) bool {
	if prior.IsNull() || prior.IsUnknown() {
		return false
	}

	var priorObj any
	if err := json.Unmarshal([]byte(prior.ValueString()), &priorObj); err != nil {
		return false
	}

	priorBytes, err := json.Marshal(priorObj)
	if err != nil {
		return false
	}

	settingsBytes, err := json.Marshal(settings)
	if err != nil {
		return false
	}

	return bytes.Equal(priorBytes, settingsBytes)
}

// dynamicPayloadsMatchJSON reports whether the prior dynamic value is
// semantically identical to the server-derived payload items, comparing their
// canonical JSON encodings. Numbers normalise to float64 on both sides and
// json.Marshal sorts object keys, so the comparison is order-independent for
// object keys and insensitive to the dynamic null-typing that otherwise causes
// a perpetual diff.
func dynamicPayloadsMatchJSON(prior types.Dynamic, apiItems []any) bool {
	if prior.IsNull() || prior.IsUnknown() {
		return false
	}

	priorJSON, err := helpers.TerraformDynamicToJSON(prior)
	if err != nil {
		return false
	}

	priorBytes, err := json.Marshal(priorJSON)
	if err != nil {
		return false
	}

	apiBytes, err := json.Marshal(apiItems)
	if err != nil {
		return false
	}

	return bytes.Equal(priorBytes, apiBytes)
}
