// Copyright 2026 Jamf Software LLC.

package blueprint

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
