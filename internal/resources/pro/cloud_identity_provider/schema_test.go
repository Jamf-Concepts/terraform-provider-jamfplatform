// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package cloud_identity_provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// TestCloudIdentityProviderResource_Metadata verifies the resource type name is
// correctly formed.
func TestCloudIdentityProviderResource_Metadata(t *testing.T) {
	r := NewCloudIdentityProviderResource()
	var resp resource.MetadataResponse
	r.(*CloudIdentityProviderResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if want := "jamfplatform_pro_cloud_identity_provider"; resp.TypeName != want {
		t.Errorf("type name = %q, want %q", resp.TypeName, want)
	}
}

// TestCloudIdentityProviderResource_Schema verifies the schema compiles without
// diagnostics and exposes the expected top-level attributes with the correct
// Required/Computed/WriteOnly properties.
func TestCloudIdentityProviderResource_Schema(t *testing.T) {
	r := NewCloudIdentityProviderResource()
	var resp resource.SchemaResponse
	r.(*CloudIdentityProviderResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	// Top-level attributes must be present.
	for _, name := range []string{"id", "display_name", "provider_name", "google", "entra_id", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing top-level attribute %q", name)
		}
	}

	// provider_name: Required.
	if pn := s.Attributes["provider_name"]; !pn.IsRequired() {
		t.Errorf("provider_name must be required")
	}

	// display_name: Required.
	if dn := s.Attributes["display_name"]; !dn.IsRequired() {
		t.Errorf("display_name must be required")
	}

	// id: Computed-only.
	if id := s.Attributes["id"]; id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only")
	}

	// google: Optional nested block.
	googleAttr, ok := s.Attributes["google"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("google must be SingleNestedAttribute")
	}
	if googleAttr.IsRequired() || !googleAttr.IsOptional() {
		t.Errorf("google must be Optional")
	}

	// Drill into google.server.keystore to verify WriteOnly attrs.
	serverAttr, ok := googleAttr.Attributes["server"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("google.server must be SingleNestedAttribute")
	}
	keystoreAttr, ok := serverAttr.Attributes["keystore"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("google.server.keystore must be SingleNestedAttribute")
	}

	// keystore.file: Optional + Sensitive + WriteOnly.
	fileAttr := keystoreAttr.Attributes["file"]
	if !fileAttr.IsOptional() || !fileAttr.IsSensitive() || !fileAttr.IsWriteOnly() {
		t.Errorf("keystore.file must be Optional+Sensitive+WriteOnly; got optional=%v sensitive=%v writeOnly=%v",
			fileAttr.IsOptional(), fileAttr.IsSensitive(), fileAttr.IsWriteOnly())
	}

	// keystore.password: Optional + Sensitive + WriteOnly.
	passwordAttr := keystoreAttr.Attributes["password"]
	if !passwordAttr.IsOptional() || !passwordAttr.IsSensitive() || !passwordAttr.IsWriteOnly() {
		t.Errorf("keystore.password must be Optional+Sensitive+WriteOnly; got optional=%v sensitive=%v writeOnly=%v",
			passwordAttr.IsOptional(), passwordAttr.IsSensitive(), passwordAttr.IsWriteOnly())
	}

	// keystore.type / subject / expiration_date: Computed-only echoes.
	for _, name := range []string{"type", "subject", "expiration_date"} {
		a := keystoreAttr.Attributes[name]
		if a == nil {
			t.Errorf("keystore.%s is missing", name)
			continue
		}
		if a.IsRequired() || a.IsOptional() || !a.IsComputed() {
			t.Errorf("keystore.%s must be computed-only; got required=%v optional=%v computed=%v",
				name, a.IsRequired(), a.IsOptional(), a.IsComputed())
		}
	}

	// google.mappings: Optional-only (NOT Computed). A Computed nested object
	// backed by a pointer-struct model is decoded as unknown when omitted,
	// which fails at Plan.Get ("target type cannot handle unknown values").
	// Only the scalar leaves are Optional+Computed.
	mappingsAttr := googleAttr.Attributes["mappings"]
	if !mappingsAttr.IsOptional() || mappingsAttr.IsComputed() {
		t.Errorf("google.mappings must be Optional-only (not Computed); got optional=%v computed=%v",
			mappingsAttr.IsOptional(), mappingsAttr.IsComputed())
	}

	// entra_id: Optional nested block.
	entraIDAttr, ok := s.Attributes["entra_id"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("entra_id must be SingleNestedAttribute")
	}
	if entraIDAttr.IsRequired() || !entraIDAttr.IsOptional() {
		t.Errorf("entra_id must be Optional")
	}

	// entra_id.tenant_id: Required.
	if tid := entraIDAttr.Attributes["tenant_id"]; !tid.IsRequired() {
		t.Errorf("entra_id.tenant_id must be Required")
	}

	// entra_id.type / migrated / deprecated_consent: Computed-only echoes.
	for _, name := range []string{"type", "migrated", "deprecated_consent"} {
		a := entraIDAttr.Attributes[name]
		if a == nil {
			t.Errorf("entra_id.%s is missing", name)
			continue
		}
		if a.IsRequired() || a.IsOptional() || !a.IsComputed() {
			t.Errorf("entra_id.%s must be computed-only; got required=%v optional=%v computed=%v",
				name, a.IsRequired(), a.IsOptional(), a.IsComputed())
		}
	}
}

// TestCloudIdentityProviderResource_ConfigValidatorsRegistered confirms exactly
// one resource-level ConfigValidator is registered.
func TestCloudIdentityProviderResource_ConfigValidatorsRegistered(t *testing.T) {
	r := NewCloudIdentityProviderResource().(*CloudIdentityProviderResource)
	if got := len(r.ConfigValidators(context.Background())); got != 1 {
		t.Errorf("expected 1 config validator, got %d", got)
	}
}

// TestCloudIdentityProviderResource_IdentitySchema verifies the identity schema
// has an id attribute (used for import).
func TestCloudIdentityProviderResource_IdentitySchema(t *testing.T) {
	r := NewCloudIdentityProviderResource()
	var resp resource.IdentitySchemaResponse
	r.(*CloudIdentityProviderResource).IdentitySchema(context.Background(), resource.IdentitySchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("identity schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Errorf("identity schema missing id attribute")
	}
}

// TestCloudIdentityProviderDataSource_Schema verifies the singular data source schema.
func TestCloudIdentityProviderDataSource_Schema(t *testing.T) {
	d := NewCloudIdentityProviderDataSource()
	var resp datasource.SchemaResponse
	d.(*CloudIdentityProviderDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	for _, name := range []string{"id", "display_name", "provider_name", "enabled", "provider_description", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("singular data source missing attribute %q", name)
		}
	}

	// id and display_name are Optional+Computed (ExactlyOneOf selector).
	if id := s.Attributes["id"]; !id.IsOptional() || !id.IsComputed() {
		t.Errorf("data source id must be Optional+Computed")
	}
	if dn := s.Attributes["display_name"]; !dn.IsOptional() || !dn.IsComputed() {
		t.Errorf("data source display_name must be Optional+Computed")
	}

	// provider_name, enabled, provider_description: Computed-only.
	for _, name := range []string{"provider_name", "enabled", "provider_description"} {
		a := s.Attributes[name]
		if a.IsRequired() || a.IsOptional() || !a.IsComputed() {
			t.Errorf("data source %s must be computed-only; got required=%v optional=%v computed=%v",
				name, a.IsRequired(), a.IsOptional(), a.IsComputed())
		}
	}
}

// TestCloudIdentityProvidersDataSource_Schema verifies the plural data source schema.
func TestCloudIdentityProvidersDataSource_Schema(t *testing.T) {
	d := NewCloudIdentityProvidersDataSource()
	var resp datasource.SchemaResponse
	d.(*CloudIdentityProvidersDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	for _, name := range []string{"cloud_identity_providers", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("plural data source missing attribute %q", name)
		}
	}

	// cloud_identity_providers: Computed-only list.
	cip := s.Attributes["cloud_identity_providers"]
	if cip.IsRequired() || cip.IsOptional() || !cip.IsComputed() {
		t.Errorf("cloud_identity_providers must be computed-only; got required=%v optional=%v computed=%v",
			cip.IsRequired(), cip.IsOptional(), cip.IsComputed())
	}
}

// TestCloudIdentityProviderListResource_Metadata verifies the list resource
// type name.
func TestCloudIdentityProviderListResource_Metadata(t *testing.T) {
	r := NewCloudIdentityProviderListResource()
	var resp resource.MetadataResponse
	r.(*CloudIdentityProviderListResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if want := "jamfplatform_pro_cloud_identity_provider"; resp.TypeName != want {
		t.Errorf("list resource type name = %q, want %q", resp.TypeName, want)
	}
}

// TestCloudIdentityProviderListResource_Schema verifies the list resource
// config schema exposes the filter block.
func TestCloudIdentityProviderListResource_Schema(t *testing.T) {
	r := NewCloudIdentityProviderListResource()
	var resp list.ListResourceSchemaResponse
	r.(*CloudIdentityProviderListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("list resource schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list resource schema missing 'filter' attribute")
	}
}
