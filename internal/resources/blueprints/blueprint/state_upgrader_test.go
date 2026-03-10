// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package blueprint

import (
	"testing"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
