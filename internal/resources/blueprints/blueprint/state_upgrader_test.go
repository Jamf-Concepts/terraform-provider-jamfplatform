// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package blueprint

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/jamf/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/jamf/terraform-provider-jamfplatform/internal/resources/blueprints/blueprint/components"
)

func TestSliceToPointer_Empty(t *testing.T) {
	var s []string
	result := sliceToPointer(s)
	if result != nil {
		t.Error("expected nil for empty slice")
	}
}

func TestSliceToPointer_Nil(t *testing.T) {
	result := sliceToPointer[int](nil)
	if result != nil {
		t.Error("expected nil for nil slice")
	}
}

func TestSliceToPointer_SingleElement(t *testing.T) {
	s := []string{"hello"}
	result := sliceToPointer(s)
	if result == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *result != "hello" {
		t.Errorf("expected 'hello', got %q", *result)
	}
}

func TestSliceToPointer_MultipleElements(t *testing.T) {
	s := []int{1, 2, 3}
	result := sliceToPointer(s)
	if result == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *result != 1 {
		t.Errorf("expected 1, got %d", *result)
	}
}

func TestBlueprintSchemaV0_HasComponentBlocks(t *testing.T) {
	s := blueprintSchemaV0()
	if s == nil {
		t.Fatal("v0 schema is nil")
	}

	expectedBlocks := []string{
		"raw_component",
		"audio_accessory_settings",
		"custom_declarations",
		"disk_management_settings",
		"math_settings",
		"passcode_policy",
		"safari_bookmarks",
		"safari_extensions",
		"safari_settings",
		"service_background_tasks",
		"service_configuration_files",
		"software_update",
		"software_update_settings",
	}

	for _, name := range expectedBlocks {
		if _, ok := s.Blocks[name]; !ok {
			t.Errorf("v0 schema missing block %q", name)
		}
	}
}

func TestBlueprintSchemaV0_BlocksAreListNested(t *testing.T) {
	s := blueprintSchemaV0()

	for name, block := range s.Blocks {
		if _, ok := block.(schema.ListNestedBlock); !ok {
			t.Errorf("block %q should be ListNestedBlock, got %T", name, block)
		}
	}
}

func TestBlueprintSchemaV0_ScalarAttributes(t *testing.T) {
	s := blueprintSchemaV0()

	expectedAttrs := []string{"id", "name", "description", "deployed", "device_groups", "legacy_payloads", "created", "updated", "deployment_state", "timeouts"}
	for _, name := range expectedAttrs {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("v0 schema missing attribute %q", name)
		}
	}
}

func TestBlueprintSchemaV0_ComponentsNotInAttributes(t *testing.T) {
	s := blueprintSchemaV0()

	componentNames := []string{
		"audio_accessory_settings",
		"safari_settings",
		"software_update",
		"raw_component",
	}

	for _, name := range componentNames {
		if _, ok := s.Attributes[name]; ok {
			t.Errorf("v0 schema should NOT have %q as an attribute (it should be a block)", name)
		}
	}
}

func TestBlueprintSchemaV0_SafariBookmarksNestedBlocks(t *testing.T) {
	s := blueprintSchemaV0()

	block, ok := s.Blocks["safari_bookmarks"]
	if !ok {
		t.Fatal("missing safari_bookmarks block")
	}

	listBlock := block.(schema.ListNestedBlock)
	managedBookmarks, ok := listBlock.NestedObject.Blocks["managed_bookmarks"]
	if !ok {
		t.Fatal("missing managed_bookmarks inner block")
	}

	mbListBlock := managedBookmarks.(schema.ListNestedBlock)
	bookmarks, ok := mbListBlock.NestedObject.Blocks["bookmarks"]
	if !ok {
		t.Fatal("missing bookmarks inner block")
	}

	bListBlock := bookmarks.(schema.ListNestedBlock)
	folder, ok := bListBlock.NestedObject.Blocks["folder"]
	if !ok {
		t.Fatal("missing folder inner block")
	}

	folderListBlock := folder.(schema.ListNestedBlock)
	if _, ok := folderListBlock.NestedObject.Attributes["title"]; !ok {
		t.Error("folder block missing 'title' attribute")
	}
	if _, ok := folderListBlock.NestedObject.Attributes["url"]; !ok {
		t.Error("folder block missing 'url' attribute")
	}
}

func TestBlueprintSchemaV0_ServiceBackgroundTasksNestedBlocks(t *testing.T) {
	s := blueprintSchemaV0()

	block := s.Blocks["service_background_tasks"].(schema.ListNestedBlock)
	bgTasks := block.NestedObject.Blocks["background_tasks"].(schema.ListNestedBlock)

	if _, ok := bgTasks.NestedObject.Attributes["task_type"]; !ok {
		t.Error("background_tasks missing 'task_type' attribute")
	}

	execRef, ok := bgTasks.NestedObject.Blocks["executable_asset_reference"]
	if !ok {
		t.Fatal("missing executable_asset_reference block")
	}
	if _, ok := execRef.(schema.SingleNestedBlock); !ok {
		t.Errorf("executable_asset_reference should be SingleNestedBlock, got %T", execRef)
	}

	launchdConfigs, ok := bgTasks.NestedObject.Blocks["launchd_configurations"]
	if !ok {
		t.Fatal("missing launchd_configurations block")
	}
	launchdList := launchdConfigs.(schema.ListNestedBlock)
	fileRef, ok := launchdList.NestedObject.Blocks["file_asset_reference"]
	if !ok {
		t.Fatal("missing file_asset_reference block")
	}
	if _, ok := fileRef.(schema.SingleNestedBlock); !ok {
		t.Errorf("file_asset_reference should be SingleNestedBlock, got %T", fileRef)
	}
}

func TestBlueprintSchemaV1_LegacyPayloadsIsString(t *testing.T) {
	s := blueprintSchemaV1()
	if s == nil {
		t.Fatal("v1 schema is nil")
	}

	attr, ok := s.Attributes["legacy_payloads"]
	if !ok {
		t.Fatal("v1 schema missing legacy_payloads attribute")
	}
	if _, ok := attr.(schema.StringAttribute); !ok {
		t.Errorf("v1 legacy_payloads should be StringAttribute, got %T", attr)
	}
}

func TestBlueprintSchemaV1_ComponentsAreAttributes(t *testing.T) {
	s := blueprintSchemaV1()

	componentAttrs := []string{
		"audio_accessory_settings",
		"custom_declarations",
		"disk_management_settings",
		"math_settings",
		"passcode_policy",
		"safari_bookmarks",
		"safari_extensions",
		"safari_settings",
		"service_background_tasks",
		"service_configuration_files",
		"software_update",
		"software_update_settings",
		"raw_component",
	}

	for _, name := range componentAttrs {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("v1 schema missing attribute %q", name)
		}
	}
}

func TestUpgradeLegacyPayloadsFromString_ValidJSON(t *testing.T) {
	input := types.StringValue(`[{"payloadType":"com.apple.applicationaccess","payloadIdentifier":"test-uuid","allowSafariHistoryClearing":false}]`)

	result := upgradeLegacyPayloadsFromString(input)

	if result.IsNull() {
		t.Fatal("expected non-null result")
	}

	raw, err := helpers.TerraformDynamicToJSON(result)
	if err != nil {
		t.Fatalf("failed to convert: %v", err)
	}
	items, ok := raw.([]any)
	if !ok {
		t.Fatalf("expected list, got %T", raw)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 payload, got %d", len(items))
	}
	payload := items[0].(map[string]any)
	if payload["payload_type"] != "com.apple.applicationaccess" {
		t.Errorf("expected payload_type 'com.apple.applicationaccess', got %v", payload["payload_type"])
	}
	if _, ok := payload["settings"]; !ok {
		t.Error("expected settings key")
	}
}

func TestUpgradeLegacyPayloadsFromString_NullString(t *testing.T) {
	result := upgradeLegacyPayloadsFromString(types.StringNull())
	if !result.IsNull() {
		t.Error("expected null for null string")
	}
}

func TestUpgradeLegacyPayloadsFromString_EmptyArray(t *testing.T) {
	result := upgradeLegacyPayloadsFromString(types.StringValue("[]"))
	if !result.IsNull() {
		t.Error("expected null for empty array")
	}
}

func TestUpgradeLegacyPayloadsFromString_InvalidJSON(t *testing.T) {
	result := upgradeLegacyPayloadsFromString(types.StringValue("not-json"))
	if !result.IsNull() {
		t.Error("expected null for invalid JSON")
	}
}

func TestUpgradeLegacyPayloadsFromString_NoSettings(t *testing.T) {
	input := types.StringValue(`[{"payloadType":"com.apple.wifi.managed","payloadIdentifier":"uuid"}]`)

	result := upgradeLegacyPayloadsFromString(input)

	if result.IsNull() {
		t.Fatal("expected non-null result")
	}

	raw, err := helpers.TerraformDynamicToJSON(result)
	if err != nil {
		t.Fatalf("failed to convert: %v", err)
	}
	items := raw.([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 payload, got %d", len(items))
	}
	payload := items[0].(map[string]any)
	if _, ok := payload["settings"]; ok {
		t.Error("expected no settings key when no extra keys")
	}
}

// TestBlueprintResourceModelV3Upgrade covers the v3 to v4 upgrade: apple_declarations stops being
// an object wrapping a `declaration` list and becomes the list.
//
// Without this upgrader, state written by v0.33.0 cannot be decoded against the new schema at all,
// so `terraform plan` fails before it reaches the configuration. It is unit-tested because an
// acceptance test cannot write state from an older provider build.
func TestBlueprintResourceModelV3Upgrade(t *testing.T) {
	declaration := AppleDeclarationModel{
		ChannelType: types.StringValue("SYSTEM"),
		Payload:     types.StringValue(`{"Enabled":true}`),
		Type:        types.StringValue("com.apple.configuration.siri.settings"),
	}

	prior := blueprintResourceModelV3{
		ID:   types.StringValue("6fdc7ca2-b1cb-4053-b0c1-b2316aabf1dc"),
		Name: types.StringValue("Baseline"),
		ComponentBlocks: []componentBlockModelV3{
			{
				Name:              types.StringValue("Declarations"),
				AppleDeclarations: &appleDeclarationsComponentV3{Declarations: []AppleDeclarationModel{declaration}},
				LegacyPayloads: []BlockLegacyPayloadModel{
					{PayloadType: types.StringValue("com.apple.dock"), Settings: types.StringValue(`{"tilesize":48}`)},
				},
			},
			{
				Name:           types.StringValue("Passcode only"),
				PasscodePolicy: &components.PasscodePolicyComponent{},
			},
			{
				Name:              types.StringValue("Empty declarations"),
				AppleDeclarations: &appleDeclarationsComponentV3{},
			},
		},
	}

	upgraded := prior.upgrade()

	if upgraded.ID != prior.ID || upgraded.Name != prior.Name {
		t.Errorf("scalars did not carry across: %+v", upgraded)
	}
	if len(upgraded.ComponentBlocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(upgraded.ComponentBlocks))
	}

	first := upgraded.ComponentBlocks[0]
	if len(first.AppleDeclarations) != 1 {
		t.Fatalf("expected the declaration list to be unwrapped, got %+v", first.AppleDeclarations)
	}
	if first.AppleDeclarations[0] != declaration {
		t.Errorf("declaration changed during the upgrade: %+v", first.AppleDeclarations[0])
	}
	// Everything else in the block has to survive, or the upgrade silently deletes components from
	// the blueprint on the next apply.
	if len(first.LegacyPayloads) != 1 {
		t.Errorf("legacy payloads were dropped: %+v", first.LegacyPayloads)
	}
	if upgraded.ComponentBlocks[1].PasscodePolicy == nil {
		t.Error("a block with no declarations lost its other components")
	}
	if upgraded.ComponentBlocks[1].AppleDeclarations != nil {
		t.Errorf("an absent component became a list: %+v", upgraded.ComponentBlocks[1].AppleDeclarations)
	}
	// An empty list writes no component, so state must not hold one either.
	if upgraded.ComponentBlocks[2].AppleDeclarations != nil {
		t.Errorf("an empty declaration list must upgrade to absent, got %+v", upgraded.ComponentBlocks[2].AppleDeclarations)
	}
}

// TestBlueprintSchemaV3MatchesTheStateItDecodes pins the two things the derived prior schema has to
// get right: apple_declarations in its old object shape, and the version it claims.
func TestBlueprintSchemaV3MatchesTheStateItDecodes(t *testing.T) {
	prior := blueprintSchemaV3(context.Background())

	if prior.Version != 3 {
		t.Errorf("Version = %d, want 3", prior.Version)
	}

	blocks, ok := prior.Attributes["component_blocks"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatal("component_blocks is missing or not a ListNestedAttribute")
	}
	declarations, ok := blocks.NestedObject.Attributes["apple_declarations"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("apple_declarations must be a SingleNestedAttribute in the v3 schema")
	}
	if _, ok := declarations.Attributes["declaration"].(schema.ListNestedAttribute); !ok {
		t.Error("apple_declarations.declaration must be a ListNestedAttribute in the v3 schema")
	}

	// Deriving the prior schema must not leave the live one mutated, or every later plan would see
	// the old shape.
	var resp resource.SchemaResponse
	NewBlueprintResource().(*BlueprintResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	current, ok := resp.Schema.Attributes["component_blocks"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatal("current component_blocks is missing or not a ListNestedAttribute")
	}
	if _, ok := current.NestedObject.Attributes["apple_declarations"].(schema.ListNestedAttribute); !ok {
		t.Error("the live apple_declarations is no longer a ListNestedAttribute")
	}
	if resp.Schema.Version != 4 {
		t.Errorf("current Version = %d, want 4", resp.Schema.Version)
	}
}

// blueprintStateV3 is a verbatim schema-version-3 state document, frozen. It records what schema
// version 3 actually wrote — most importantly a component block carrying apple_declarations in its
// old single-object-wrapping-a-list shape — and must never be regenerated from the live schema:
// regenerating it would make it agree with whatever the schema has since become, which is the one
// thing TestBlueprintSchemaV3DecodesFrozenV3State exists to detect. It populates a first block's
// apple_declarations, legacy_payloads and passcode_policy, leaves a second block's components
// absent, and carries every remaining attribute as an explicit JSON null, the way Terraform writes
// state.
const blueprintStateV3 = `{
  "id": "3f6c1d8e-4b2a-4c77-9e51-0a2b6d9f1c34",
  "name": "Frozen v3 blueprint",
  "description": "A blueprint written by schema version 3.",
  "deployed": true,
  "device_groups": ["3a1f0c72-5d88-4e19-b6a2-9c7e4f10d5bb"],
  "activation_conditions": "@status(os.version) >= \"15.0\"",
  "created": "2026-01-14T09:12:33Z",
  "updated": "2026-02-02T17:45:01Z",
  "deployment_state": "SUCCEEDED",
  "timeouts": {"create": "10m", "read": null, "update": "10m", "delete": null},
  "legacy_payloads": null,
  "raw_component": null,
  "audio_accessory_settings": null,
  "custom_declarations": null,
  "disk_management_settings": null,
  "math_settings": null,
  "passcode_policy": null,
  "safari_bookmarks": null,
  "safari_extensions": null,
  "safari_settings": null,
  "service_background_tasks": null,
  "service_configuration_files": null,
  "software_update": null,
  "software_update_settings": null,
  "component_blocks": [
    {
      "name": "Baseline",
      "activation_conditions": "@status(os.version) >= \"15.0\"",
      "apple_declarations": {
        "declaration": [
          {
            "channel": "DEVICE",
            "payload": "{\"Enabled\":true}",
            "type": "com.apple.configuration.softwareupdate.enforcement.specific"
          },
          {
            "channel": "USER",
            "payload": "{\"PayloadContentIdentifiers\":[]}",
            "type": "com.apple.configuration.services.configuration-files"
          }
        ]
      },
      "legacy_payloads": [
        {
          "payload_type": "com.apple.applicationaccess",
          "settings": "{\"allowCamera\":false}"
        }
      ],
      "passcode_policy": {
        "change_at_next_auth": null,
        "custom_regex_description": null,
        "custom_regex_pattern": null,
        "failed_attempts_reset_in_minutes": null,
        "maximum_failed_attempts": 6,
        "maximum_grace_period_in_minutes": null,
        "maximum_inactivity_in_minutes": null,
        "maximum_passcode_age_in_days": null,
        "minimum_complex_characters": null,
        "minimum_length": 8,
        "passcode_reuse_limit": null,
        "require_alphanumeric_passcode": false,
        "require_complex_passcode": null,
        "require_passcode": true
      },
      "ai_governance": null,
      "raw_component": null,
      "audio_accessory_settings": null,
      "custom_declarations": null,
      "disk_management_settings": null,
      "math_settings": null,
      "safari_bookmarks": null,
      "safari_extensions": null,
      "safari_settings": null,
      "service_background_tasks": null,
      "service_configuration_files": null,
      "software_update": null,
      "software_update_settings": null
    },
    {
      "name": "Empty",
      "activation_conditions": null,
      "apple_declarations": null,
      "legacy_payloads": null,
      "passcode_policy": null,
      "ai_governance": null,
      "raw_component": null,
      "audio_accessory_settings": null,
      "custom_declarations": null,
      "disk_management_settings": null,
      "math_settings": null,
      "safari_bookmarks": null,
      "safari_extensions": null,
      "safari_settings": null,
      "service_background_tasks": null,
      "service_configuration_files": null,
      "software_update": null,
      "software_update_settings": null
    }
  ]
}`

// TestBlueprintSchemaV3DecodesFrozenV3State proves the derived prior schema can still read real
// state. blueprintSchemaV3 builds itself from the live attribute set, so an attribute whose type
// changes — exactly what this change did to apple_declarations — silently reshapes the v3 schema
// out from under state a user has on disk, and a shape assertion against the derived schema cannot
// see it. Running a frozen document through the whole 3 → 4 upgrade can: decoding it with the same
// options the framework uses, then reading it into the frozen blueprintResourceModelV3, so the next
// reshape of any component_blocks attribute fails here instead of inside a user's upgrade.
func TestBlueprintSchemaV3DecodesFrozenV3State(t *testing.T) {
	ctx := context.Background()
	r := NewBlueprintResource().(*BlueprintResource)

	upgrader, ok := r.UpgradeState(ctx)[3]
	if !ok {
		t.Fatal("no state upgrader is registered for schema version 3")
	}

	priorSchema := upgrader.PriorSchema
	if priorSchema == nil {
		t.Fatal("the version 3 upgrader declares no prior schema")
	}

	raw := tfprotov6.RawState{JSON: []byte(blueprintStateV3)}
	priorValue, err := raw.UnmarshalWithOpts(priorSchema.Type().TerraformType(ctx), tfprotov6.UnmarshalOpts{
		ValueFromJSONOpts: tftypes.ValueFromJSONOpts{IgnoreUndefinedAttributes: true},
	})
	if err != nil {
		t.Fatalf("the derived v3 schema no longer decodes real v3 state: %v", err)
	}

	var currentSchema resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &currentSchema)

	req := resource.UpgradeStateRequest{
		State:    &tfsdk.State{Raw: priorValue, Schema: *priorSchema},
		RawState: &raw,
	}
	resp := resource.UpgradeStateResponse{
		State: tfsdk.State{
			Raw:    tftypes.NewValue(currentSchema.Schema.Type().TerraformType(ctx), nil),
			Schema: currentSchema.Schema,
		},
	}
	upgrader.StateUpgrader(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrading frozen v3 state failed: %v", resp.Diagnostics.Errors())
	}

	var upgraded BlueprintResourceModel
	if diags := resp.State.Get(ctx, &upgraded); diags.HasError() {
		t.Fatalf("reading the upgraded state failed: %v", diags.Errors())
	}

	if len(upgraded.ComponentBlocks) != 2 {
		t.Fatalf("ComponentBlocks length = %d, want 2", len(upgraded.ComponentBlocks))
	}
	if got := len(upgraded.ComponentBlocks[0].AppleDeclarations); got != 2 {
		t.Errorf("the first block carries %d declarations, want 2", got)
	}
	if upgraded.ComponentBlocks[1].AppleDeclarations != nil {
		t.Errorf("the second block gained declarations: %+v", upgraded.ComponentBlocks[1].AppleDeclarations)
	}
	if upgraded.ComponentBlocks[0].PasscodePolicy == nil {
		t.Error("the first block lost its passcode_policy")
	}
	if got := len(upgraded.ComponentBlocks[0].LegacyPayloads); got != 1 {
		t.Errorf("the first block carries %d legacy payloads, want 1", got)
	}
}
