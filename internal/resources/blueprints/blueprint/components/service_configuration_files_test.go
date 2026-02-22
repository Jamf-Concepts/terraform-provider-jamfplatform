// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestServiceConfigurationFiles_GetIdentifier(t *testing.T) {
	c := &ServiceConfigurationFilesComponent{}
	if c.GetIdentifier() != "com.jamf.ddm.service-configuration-files" {
		t.Errorf("expected 'com.jamf.ddm.service-configuration-files', got %q", c.GetIdentifier())
	}
}

func TestServiceConfigurationFiles_ToRawConfiguration_Full(t *testing.T) {
	c := &ServiceConfigurationFilesComponent{
		ServiceConfigFiles: []ServiceConfigFileModel{
			{
				ServiceType: types.StringValue("com.apple.sshd"),
				DataAssetReference: &ServiceConfigDataAssetRefModel{
					DataURL:    types.StringValue("https://example.com/config.zip"),
					HashSHA256: types.StringValue("abc123"),
				},
			},
		},
	}

	config, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	files, ok := config["serviceConfigFiles"].([]any)
	if !ok {
		t.Fatal("expected serviceConfigFiles to be a slice")
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	file := files[0].(map[string]any)
	if file["ServiceType"] != "com.apple.sshd" {
		t.Errorf("expected ServiceType 'com.apple.sshd', got %v", file["ServiceType"])
	}

	dataRef := file["DataAssetReference"].(map[string]any)
	ref := dataRef["Reference"].(map[string]any)
	if ref["DataURL"] != "https://example.com/config.zip" {
		t.Errorf("expected DataURL 'https://example.com/config.zip', got %v", ref["DataURL"])
	}
	if ref["Hash-SHA-256"] != "abc123" {
		t.Errorf("expected Hash-SHA-256 'abc123', got %v", ref["Hash-SHA-256"])
	}
	if ref["ContentType"] != "application/zip" {
		t.Errorf("expected ContentType 'application/zip', got %v", ref["ContentType"])
	}
}

func TestServiceConfigurationFiles_ToRawConfiguration_Empty(t *testing.T) {
	c := &ServiceConfigurationFilesComponent{}

	config, err := c.ToRawConfiguration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, exists := config["serviceConfigFiles"]; exists {
		t.Error("expected no serviceConfigFiles key for empty component")
	}
}

func TestServiceConfigurationFiles_FromRawConfiguration(t *testing.T) {
	raw := map[string]any{
		"serviceConfigFiles": []any{
			map[string]any{
				"ServiceType": "com.apple.httpd",
				"DataAssetReference": map[string]any{
					"Reference": map[string]any{
						"DataURL":      "https://example.com/httpd.zip",
						"Hash-SHA-256": "xyz789",
						"ContentType":  "application/zip",
					},
				},
			},
		},
	}

	c := &ServiceConfigurationFilesComponent{}
	if err := c.FromRawConfiguration(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(c.ServiceConfigFiles) != 1 {
		t.Fatalf("expected 1 file, got %d", len(c.ServiceConfigFiles))
	}

	file := c.ServiceConfigFiles[0]
	if file.ServiceType.ValueString() != "com.apple.httpd" {
		t.Errorf("expected ServiceType 'com.apple.httpd', got %q", file.ServiceType.ValueString())
	}
	if file.DataAssetReference == nil {
		t.Fatal("expected non-nil DataAssetReference")
	}
	if file.DataAssetReference.DataURL.ValueString() != "https://example.com/httpd.zip" {
		t.Errorf("expected DataURL 'https://example.com/httpd.zip', got %q", file.DataAssetReference.DataURL.ValueString())
	}
	if file.DataAssetReference.HashSHA256.ValueString() != "xyz789" {
		t.Errorf("expected HashSHA256 'xyz789', got %q", file.DataAssetReference.HashSHA256.ValueString())
	}
	if file.DataAssetReference.ContentType.ValueString() != "application/zip" {
		t.Errorf("expected ContentType 'application/zip', got %q", file.DataAssetReference.ContentType.ValueString())
	}
}

func TestServiceConfigurationFiles_FromRawConfiguration_Empty(t *testing.T) {
	c := &ServiceConfigurationFilesComponent{}
	if err := c.FromRawConfiguration(map[string]any{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(c.ServiceConfigFiles) != 0 {
		t.Errorf("expected 0 files, got %d", len(c.ServiceConfigFiles))
	}
}

func TestServiceConfigurationFiles_Roundtrip(t *testing.T) {
	original := &ServiceConfigurationFilesComponent{
		ServiceConfigFiles: []ServiceConfigFileModel{
			{
				ServiceType: types.StringValue("com.apple.smbd"),
				DataAssetReference: &ServiceConfigDataAssetRefModel{
					DataURL:    types.StringValue("https://test.com/smb.zip"),
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

	restored := &ServiceConfigurationFilesComponent{}
	if err := restored.FromRawConfiguration(parsed); err != nil {
		t.Fatalf("FromRawConfiguration error: %v", err)
	}

	if len(restored.ServiceConfigFiles) != 1 {
		t.Fatalf("roundtrip: expected 1 file, got %d", len(restored.ServiceConfigFiles))
	}

	file := restored.ServiceConfigFiles[0]
	if file.ServiceType.ValueString() != "com.apple.smbd" {
		t.Errorf("roundtrip: expected ServiceType 'com.apple.smbd', got %q", file.ServiceType.ValueString())
	}
	if file.DataAssetReference == nil {
		t.Fatal("roundtrip: expected non-nil DataAssetReference")
	}
	if file.DataAssetReference.DataURL.ValueString() != "https://test.com/smb.zip" {
		t.Errorf("roundtrip: expected DataURL 'https://test.com/smb.zip', got %q", file.DataAssetReference.DataURL.ValueString())
	}
}

func TestServiceConfigurationFiles_ToClientComponent(t *testing.T) {
	c := &ServiceConfigurationFilesComponent{
		ServiceConfigFiles: []ServiceConfigFileModel{
			{
				ServiceType: types.StringValue("com.apple.sshd"),
			},
		},
	}

	comp, err := c.ToClientComponent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if comp.Identifier != "com.jamf.ddm.service-configuration-files" {
		t.Errorf("expected identifier 'com.jamf.ddm.service-configuration-files', got %q", comp.Identifier)
	}
	if comp.Configuration == nil {
		t.Fatal("expected non-nil configuration")
	}
}
