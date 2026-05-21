// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// --- Resource ---

func TestDeviceGroupResource_Metadata(t *testing.T) {
	r := NewDeviceGroupResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*DeviceGroupResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_device_group" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_device_group", resp.TypeName)
	}
}

func TestDeviceGroupResource_Schema(t *testing.T) {
	r := NewDeviceGroupResource()
	req := resource.SchemaRequest{}
	var resp resource.SchemaResponse
	r.(*DeviceGroupResource).Schema(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	if s.Version != 1 {
		t.Errorf("expected schema version 1, got %d", s.Version)
	}

	requiredAttrs := []string{"name", "device_type", "group_type"}
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

	computedAttrs := []string{"id", "jamf_pro_id", "member_count"}
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

	jamfProID, ok := s.Attributes["jamf_pro_id"].(resourceschema.StringAttribute)
	if !ok {
		t.Fatal("jamf_pro_id should be a StringAttribute")
	}
	if jamfProID.IsRequired() || jamfProID.IsOptional() {
		t.Error("jamf_pro_id must be Computed-only (not Required, not Optional)")
	}
	if len(jamfProID.PlanModifiers) == 0 {
		t.Error("jamf_pro_id must carry a UseStateForUnknown plan modifier")
	}

	optionalAttrs := []string{"description", "members", "criteria", "timeouts"}
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

func TestDeviceGroupResource_SchemaAttributes(t *testing.T) {
	r := NewDeviceGroupResource()
	req := resource.SchemaRequest{}
	var resp resource.SchemaResponse
	r.(*DeviceGroupResource).Schema(context.Background(), req, &resp)

	criteria, ok := resp.Schema.Attributes["criteria"]
	if !ok {
		t.Fatal("missing criteria attribute")
	}

	nested, ok := criteria.(resourceschema.SetNestedAttribute)
	if !ok {
		t.Fatal("criteria should be a SetNestedAttribute")
	}

	expectedCriteriaAttrs := []string{"order", "criteria", "operator", "value", "and_or", "has_opening_parenthesis", "has_closing_parenthesis"}
	for _, name := range expectedCriteriaAttrs {
		if _, ok := nested.NestedObject.Attributes[name]; !ok {
			t.Errorf("criteria nested object missing attribute %q", name)
		}
	}
}

// --- Data Source ---

func TestDeviceGroupDataSource_Metadata(t *testing.T) {
	d := NewDeviceGroupDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*DeviceGroupDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_device_group" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_device_group", resp.TypeName)
	}
}

func TestDeviceGroupDataSource_Schema(t *testing.T) {
	d := NewDeviceGroupDataSource()
	req := datasource.SchemaRequest{}
	var resp datasource.SchemaResponse
	d.(*DeviceGroupDataSource).Schema(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	requiredAttrs := []string{"id"}
	for _, name := range requiredAttrs {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Errorf("missing required attribute %q", name)
			continue
		}
		if !attr.IsRequired() {
			t.Errorf("attribute %q should be required", name)
		}
	}

	computedAttrs := []string{"name", "description", "device_type", "group_type", "jamf_pro_id", "member_count", "members"}
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

	criteria, ok := resp.Schema.Attributes["criteria"]
	if !ok {
		t.Fatal("missing criteria attribute")
	}
	if _, ok := criteria.(datasourceschema.ListNestedAttribute); !ok {
		t.Error("criteria should be a ListNestedAttribute in data source")
	}
}

// --- List Resource ---

func TestDeviceGroupListResource_Metadata(t *testing.T) {
	r := NewDeviceGroupListResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*DeviceGroupListResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_device_group" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_device_group", resp.TypeName)
	}
}
