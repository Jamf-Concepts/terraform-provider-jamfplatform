// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package dock_item

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildDockItemInput_AllFieldsSet(t *testing.T) {
	plan := DockItemResourceModel{
		Name: types.StringValue("Calculator"),
		Type: types.StringValue("App"),
		Path: types.StringValue("/Applications/Calculator.app"),
		// Contents is server-computed — even if a stale state value sat here,
		// the input builder must drop it before writing.
		Contents: types.StringValue("stale plist value"),
	}
	got := buildDockItemInput(plan)

	if got.Name == nil || *got.Name != "Calculator" {
		t.Errorf("expected Name=Calculator, got %v", got.Name)
	}
	if got.Type == nil || *got.Type != "App" {
		t.Errorf("expected Type=App, got %v", got.Type)
	}
	if got.Path == nil || *got.Path != "/Applications/Calculator.app" {
		t.Errorf("expected Path set, got %v", got.Path)
	}
	if got.Contents != nil {
		t.Errorf("Contents must NEVER be sent on writes (server recomputes) — got %v", *got.Contents)
	}
	if got.ID != nil {
		t.Errorf("expected nil ID on write payload, got %d", *got.ID)
	}
}

func TestBuildDockItemInput_NullFieldsOmitted(t *testing.T) {
	plan := DockItemResourceModel{
		Name: types.StringNull(),
		Type: types.StringNull(),
		Path: types.StringNull(),
	}
	got := buildDockItemInput(plan)

	if got.Name != nil {
		t.Errorf("null Name must serialise to nil, got %v", *got.Name)
	}
	if got.Type != nil {
		t.Errorf("null Type must serialise to nil, got %v", *got.Type)
	}
	if got.Path != nil {
		t.Errorf("null Path must serialise to nil, got %v", *got.Path)
	}
}

func TestBuildDockItemInput_FileType(t *testing.T) {
	plan := DockItemResourceModel{
		Name: types.StringValue("Readme"),
		Type: types.StringValue("File"),
		Path: types.StringValue("file://localhost/Library/Documentation/README.txt"),
	}
	got := buildDockItemInput(plan)

	if got.Type == nil || *got.Type != "File" {
		t.Errorf("expected Type=File, got %v", got.Type)
	}
	if got.Path == nil || *got.Path != "file://localhost/Library/Documentation/README.txt" {
		t.Errorf("expected URI Path, got %v", got.Path)
	}
}

func TestBuildDockItemInput_FolderType(t *testing.T) {
	plan := DockItemResourceModel{
		Name: types.StringValue("Downloads"),
		Type: types.StringValue("Folder"),
		Path: types.StringValue("~/Downloads"),
	}
	got := buildDockItemInput(plan)

	if got.Type == nil || *got.Type != "Folder" {
		t.Errorf("expected Type=Folder, got %v", got.Type)
	}
}
