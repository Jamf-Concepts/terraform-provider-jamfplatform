// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_installer

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

const wantTypeName = "jamfplatform_pro_app_installer"

func TestAppInstallerResource_Metadata(t *testing.T) {
	r := NewAppInstallerResource()
	var resp resource.MetadataResponse
	r.(*AppInstallerResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != wantTypeName {
		t.Errorf("expected type name %q, got %q", wantTypeName, resp.TypeName)
	}
}

func TestAppInstallerResource_Schema(t *testing.T) {
	r := NewAppInstallerResource()
	var resp resource.SchemaResponse
	r.(*AppInstallerResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	for _, name := range []string{
		"id", "name", "enabled", "app_title_name", "app_title_id", "deployment_type", "update_behavior",
		"selected_version", "latest_available_version", "title_available_in_ais",
		"version_removed", "category_id", "site_id", "smart_group_id",
		"install_predefined_config_profiles", "trigger_admin_notifications",
		"notification_settings", "self_service_settings", "timeouts",
	} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing top-level attribute %q", name)
		}
	}

	if id := s.Attributes["id"]; id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only")
	}
	// app_title_name is the user-facing reference; app_title_id is resolved from it.
	for _, name := range []string{"name", "app_title_name", "deployment_type", "update_behavior"} {
		if a := s.Attributes[name]; !a.IsRequired() {
			t.Errorf("%s must be required", name)
		}
	}
	// Server-derived fields are plain Computed (no Optional). app_title_id is
	// derived from the mutable app_title_name; selected_version is server-controlled.
	for _, name := range []string{"app_title_id", "latest_available_version", "title_available_in_ais", "version_removed", "selected_version"} {
		a := s.Attributes[name]
		if a.IsRequired() || a.IsOptional() || !a.IsComputed() {
			t.Errorf("%s must be computed-only, got optional=%v required=%v computed=%v", name, a.IsOptional(), a.IsRequired(), a.IsComputed())
		}
	}
	// Sentinel-bearing IDs and trigger_admin_notifications are Optional+Computed.
	for _, name := range []string{"category_id", "site_id", "smart_group_id", "trigger_admin_notifications"} {
		a := s.Attributes[name]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%s must be optional+computed", name)
		}
	}

	ss, ok := s.Attributes["self_service_settings"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("self_service_settings must be a SingleNestedAttribute")
	}
	if ss.IsRequired() || ss.IsComputed() || !ss.IsOptional() {
		t.Errorf("self_service_settings must be optional-only")
	}
	cats, ok := ss.Attributes["categories"].(schema.SetNestedAttribute)
	if !ok {
		t.Fatalf("self_service_settings.categories must be a SetNestedAttribute")
	}
	if _, ok := cats.NestedObject.Attributes["category_id"]; !ok {
		t.Errorf("categories element missing category_id")
	}
	if _, ok := cats.NestedObject.Attributes["featured"]; !ok {
		t.Errorf("categories element missing featured")
	}

	ns, ok := s.Attributes["notification_settings"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("notification_settings must be a SingleNestedAttribute")
	}
	if ns.IsRequired() || ns.IsComputed() || !ns.IsOptional() {
		t.Errorf("notification_settings must be optional-only")
	}
}

func TestAppInstallerDataSource_Metadata(t *testing.T) {
	d := NewAppInstallerDataSource()
	var resp datasource.MetadataResponse
	d.(*AppInstallerDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != wantTypeName {
		t.Errorf("expected type name %q, got %q", wantTypeName, resp.TypeName)
	}
}

func TestAppInstallerDataSource_ConfigValidators(t *testing.T) {
	d := NewAppInstallerDataSource().(*AppInstallerDataSource)
	if got := d.ConfigValidators(context.Background()); len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestAppInstallerListResource_Schema(t *testing.T) {
	r := NewAppInstallerListResource()
	var resp list.ListResourceSchemaResponse
	r.(*AppInstallerListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
