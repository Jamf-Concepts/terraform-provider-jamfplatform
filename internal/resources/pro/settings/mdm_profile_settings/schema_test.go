// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mdm_profile_settings

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// settingAttrs lists the six tenant-controlled attributes shared by the resource and
// data source schemas.
var settingAttrs = []string{
	"auto_renew_computer_profile_when_ca_renewed",
	"auto_renew_computer_profile_before_expiry",
	"computer_profile_expiration_limit_days",
	"auto_renew_mobile_device_profile_when_ca_renewed",
	"auto_renew_mobile_device_profile_before_expiry",
	"mobile_device_profile_expiration_limit_days",
}

func TestMDMProfileSettingsResource_Metadata(t *testing.T) {
	r := NewMDMProfileSettingsResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*MDMProfileSettingsResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_mdm_profile_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_mdm_profile_settings", resp.TypeName)
	}
}

func TestMDMProfileSettingsResource_Schema(t *testing.T) {
	r := NewMDMProfileSettingsResource()
	var resp resource.SchemaResponse
	r.(*MDMProfileSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range append([]string{"id", "timeouts"}, settingAttrs...) {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	for _, name := range settingAttrs {
		attr := s.Attributes[name]
		if !attr.IsOptional() || !attr.IsComputed() {
			t.Errorf("%q must be Optional+Computed, got optional=%v computed=%v", name, attr.IsOptional(), attr.IsComputed())
		}
	}
}

func TestMDMProfileSettingsResource_IdentitySchema(t *testing.T) {
	r := NewMDMProfileSettingsResource()
	var resp resource.IdentitySchemaResponse
	r.(*MDMProfileSettingsResource).IdentitySchema(context.Background(), resource.IdentitySchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("identity schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Errorf("identity schema missing id attribute")
	}
}

func TestMDMProfileSettingsDataSource_Metadata(t *testing.T) {
	d := NewMDMProfileSettingsDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*MDMProfileSettingsDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_mdm_profile_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_mdm_profile_settings", resp.TypeName)
	}
}

func TestMDMProfileSettingsDataSource_Schema(t *testing.T) {
	d := NewMDMProfileSettingsDataSource()
	var resp datasource.SchemaResponse
	d.(*MDMProfileSettingsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range append([]string{"id", "timeouts"}, settingAttrs...) {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if !s.Attributes["id"].IsComputed() {
		t.Errorf("data source id must be computed for singleton")
	}
	for _, name := range settingAttrs {
		if !s.Attributes[name].IsComputed() {
			t.Errorf("data source %q must be computed", name)
		}
	}
}
