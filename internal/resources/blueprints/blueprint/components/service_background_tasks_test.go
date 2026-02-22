// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestServiceBackgroundTasks_GetIdentifier(t *testing.T) {
	c := &ServiceBackgroundTasksComponent{}
	if c.GetIdentifier() != "com.jamf.ddm.service-background-tasks" {
		t.Errorf("expected 'com.jamf.ddm.service-background-tasks', got %q", c.GetIdentifier())
	}
}

func TestServiceBackgroundTasks_ToRawConfiguration_Full(t *testing.T) {
	c := &ServiceBackgroundTasksComponent{
		BackgroundTasks: []ServiceBackgroundTaskModel{
			{
				TaskType:        types.StringValue("com.example.task"),
				TaskDescription: types.StringValue("Test task"),
				ExecutableAssetReference: &DataAssetRefModel{
					DataURL:    types.StringValue("https://example.com/exec.zip"),
					HashSHA256: types.StringValue("abc123"),
				},
				LaunchdConfigurations: []LaunchdItemModel{
					{
						Context: types.StringValue("daemon"),
						FileAssetReference: &DataAssetRefModel{
							DataURL:     types.StringValue("https://example.com/plist.zip"),
							HashSHA256:  types.StringValue("def456"),
							ContentType: types.StringValue("application/xml"),
						},
					},
				},
			},
		},
	}

	config, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tasks, ok := config["backgroundTasks"].([]any)
	if !ok {
		t.Fatal("expected backgroundTasks to be a slice")
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	task := tasks[0].(map[string]any)
	if task["TaskType"] != "com.example.task" {
		t.Errorf("expected TaskType 'com.example.task', got %v", task["TaskType"])
	}
	if task["TaskDescription"] != "Test task" {
		t.Errorf("expected TaskDescription 'Test task', got %v", task["TaskDescription"])
	}

	execRef, ok := task["ExecutableAssetReference"].(map[string]any)
	if !ok {
		t.Fatal("expected ExecutableAssetReference to be a map")
	}
	ref := execRef["Reference"].(map[string]any)
	if ref["DataURL"] != "https://example.com/exec.zip" {
		t.Errorf("expected DataURL 'https://example.com/exec.zip', got %v", ref["DataURL"])
	}
	if ref["Hash-SHA-256"] != "abc123" {
		t.Errorf("expected Hash-SHA-256 'abc123', got %v", ref["Hash-SHA-256"])
	}
	if ref["ContentType"] != "application/zip" {
		t.Errorf("expected ContentType 'application/zip', got %v", ref["ContentType"])
	}

	launchdConfigs, ok := task["LaunchdConfigurations"].([]any)
	if !ok {
		t.Fatal("expected LaunchdConfigurations to be a slice")
	}
	if len(launchdConfigs) != 1 {
		t.Fatalf("expected 1 launchd config, got %d", len(launchdConfigs))
	}

	launchd := launchdConfigs[0].(map[string]any)
	if launchd["Context"] != "daemon" {
		t.Errorf("expected Context 'daemon', got %v", launchd["Context"])
	}

	fileRef := launchd["FileAssetReference"].(map[string]any)
	fileRefInner := fileRef["Reference"].(map[string]any)
	if fileRefInner["DataURL"] != "https://example.com/plist.zip" {
		t.Errorf("expected file DataURL 'https://example.com/plist.zip', got %v", fileRefInner["DataURL"])
	}
}

func TestServiceBackgroundTasks_ToRawConfiguration_Empty(t *testing.T) {
	c := &ServiceBackgroundTasksComponent{}

	config, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, exists := config["backgroundTasks"]; exists {
		t.Error("expected no backgroundTasks key for empty component")
	}
}

func TestServiceBackgroundTasks_FromRawConfiguration(t *testing.T) {
	raw := map[string]any{
		"backgroundTasks": []any{
			map[string]any{
				"TaskType":        "com.example.bg",
				"TaskDescription": "Background task",
				"ExecutableAssetReference": map[string]any{
					"Reference": map[string]any{
						"DataURL":      "https://example.com/bin.zip",
						"Hash-SHA-256": "hash123",
						"ContentType":  "application/zip",
					},
				},
				"LaunchdConfigurations": []any{
					map[string]any{
						"Context": "agent",
						"FileAssetReference": map[string]any{
							"Reference": map[string]any{
								"DataURL":      "https://example.com/config.zip",
								"Hash-SHA-256": "hash456",
								"ContentType":  "application/xml",
							},
						},
					},
				},
			},
		},
	}

	c := &ServiceBackgroundTasksComponent{}
	if err := c.FromRawConfiguration(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(c.BackgroundTasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(c.BackgroundTasks))
	}

	task := c.BackgroundTasks[0]
	if task.TaskType.ValueString() != "com.example.bg" {
		t.Errorf("expected TaskType 'com.example.bg', got %q", task.TaskType.ValueString())
	}
	if task.TaskDescription.ValueString() != "Background task" {
		t.Errorf("expected TaskDescription 'Background task', got %q", task.TaskDescription.ValueString())
	}
	if task.ExecutableAssetReference == nil {
		t.Fatal("expected non-nil ExecutableAssetReference")
	}
	if task.ExecutableAssetReference.DataURL.ValueString() != "https://example.com/bin.zip" {
		t.Errorf("expected exec DataURL 'https://example.com/bin.zip', got %q", task.ExecutableAssetReference.DataURL.ValueString())
	}
	if task.ExecutableAssetReference.HashSHA256.ValueString() != "hash123" {
		t.Errorf("expected exec HashSHA256 'hash123', got %q", task.ExecutableAssetReference.HashSHA256.ValueString())
	}
	if task.ExecutableAssetReference.ContentType.ValueString() != "application/zip" {
		t.Errorf("expected exec ContentType 'application/zip', got %q", task.ExecutableAssetReference.ContentType.ValueString())
	}

	if len(task.LaunchdConfigurations) != 1 {
		t.Fatalf("expected 1 launchd config, got %d", len(task.LaunchdConfigurations))
	}
	launchd := task.LaunchdConfigurations[0]
	if launchd.Context.ValueString() != "agent" {
		t.Errorf("expected Context 'agent', got %q", launchd.Context.ValueString())
	}
	if launchd.FileAssetReference == nil {
		t.Fatal("expected non-nil FileAssetReference")
	}
	if launchd.FileAssetReference.DataURL.ValueString() != "https://example.com/config.zip" {
		t.Errorf("expected file DataURL 'https://example.com/config.zip', got %q", launchd.FileAssetReference.DataURL.ValueString())
	}
}

func TestServiceBackgroundTasks_FromRawConfiguration_Empty(t *testing.T) {
	c := &ServiceBackgroundTasksComponent{}
	if err := c.FromRawConfiguration(map[string]any{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(c.BackgroundTasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(c.BackgroundTasks))
	}
}

func TestServiceBackgroundTasks_Roundtrip(t *testing.T) {
	original := &ServiceBackgroundTasksComponent{
		BackgroundTasks: []ServiceBackgroundTaskModel{
			{
				TaskType:        types.StringValue("com.test.task"),
				TaskDescription: types.StringValue("Roundtrip task"),
				ExecutableAssetReference: &DataAssetRefModel{
					DataURL:    types.StringValue("https://test.com/exec.zip"),
					HashSHA256: types.StringValue("roundtrip-hash"),
				},
			},
		},
	}

	config, err := original.ToRawConfiguration()
	if err != nil {
		t.Fatalf("ToRawConfiguration error: %v", err)
	}

	jsonBytes, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	restored := &ServiceBackgroundTasksComponent{}
	if err := restored.FromRawConfiguration(parsed); err != nil {
		t.Fatalf("FromRawConfiguration error: %v", err)
	}

	if len(restored.BackgroundTasks) != 1 {
		t.Fatalf("roundtrip: expected 1 task, got %d", len(restored.BackgroundTasks))
	}

	task := restored.BackgroundTasks[0]
	if task.TaskType.ValueString() != "com.test.task" {
		t.Errorf("roundtrip: expected TaskType 'com.test.task', got %q", task.TaskType.ValueString())
	}
	if task.ExecutableAssetReference == nil {
		t.Fatal("roundtrip: expected non-nil ExecutableAssetReference")
	}
	if task.ExecutableAssetReference.DataURL.ValueString() != "https://test.com/exec.zip" {
		t.Errorf("roundtrip: expected exec DataURL 'https://test.com/exec.zip', got %q", task.ExecutableAssetReference.DataURL.ValueString())
	}
}

func TestServiceBackgroundTasks_ToClientComponent(t *testing.T) {
	c := &ServiceBackgroundTasksComponent{
		BackgroundTasks: []ServiceBackgroundTaskModel{
			{
				TaskType: types.StringValue("com.example.task"),
			},
		},
	}

	comp, err := c.ToClientComponent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if comp.Identifier != "com.jamf.ddm.service-background-tasks" {
		t.Errorf("expected identifier 'com.jamf.ddm.service-background-tasks', got %q", comp.Identifier)
	}
	if comp.Configuration == nil {
		t.Fatal("expected non-nil configuration")
	}
}
