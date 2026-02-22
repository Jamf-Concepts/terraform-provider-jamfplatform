// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package blueprint

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// --- Resource ---

func TestBlueprintResource_Metadata(t *testing.T) {
	r := NewBlueprintResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*BlueprintResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_blueprints_blueprint" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_blueprints_blueprint", resp.TypeName)
	}
}

func TestBlueprintResource_Schema(t *testing.T) {
	r := NewBlueprintResource()
	req := resource.SchemaRequest{}
	var resp resource.SchemaResponse
	r.(*BlueprintResource).Schema(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	if s.Version != 1 {
		t.Errorf("expected schema version 1, got %d", s.Version)
	}

	requiredAttrs := []string{"name", "deployed", "device_groups"}
	for _, name := range requiredAttrs {
		attr, ok := s.Attributes[name]
		if !ok {
			t.Errorf("missing required attribute %q", name)
			continue
		}
		if !attr.IsRequired() {
			t.Errorf("attribute %q should be required", name)
		}
	}

	computedAttrs := []string{"id", "created", "updated", "deployment_state"}
	for _, name := range computedAttrs {
		attr, ok := s.Attributes[name]
		if !ok {
			t.Errorf("missing computed attribute %q", name)
			continue
		}
		if !attr.IsComputed() {
			t.Errorf("attribute %q should be computed", name)
		}
	}

	optionalAttrs := []string{"description", "legacy_payloads", "timeouts", "raw_component"}
	for _, name := range optionalAttrs {
		attr, ok := s.Attributes[name]
		if !ok {
			t.Errorf("missing optional attribute %q", name)
			continue
		}
		if !attr.IsOptional() {
			t.Errorf("attribute %q should be optional", name)
		}
	}
}

func TestBlueprintResource_SchemaComponentAttributes(t *testing.T) {
	r := NewBlueprintResource()
	req := resource.SchemaRequest{}
	var resp resource.SchemaResponse
	r.(*BlueprintResource).Schema(context.Background(), req, &resp)

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
	}
	for _, name := range componentAttrs {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Errorf("missing component attribute %q", name)
			continue
		}
		if _, ok := attr.(resourceschema.SingleNestedAttribute); !ok {
			t.Errorf("component attribute %q should be SingleNestedAttribute", name)
		}
	}
}

// --- Data Source ---

func TestBlueprintDataSource_Metadata(t *testing.T) {
	d := NewBlueprintDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*BlueprintDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_blueprints_blueprint" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_blueprints_blueprint", resp.TypeName)
	}
}

func TestBlueprintDataSource_Schema(t *testing.T) {
	d := NewBlueprintDataSource()
	req := datasource.SchemaRequest{}
	var resp datasource.SchemaResponse
	d.(*BlueprintDataSource).Schema(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	optionalAttrs := []string{"id", "name"}
	for _, name := range optionalAttrs {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Errorf("missing optional attribute %q", name)
			continue
		}
		if !attr.IsOptional() {
			t.Errorf("attribute %q should be optional", name)
		}
	}

	computedAttrs := []string{"blueprint_id", "description", "created", "updated", "deployment_state", "device_groups"}
	for _, name := range computedAttrs {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Errorf("missing computed attribute %q", name)
			continue
		}
		if !attr.IsComputed() {
			t.Errorf("attribute %q should be computed", name)
		}
	}
}

// --- List Resource ---

func TestBlueprintListResource_Metadata(t *testing.T) {
	r := NewBlueprintListResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*BlueprintListResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_blueprints_blueprint" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_blueprints_blueprint", resp.TypeName)
	}
}
