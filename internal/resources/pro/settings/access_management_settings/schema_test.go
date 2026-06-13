// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package access_management_settings

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestAccessManagementSettingsResource_Metadata(t *testing.T) {
	r := NewAccessManagementSettingsResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*AccessManagementSettingsResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_access_management_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_access_management_settings", resp.TypeName)
	}
}

func TestAccessManagementSettingsResource_Schema(t *testing.T) {
	r := NewAccessManagementSettingsResource()
	var resp resource.SchemaResponse
	r.(*AccessManagementSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "automated_device_enrollment_server_uuid", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	uuid := s.Attributes["automated_device_enrollment_server_uuid"]
	if !uuid.IsOptional() || !uuid.IsComputed() {
		t.Errorf("automated_device_enrollment_server_uuid must be optional+computed, got optional=%v computed=%v", uuid.IsOptional(), uuid.IsComputed())
	}
}

func TestAccessManagementSettingsResource_IdentitySchema(t *testing.T) {
	r := NewAccessManagementSettingsResource()
	var resp resource.IdentitySchemaResponse
	r.(*AccessManagementSettingsResource).IdentitySchema(context.Background(), resource.IdentitySchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("identity schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Errorf("identity schema missing id attribute")
	}
}

func TestAccessManagementSettingsDataSource_Metadata(t *testing.T) {
	d := NewAccessManagementSettingsDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*AccessManagementSettingsDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_access_management_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_access_management_settings", resp.TypeName)
	}
}

func TestAccessManagementSettingsDataSource_Schema(t *testing.T) {
	d := NewAccessManagementSettingsDataSource()
	var resp datasource.SchemaResponse
	d.(*AccessManagementSettingsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "automated_device_enrollment_server_uuid", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if !s.Attributes["id"].IsComputed() {
		t.Errorf("data source id must be computed for singleton")
	}
}
