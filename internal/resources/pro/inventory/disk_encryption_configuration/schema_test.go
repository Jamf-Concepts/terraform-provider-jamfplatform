// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package disk_encryption_configuration

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestDiskEncryptionConfigurationResource_Metadata(t *testing.T) {
	r := NewDiskEncryptionConfigurationResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*DiskEncryptionConfigurationResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_disk_encryption_configuration" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_disk_encryption_configuration", resp.TypeName)
	}
}

func TestDiskEncryptionConfigurationResource_Schema(t *testing.T) {
	r := NewDiskEncryptionConfigurationResource()
	var resp resource.SchemaResponse
	r.(*DiskEncryptionConfigurationResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	required := []string{"id", "name", "key_type", "file_vault_enabled_users", "institutional_recovery_key", "timeouts"}
	for _, name := range required {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	for _, req := range []string{"name", "key_type", "file_vault_enabled_users"} {
		if !s.Attributes[req].IsRequired() {
			t.Errorf("%q must be required", req)
		}
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	// The institutional_recovery_key block must be Optional-only (typed
	// pointer model — see STYLE_GUIDE Optional-only typed pointer rule).
	irk, ok := s.Attributes["institutional_recovery_key"].(rschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("institutional_recovery_key must be a SingleNestedAttribute")
	}
	if !irk.IsOptional() {
		t.Errorf("institutional_recovery_key must be Optional")
	}
	if irk.IsComputed() {
		t.Errorf("institutional_recovery_key must NOT be Computed — framework cannot fit Unknown into typed pointer model")
	}

	// Inner-field shape.
	innerNames := []string{"key", "certificate_type", "password", "password_sha256", "data"}
	for _, name := range innerNames {
		if _, ok := irk.Attributes[name]; !ok {
			t.Errorf("institutional_recovery_key missing attribute %q", name)
		}
	}

	// `password` must be Optional + Sensitive.
	pw := irk.Attributes["password"]
	if !pw.IsOptional() {
		t.Errorf("institutional_recovery_key.password must be Optional")
	}
	if !pw.IsSensitive() {
		t.Errorf("institutional_recovery_key.password must be Sensitive — it is a write-only credential")
	}

	// `password_sha256` is the masked sentinel — Computed-only.
	ph := irk.Attributes["password_sha256"]
	if ph.IsRequired() || ph.IsOptional() || !ph.IsComputed() {
		t.Errorf("password_sha256 must be Computed-only (it is the server's redaction sentinel, not a real hash)")
	}

	// `key` is server-derived — Computed-only.
	key := irk.Attributes["key"]
	if key.IsRequired() || key.IsOptional() || !key.IsComputed() {
		t.Errorf("\"key\" must be Computed-only (server-derived Subject DN)")
	}

	// `certificate_type` must be Required — classic POST endpoint rejects
	// an IRK block without it (server: "Certificate type is required if
	// a recovery key is specified").
	ct := irk.Attributes["certificate_type"]
	if !ct.IsRequired() {
		t.Errorf("\"certificate_type\" must be Required — server rejects POST/PUT without it")
	}

	// `data` is Required + Sensitive (PKCS12 carries the private key).
	d := irk.Attributes["data"]
	if !d.IsRequired() {
		t.Errorf("institutional_recovery_key.data must be Required — the IRK block is meaningless without a cert payload")
	}
	if !d.IsSensitive() {
		t.Errorf("institutional_recovery_key.data must be Sensitive — PKCS12 payloads contain the wrapped private key")
	}
}

func TestDiskEncryptionConfigurationResource_ConfigValidators(t *testing.T) {
	r := NewDiskEncryptionConfigurationResource().(*DiskEncryptionConfigurationResource)
	got := r.ConfigValidators(context.Background())
	if len(got) != 1 {
		t.Fatalf("expected 1 config validator (institutionalKeyTypeRequiresIRKConfigValidator), got %d", len(got))
	}
}

func TestDiskEncryptionConfigurationDataSource_Metadata(t *testing.T) {
	d := NewDiskEncryptionConfigurationDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*DiskEncryptionConfigurationDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_disk_encryption_configuration" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_disk_encryption_configuration", resp.TypeName)
	}
}

func TestDiskEncryptionConfigurationDataSource_Schema(t *testing.T) {
	d := NewDiskEncryptionConfigurationDataSource()
	var resp datasource.SchemaResponse
	d.(*DiskEncryptionConfigurationDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "name", "key_type", "file_vault_enabled_users", "institutional_recovery_key", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("data source missing attribute %q", name)
		}
	}

	// Data source must NOT carry a write-only `password` attribute on
	// the nested IRK block — the wire never returns plaintext on read.
	irk, ok := s.Attributes["institutional_recovery_key"].(dsschema.SingleNestedAttribute)
	if !ok {
		t.Fatal("data source institutional_recovery_key must be a SingleNestedAttribute")
	}
	if _, hasPassword := irk.Attributes["password"]; hasPassword {
		t.Errorf("data source institutional_recovery_key must not expose `password` — the wire never returns plaintext")
	}
	if !irk.IsComputed() {
		t.Errorf("data source institutional_recovery_key must be Computed-only")
	}
	// password_sha256 surfaces on the data source so callers can detect
	// "is a password set" out of band.
	if _, ok := irk.Attributes["password_sha256"]; !ok {
		t.Errorf("data source institutional_recovery_key.password_sha256 must surface")
	}
}

func TestDiskEncryptionConfigurationDataSource_ConfigValidators_ExactlyOneSelector(t *testing.T) {
	d := NewDiskEncryptionConfigurationDataSource().(*DiskEncryptionConfigurationDataSource)
	got := d.ConfigValidators(context.Background())
	if len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestDiskEncryptionConfigurationListResource_Metadata(t *testing.T) {
	r := NewDiskEncryptionConfigurationListResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*DiskEncryptionConfigurationListResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_disk_encryption_configuration" {
		t.Errorf("expected list type name %q, got %q", "jamfplatform_pro_disk_encryption_configuration", resp.TypeName)
	}
}

func TestDiskEncryptionConfigurationListResource_Schema(t *testing.T) {
	r := NewDiskEncryptionConfigurationListResource()
	var resp list.ListResourceSchemaResponse
	r.(*DiskEncryptionConfigurationListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
