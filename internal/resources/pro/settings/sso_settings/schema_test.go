// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_settings

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// TestSsoSettingsResource_Metadata checks the resource type name.
func TestSsoSettingsResource_Metadata(t *testing.T) {
	r := NewSsoSettingsResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*SsoSettingsResource).Metadata(context.Background(), req, &resp)
	if resp.TypeName != "jamfplatform_pro_sso_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_sso_settings", resp.TypeName)
	}
}

// TestSsoSettingsResource_Schema_TopLevelAttributes asserts the required and
// computed top-level attributes exist with the expected shape.
func TestSsoSettingsResource_Schema_TopLevelAttributes(t *testing.T) {
	r := NewSsoSettingsResource()
	var resp resource.SchemaResponse
	r.(*SsoSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	// configuration_type is the only Required top-level attribute. The
	// top-level bools (sso_enabled, sso_bypass_allowed, ...) are
	// Optional+Computed so the server's defaults flow through cleanly
	// when the user omits a value.
	cfgType, ok := resp.Schema.Attributes["configuration_type"]
	if !ok || !cfgType.IsRequired() {
		t.Errorf("configuration_type must be Required")
	}

	optComputed := []string{
		"sso_enabled",
		"sso_bypass_allowed",
		"sso_for_enrollment_enabled",
		"sso_for_macos_self_service_enabled",
		"enrollment_sso_for_account_driven_enrollment_enabled",
		"group_enrollment_access_enabled",
	}
	for _, name := range optComputed {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Errorf("missing attribute %q", name)
			continue
		}
		if !attr.IsOptional() || !attr.IsComputed() {
			t.Errorf("attribute %q must be Optional+Computed", name)
		}
	}

	if id, ok := resp.Schema.Attributes["id"]; !ok || !id.IsComputed() {
		t.Errorf("id must be Computed")
	}
}

// TestSsoSettingsResource_Schema_NestedBlocks asserts the four nested blocks
// are present and the signing_certificate is a SingleNestedAttribute (not a
// list-of-one) per the WriteOnly nested-attr requirement.
func TestSsoSettingsResource_Schema_NestedBlocks(t *testing.T) {
	r := NewSsoSettingsResource()
	var resp resource.SchemaResponse
	r.(*SsoSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	for _, name := range []string{"oidc_settings", "saml_settings", "enrollment_sso_config"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Errorf("missing nested attribute %q", name)
			continue
		}
		if _, ok := attr.(rschema.SingleNestedAttribute); !ok {
			t.Errorf("attribute %q must be SingleNestedAttribute", name)
		}
	}

	cert, ok := resp.Schema.Attributes["signing_certificate"].(rschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("signing_certificate must be SingleNestedAttribute")
	}
	for _, name := range []string{
		"setup_type", "type", "key", "keystore_file", "keystore_file_name",
		"keystore_password", "keystore_password_wo_version",
		"password", "password_wo_version",
		"serial_number", "subject", "issuer", "expiration", "keys",
	} {
		if _, ok := cert.Attributes[name]; !ok {
			t.Errorf("signing_certificate missing attribute %q", name)
		}
	}

	// WriteOnly + Sensitive on the two password fields.
	for _, name := range []string{"keystore_password", "password"} {
		attr, ok := cert.Attributes[name].(rschema.StringAttribute)
		if !ok {
			t.Errorf("signing_certificate.%s must be StringAttribute", name)
			continue
		}
		if !attr.IsSensitive() {
			t.Errorf("signing_certificate.%s must be Sensitive", name)
		}
		if !attr.IsWriteOnly() {
			t.Errorf("signing_certificate.%s must be WriteOnly", name)
		}
	}
}

// TestSsoSettingsDataSource_Metadata checks the DS type name.
func TestSsoSettingsDataSource_Metadata(t *testing.T) {
	d := NewSsoSettingsDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*SsoSettingsDataSource).Metadata(context.Background(), req, &resp)
	if resp.TypeName != "jamfplatform_pro_sso_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_sso_settings", resp.TypeName)
	}
}

// TestSsoDependenciesDataSource_Metadata checks the dependencies DS type name.
func TestSsoDependenciesDataSource_Metadata(t *testing.T) {
	d := NewSsoDependenciesDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*SsoDependenciesDataSource).Metadata(context.Background(), req, &resp)
	if resp.TypeName != "jamfplatform_pro_sso_dependencies" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_sso_dependencies", resp.TypeName)
	}
}

// TestSsoSpMetadataDataSource_Metadata checks the SP metadata DS type name.
func TestSsoSpMetadataDataSource_Metadata(t *testing.T) {
	d := NewSsoSpMetadataDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*SsoSpMetadataDataSource).Metadata(context.Background(), req, &resp)
	if resp.TypeName != "jamfplatform_pro_sso_sp_metadata" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_sso_sp_metadata", resp.TypeName)
	}
}

// TestExtractIDFromHyperlink covers the hyperlink → id projection used by the
// dependencies data source.
func TestExtractIDFromHyperlink(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"/view/settings/global-management/enrollment-customization/108", "108"},
		{"/view/settings/global-management/enrollment-customization/108/", "108"},
		{"/foo", "foo"},
		{"", ""},
	}
	for _, c := range cases {
		got := extractIDFromHyperlink(c.in)
		if got != c.want {
			t.Errorf("extractIDFromHyperlink(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
