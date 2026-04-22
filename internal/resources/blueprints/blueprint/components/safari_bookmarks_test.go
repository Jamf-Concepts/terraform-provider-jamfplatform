// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSafariBookmarks_GetIdentifier(t *testing.T) {
	c := &SafariBookmarksComponent{}
	if c.GetIdentifier() != "com.jamf.ddm.safari-bookmarks" {
		t.Errorf("expected 'com.jamf.ddm.safari-bookmarks', got %q", c.GetIdentifier())
	}
}

func TestSafariBookmarks_ToRawConfiguration_WithBookmarks(t *testing.T) {
	c := &SafariBookmarksComponent{
		ManagedBookmarks: []BookmarkGroupModel{
			{
				GroupIdentifier: types.StringValue("group-1"),
				Title:           types.StringValue("Work Links"),
				Bookmarks: []BookmarkModel{
					{
						Type:  types.StringValue("bookmark"),
						Title: types.StringValue("Jamf"),
						URL:   types.StringValue("https://www.jamf.com"),
					},
					{
						Type:  types.StringValue("folder"),
						Title: types.StringValue("Resources"),
						Folder: []UrlBookmarkModel{
							{
								Title: types.StringValue("Docs"),
								URL:   types.StringValue("https://docs.jamf.com"),
							},
						},
					},
				},
			},
		},
	}

	rawCfg, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var config map[string]any
	if err := json.Unmarshal(rawCfg, &config); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	managedBookmarks, ok := config["ManagedBookmarks"].([]any)
	if !ok {
		t.Fatal("expected ManagedBookmarks to be a slice")
	}
	if len(managedBookmarks) != 1 {
		t.Fatalf("expected 1 group, got %d", len(managedBookmarks))
	}

	group, ok := managedBookmarks[0].(map[string]any)
	if !ok {
		t.Fatal("expected group to be a map")
	}
	if group["GroupIdentifier"] != "group-1" {
		t.Errorf("expected GroupIdentifier 'group-1', got %v", group["GroupIdentifier"])
	}
	if group["Title"] != "Work Links" {
		t.Errorf("expected Title 'Work Links', got %v", group["Title"])
	}

	bookmarks, ok := group["Bookmarks"].([]any)
	if !ok {
		t.Fatal("expected Bookmarks to be a slice")
	}
	if len(bookmarks) != 2 {
		t.Fatalf("expected 2 bookmarks, got %d", len(bookmarks))
	}

	bookmark1 := bookmarks[0].(map[string]any)
	if bookmark1["Type"] != "BOOKMARK" {
		t.Errorf("expected bookmark Type 'BOOKMARK', got %v", bookmark1["Type"])
	}
	if bookmark1["Title"] != "Jamf" {
		t.Errorf("expected bookmark Title 'Jamf', got %v", bookmark1["Title"])
	}
	if bookmark1["URL"] != "https://www.jamf.com" {
		t.Errorf("expected bookmark URL 'https://www.jamf.com', got %v", bookmark1["URL"])
	}

	folder := bookmarks[1].(map[string]any)
	if folder["Type"] != "FOLDER" {
		t.Errorf("expected folder Type 'FOLDER', got %v", folder["Type"])
	}

	folderItems, ok := folder["Folder"].([]any)
	if !ok {
		t.Fatal("expected Folder to be a slice")
	}
	if len(folderItems) != 1 {
		t.Fatalf("expected 1 folder item, got %d", len(folderItems))
	}

	folderItem := folderItems[0].(map[string]any)
	if folderItem["Type"] != "BOOKMARK" {
		t.Errorf("expected folder item Type 'BOOKMARK', got %v", folderItem["Type"])
	}
	if folderItem["Title"] != "Docs" {
		t.Errorf("expected folder item Title 'Docs', got %v", folderItem["Title"])
	}
}

func TestSafariBookmarks_ToRawConfiguration_Empty(t *testing.T) {
	c := &SafariBookmarksComponent{}

	rawCfg, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var config map[string]any
	if err := json.Unmarshal(rawCfg, &config); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	if len(config["ManagedBookmarks"].([]any)) != 0 {
		t.Error("expected empty ManagedBookmarks slice for empty component")
	}
}

func TestSafariBookmarks_FromRawConfiguration(t *testing.T) {
	rawMap := map[string]any{
		"ManagedBookmarks": []any{
			map[string]any{
				"GroupIdentifier": "grp-1",
				"Title":           "Test Group",
				"Bookmarks": []any{
					map[string]any{
						"Type":  "BOOKMARK",
						"Title": "Example",
						"URL":   "https://example.com",
					},
					map[string]any{
						"Type":  "FOLDER",
						"Title": "Sub Folder",
						"Folder": []any{
							map[string]any{
								"Type":  "BOOKMARK",
								"Title": "Nested",
								"URL":   "https://nested.example.com",
							},
						},
					},
				},
			},
		},
	}
	raw, _ := json.Marshal(rawMap)

	c := &SafariBookmarksComponent{}
	if err := c.FromRawConfiguration(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(c.ManagedBookmarks) != 1 {
		t.Fatalf("expected 1 group, got %d", len(c.ManagedBookmarks))
	}

	group := c.ManagedBookmarks[0]
	if group.GroupIdentifier.ValueString() != "grp-1" {
		t.Errorf("expected GroupIdentifier 'grp-1', got %q", group.GroupIdentifier.ValueString())
	}
	if group.Title.ValueString() != "Test Group" {
		t.Errorf("expected Title 'Test Group', got %q", group.Title.ValueString())
	}
	if len(group.Bookmarks) != 2 {
		t.Fatalf("expected 2 bookmarks, got %d", len(group.Bookmarks))
	}

	bm1 := group.Bookmarks[0]
	if bm1.Type.ValueString() != "bookmark" {
		t.Errorf("expected Type 'bookmark', got %q", bm1.Type.ValueString())
	}
	if bm1.Title.ValueString() != "Example" {
		t.Errorf("expected Title 'Example', got %q", bm1.Title.ValueString())
	}
	if bm1.URL.ValueString() != "https://example.com" {
		t.Errorf("expected URL 'https://example.com', got %q", bm1.URL.ValueString())
	}

	bm2 := group.Bookmarks[1]
	if bm2.Type.ValueString() != "folder" {
		t.Errorf("expected Type 'folder', got %q", bm2.Type.ValueString())
	}
	if len(bm2.Folder) != 1 {
		t.Fatalf("expected 1 folder item, got %d", len(bm2.Folder))
	}
	if bm2.Folder[0].Title.ValueString() != "Nested" {
		t.Errorf("expected nested Title 'Nested', got %q", bm2.Folder[0].Title.ValueString())
	}
	if bm2.Folder[0].URL.ValueString() != "https://nested.example.com" {
		t.Errorf("expected nested URL 'https://nested.example.com', got %q", bm2.Folder[0].URL.ValueString())
	}
}

func TestSafariBookmarks_FromRawConfiguration_Empty(t *testing.T) {
	c := &SafariBookmarksComponent{}
	if err := c.FromRawConfiguration(json.RawMessage("{}")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(c.ManagedBookmarks) != 0 {
		t.Errorf("expected 0 groups, got %d", len(c.ManagedBookmarks))
	}
}

func TestSafariBookmarks_Roundtrip(t *testing.T) {
	original := &SafariBookmarksComponent{
		ManagedBookmarks: []BookmarkGroupModel{
			{
				GroupIdentifier: types.StringValue("roundtrip-group"),
				Title:           types.StringValue("Roundtrip"),
				Bookmarks: []BookmarkModel{
					{
						Type:  types.StringValue("bookmark"),
						Title: types.StringValue("Test"),
						URL:   types.StringValue("https://test.com"),
					},
				},
			},
		},
	}

	rawCfg, err := original.ToRawConfiguration()
	if err != nil {
		t.Fatalf("ToRawConfiguration error: %v", err)
	}

	restored := &SafariBookmarksComponent{}
	if err := restored.FromRawConfiguration(rawCfg); err != nil {
		t.Fatalf("FromRawConfiguration error: %v", err)
	}

	if len(restored.ManagedBookmarks) != 1 {
		t.Fatalf("roundtrip: expected 1 group, got %d", len(restored.ManagedBookmarks))
	}

	group := restored.ManagedBookmarks[0]
	if group.GroupIdentifier.ValueString() != "roundtrip-group" {
		t.Errorf("roundtrip: expected GroupIdentifier 'roundtrip-group', got %q", group.GroupIdentifier.ValueString())
	}
	if len(group.Bookmarks) != 1 {
		t.Fatalf("roundtrip: expected 1 bookmark, got %d", len(group.Bookmarks))
	}
	if group.Bookmarks[0].Type.ValueString() != "bookmark" {
		t.Errorf("roundtrip: expected Type 'bookmark', got %q", group.Bookmarks[0].Type.ValueString())
	}
	if group.Bookmarks[0].URL.ValueString() != "https://test.com" {
		t.Errorf("roundtrip: expected URL 'https://test.com', got %q", group.Bookmarks[0].URL.ValueString())
	}
}

func TestSafariBookmarks_ToClientComponent(t *testing.T) {
	c := &SafariBookmarksComponent{
		ManagedBookmarks: []BookmarkGroupModel{
			{
				GroupIdentifier: types.StringValue("g1"),
				Title:           types.StringValue("G1"),
			},
		},
	}

	comp, err := c.ToClientComponent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if comp.Identifier != "com.jamf.ddm.safari-bookmarks" {
		t.Errorf("expected identifier 'com.jamf.ddm.safari-bookmarks', got %q", comp.Identifier)
	}
	if comp.Configuration == nil {
		t.Fatal("expected non-nil configuration")
	}
}
