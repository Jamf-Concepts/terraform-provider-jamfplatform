// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package login_page

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// editableAttrNames is the set of Optional+Computed editable attributes on the resource.
var editableAttrNames = []string{
	"include_custom_disclaimer",
	"disclaimer_heading",
	"disclaimer_main_text",
	"action_text",
}

func TestLoginPageSettingsResource_Metadata(t *testing.T) {
	r := NewLoginPageSettingsResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*LoginPageSettingsResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_login_page_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_login_page_settings", resp.TypeName)
	}
}

func TestLoginPageSettingsResource_Schema(t *testing.T) {
	r := NewLoginPageSettingsResource()
	var resp resource.SchemaResponse
	r.(*LoginPageSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	want := append([]string{"id", "timeouts"}, editableAttrNames...)
	for _, name := range want {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	for _, name := range editableAttrNames {
		a := s.Attributes[name]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%s must be optional+computed, got optional=%v computed=%v", name, a.IsOptional(), a.IsComputed())
		}
	}
}

func TestLoginPageSettingsResource_IdentitySchema(t *testing.T) {
	r := NewLoginPageSettingsResource()
	var resp resource.IdentitySchemaResponse
	r.(*LoginPageSettingsResource).IdentitySchema(context.Background(), resource.IdentitySchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("identity schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Errorf("identity schema missing id attribute")
	}
}

func TestLoginPageSettingsDataSource_Metadata(t *testing.T) {
	d := NewLoginPageSettingsDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*LoginPageSettingsDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_login_page_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_login_page_settings", resp.TypeName)
	}
}

func TestLoginPageSettingsDataSource_Schema(t *testing.T) {
	d := NewLoginPageSettingsDataSource()
	var resp datasource.SchemaResponse
	d.(*LoginPageSettingsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	want := append([]string{"id", "timeouts"}, editableAttrNames...)
	for _, name := range want {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if !s.Attributes["id"].IsComputed() {
		t.Errorf("data source id must be computed for singleton")
	}
}
