// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package self_service_macos_settings

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// settingsAttrNames is the set of Optional+Computed user-settable attributes on the resource.
var settingsAttrNames = []string{
	"install_automatically",
	"install_location",
	"login_method",
	"authentication_type",
	"keychain_credential_storage_enabled",
	"fido2_enabled",
	"notifications_enabled",
	"alert_user_approved_mdm",
	"default_landing_page",
	"default_home_category_id",
	"bookmarks_display_name",
}

func TestSelfServiceMacosSettingsResource_Metadata(t *testing.T) {
	r := NewSelfServiceMacosSettingsResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*SelfServiceMacosSettingsResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_self_service_macos_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_self_service_macos_settings", resp.TypeName)
	}
}

func TestSelfServiceMacosSettingsResource_Schema(t *testing.T) {
	r := NewSelfServiceMacosSettingsResource()
	var resp resource.SchemaResponse
	r.(*SelfServiceMacosSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	want := append([]string{"id", "timeouts"}, settingsAttrNames...)
	for _, name := range want {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	for _, name := range settingsAttrNames {
		a := s.Attributes[name]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%s must be optional+computed, got optional=%v computed=%v", name, a.IsOptional(), a.IsComputed())
		}
	}
}

func TestSelfServiceMacosSettingsResource_IdentitySchema(t *testing.T) {
	r := NewSelfServiceMacosSettingsResource()
	var resp resource.IdentitySchemaResponse
	r.(*SelfServiceMacosSettingsResource).IdentitySchema(context.Background(), resource.IdentitySchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("identity schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Errorf("identity schema missing id attribute")
	}
}

func TestSelfServiceMacosSettingsResource_ConfigValidatorsWired(t *testing.T) {
	r := NewSelfServiceMacosSettingsResource()
	validators := r.(*SelfServiceMacosSettingsResource).ConfigValidators(context.Background())
	if len(validators) != 2 {
		t.Errorf("expected 2 ConfigValidators (install-location requirement + category-requires-Browse), got %d", len(validators))
	}
}

func TestSelfServiceMacosSettingsDataSource_Metadata(t *testing.T) {
	d := NewSelfServiceMacosSettingsDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*SelfServiceMacosSettingsDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_self_service_macos_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_self_service_macos_settings", resp.TypeName)
	}
}

func TestSelfServiceMacosSettingsDataSource_Schema(t *testing.T) {
	d := NewSelfServiceMacosSettingsDataSource()
	var resp datasource.SchemaResponse
	d.(*SelfServiceMacosSettingsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	want := append([]string{"id", "timeouts"}, settingsAttrNames...)
	for _, name := range want {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if !s.Attributes["id"].IsComputed() {
		t.Errorf("data source id must be computed for singleton")
	}
}
