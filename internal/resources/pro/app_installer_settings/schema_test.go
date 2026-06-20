// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_installer_settings

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestAppInstallerSettingsResource_Metadata(t *testing.T) {
	r := NewAppInstallerSettingsResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*AppInstallerSettingsResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_app_installer_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_app_installer_settings", resp.TypeName)
	}
}

func TestAppInstallerSettingsResource_Schema(t *testing.T) {
	r := NewAppInstallerSettingsResource()
	var resp resource.SchemaResponse
	r.(*AppInstallerSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "deployment_settings", "end_user_experience", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	dpc := s.Attributes["deployment_settings"]
	if !dpc.IsOptional() || !dpc.IsComputed() || dpc.IsRequired() {
		t.Errorf("deployment_settings must be Optional+Computed, got optional=%v computed=%v required=%v",
			dpc.IsOptional(), dpc.IsComputed(), dpc.IsRequired())
	}

	eux := s.Attributes["end_user_experience"]
	if !eux.IsOptional() || !eux.IsComputed() || eux.IsRequired() {
		t.Errorf("end_user_experience must be Optional+Computed, got optional=%v computed=%v required=%v",
			eux.IsOptional(), eux.IsComputed(), eux.IsRequired())
	}
}

func TestAppInstallerSettingsResource_IdentitySchema(t *testing.T) {
	r := NewAppInstallerSettingsResource()
	var resp resource.IdentitySchemaResponse
	r.(*AppInstallerSettingsResource).IdentitySchema(context.Background(), resource.IdentitySchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("identity schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Error("identity schema missing id attribute")
	}
}

func TestAppInstallerSettingsDataSource_Metadata(t *testing.T) {
	d := NewAppInstallerSettingsDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*AppInstallerSettingsDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_app_installer_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_app_installer_settings", resp.TypeName)
	}
}

func TestAppInstallerSettingsDataSource_Schema(t *testing.T) {
	d := NewAppInstallerSettingsDataSource()
	var resp datasource.SchemaResponse
	d.(*AppInstallerSettingsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "deployment_settings", "end_user_experience", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if !s.Attributes["id"].IsComputed() {
		t.Error("data source id must be computed for singleton")
	}
	// Nested blocks on a DS must be Computed-only — no Optional (they are read-only).
	dpc := s.Attributes["deployment_settings"]
	if dpc.IsOptional() {
		t.Error("data source deployment_settings must not be optional")
	}
	if !dpc.IsComputed() {
		t.Error("data source deployment_settings must be computed")
	}
	eux := s.Attributes["end_user_experience"]
	if eux.IsOptional() {
		t.Error("data source end_user_experience must not be optional")
	}
	if !eux.IsComputed() {
		t.Error("data source end_user_experience must be computed")
	}
}
