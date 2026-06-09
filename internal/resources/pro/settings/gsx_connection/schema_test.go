// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package gsx_connection

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// writeOnlySecretNames is the set of Required + WriteOnly secret attributes.
var writeOnlySecretNames = []string{"token_wo", "keystore_bytes_wo", "keystore_password_wo"}

// optionalComputedNames is the set of Optional+Computed non-secret attributes.
var optionalComputedNames = []string{"enabled", "ship_to_number", "keystore_name"}

// computedOnlyNames is the set of read-only metadata attributes.
var computedOnlyNames = []string{"keystore_error_message", "keystore_expiration_epoch"}

func TestGsxConnectionSettingsResource_Metadata(t *testing.T) {
	r := NewGsxConnectionSettingsResource()
	var resp resource.MetadataResponse
	r.(*GsxConnectionSettingsResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)

	if resp.TypeName != "jamfplatform_pro_gsx_connection_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_gsx_connection_settings", resp.TypeName)
	}
}

func TestGsxConnectionSettingsResource_Schema(t *testing.T) {
	r := NewGsxConnectionSettingsResource()
	var resp resource.SchemaResponse
	r.(*GsxConnectionSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	want := []string{"id", "timeouts", "username", "service_account_number"}
	want = append(want, writeOnlySecretNames...)
	want = append(want, optionalComputedNames...)
	want = append(want, computedOnlyNames...)
	for _, name := range want {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	// id is computed-only.
	if id := s.Attributes["id"]; id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only")
	}

	// username + service_account_number are Required.
	for _, name := range []string{"username", "service_account_number"} {
		if a := s.Attributes[name]; !a.IsRequired() {
			t.Errorf("%s must be required", name)
		}
	}

	// The three secrets are Required + WriteOnly + Sensitive, with no _wo_version companion.
	for _, name := range writeOnlySecretNames {
		a := s.Attributes[name]
		if !a.IsRequired() {
			t.Errorf("%s must be required", name)
		}
		if !a.IsWriteOnly() {
			t.Errorf("%s must be write-only", name)
		}
		if !a.IsSensitive() {
			t.Errorf("%s must be sensitive", name)
		}
		if _, ok := s.Attributes[name+"_version"]; ok {
			t.Errorf("%s must NOT have a _wo_version companion (Design B re-sends every write)", name)
		}
	}

	// Non-secret echoed fields are Optional+Computed.
	for _, name := range optionalComputedNames {
		a := s.Attributes[name]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%s must be optional+computed, got optional=%v computed=%v", name, a.IsOptional(), a.IsComputed())
		}
	}

	// Read-only metadata is computed-only.
	for _, name := range computedOnlyNames {
		a := s.Attributes[name]
		if a.IsOptional() || a.IsRequired() || !a.IsComputed() {
			t.Errorf("%s must be computed-only", name)
		}
	}
}

func TestGsxConnectionSettingsResource_IdentitySchema(t *testing.T) {
	r := NewGsxConnectionSettingsResource()
	var resp resource.IdentitySchemaResponse
	r.(*GsxConnectionSettingsResource).IdentitySchema(context.Background(), resource.IdentitySchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("identity schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Errorf("identity schema missing id attribute")
	}
}

func TestGsxConnectionSettingsDataSource_Metadata(t *testing.T) {
	d := NewGsxConnectionSettingsDataSource()
	var resp datasource.MetadataResponse
	d.(*GsxConnectionSettingsDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_gsx_connection_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_gsx_connection_settings", resp.TypeName)
	}
}

func TestGsxConnectionSettingsDataSource_Schema(t *testing.T) {
	d := NewGsxConnectionSettingsDataSource()
	var resp datasource.SchemaResponse
	d.(*GsxConnectionSettingsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	// The data source must expose only non-secret fields — never the secrets.
	for _, name := range writeOnlySecretNames {
		if _, ok := s.Attributes[name]; ok {
			t.Errorf("data source must NOT expose secret %q", name)
		}
	}
	for _, name := range []string{"id", "enabled", "username", "service_account_number", "ship_to_number", "keystore_name", "keystore_error_message", "keystore_expiration_epoch", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("data source missing attribute %q", name)
		}
	}
	if !s.Attributes["id"].IsComputed() {
		t.Errorf("data source id must be computed for singleton")
	}
}
