// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package self_service_plus_settings

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestSelfServicePlusSettingsResource_Metadata(t *testing.T) {
	r := NewSelfServicePlusSettingsResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*SelfServicePlusSettingsResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_self_service_plus_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_self_service_plus_settings", resp.TypeName)
	}
}

func TestSelfServicePlusSettingsResource_Schema(t *testing.T) {
	r := NewSelfServicePlusSettingsResource()
	var resp resource.SchemaResponse
	r.(*SelfServicePlusSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "enabled", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	enabled := s.Attributes["enabled"]
	if !enabled.IsRequired() {
		t.Errorf("enabled must be required")
	}
}

func TestSelfServicePlusSettingsResource_IdentitySchema(t *testing.T) {
	r := NewSelfServicePlusSettingsResource()
	var resp resource.IdentitySchemaResponse
	r.(*SelfServicePlusSettingsResource).IdentitySchema(context.Background(), resource.IdentitySchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("identity schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Errorf("identity schema missing id attribute")
	}
}

func TestSelfServicePlusSettingsDataSource_Metadata(t *testing.T) {
	d := NewSelfServicePlusSettingsDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*SelfServicePlusSettingsDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_self_service_plus_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_self_service_plus_settings", resp.TypeName)
	}
}

func TestSelfServicePlusSettingsDataSource_Schema(t *testing.T) {
	d := NewSelfServicePlusSettingsDataSource()
	var resp datasource.SchemaResponse
	d.(*SelfServicePlusSettingsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "enabled", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if !s.Attributes["id"].IsComputed() {
		t.Errorf("data source id must be computed for singleton")
	}
}
