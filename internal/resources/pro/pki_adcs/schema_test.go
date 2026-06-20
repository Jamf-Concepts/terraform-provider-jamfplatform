// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pki_adcs

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestAdcsResource_Metadata(t *testing.T) {
	r := NewAdcsResource()
	var resp resource.MetadataResponse
	r.(*AdcsResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if want := "jamfplatform_pro_pki_adcs"; resp.TypeName != want {
		t.Errorf("type name = %q, want %q", resp.TypeName, want)
	}
}

func TestAdcsResource_Schema(t *testing.T) {
	r := NewAdcsResource()
	var resp resource.SchemaResponse
	r.(*AdcsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	for _, name := range []string{
		"id", "connector_mode", "display_name", "ca_name", "fqdn", "revocation_enabled",
		"adcs_url", "api_client_id", "server_certificate", "client_certificate",
		"server_certificate_details", "client_certificate_details", "connector_last_check_in", "timeouts",
	} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing top-level attribute %q", name)
		}
	}

	if a := s.Attributes["id"]; a.IsRequired() || a.IsOptional() || !a.IsComputed() {
		t.Error("id must be Computed-only")
	}
	if a := s.Attributes["connector_mode"]; !a.IsRequired() {
		t.Error("connector_mode must be Required")
	}
	// display_name + ca_name + fqdn are Required (Jamf Pro mandates them on create, both modes).
	for _, name := range []string{"display_name", "ca_name", "fqdn"} {
		if a := s.Attributes[name]; !a.IsRequired() {
			t.Errorf("%s must be Required", name)
		}
	}
	// revocation_enabled + the mode-gated adcs_url / api_client_id are Optional+Computed.
	for _, name := range []string{"revocation_enabled", "adcs_url", "api_client_id"} {
		a := s.Attributes[name]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%s must be Optional+Computed", name)
		}
	}
	if a := s.Attributes["connector_last_check_in"]; a.IsRequired() || a.IsOptional() || !a.IsComputed() {
		t.Error("connector_last_check_in must be Computed-only")
	}

	// Input cert blocks: Optional-only typed-pointer, WriteOnly data_wo.
	assertCertInputBlock(t, s, "server_certificate", false)
	assertCertInputBlock(t, s, "client_certificate", true)

	// Details blocks: Computed-only nested objects.
	for _, name := range []string{"server_certificate_details", "client_certificate_details"} {
		block, ok := s.Attributes[name].(schema.SingleNestedAttribute)
		if !ok {
			t.Fatalf("%s must be SingleNestedAttribute", name)
		}
		if block.IsRequired() || block.IsOptional() || !block.IsComputed() {
			t.Errorf("%s must be Computed-only", name)
		}
		for _, child := range []string{"filename", "serial_number", "subject", "issuer", "expiration_date"} {
			if c, ok := block.Attributes[child]; !ok || !c.IsComputed() {
				t.Errorf("%s.%s must be present and Computed", name, child)
			}
		}
	}
}

func assertCertInputBlock(t *testing.T, s schema.Schema, name string, expectPassword bool) {
	t.Helper()
	block, ok := s.Attributes[name].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("%s must be SingleNestedAttribute", name)
	}
	if !block.IsOptional() || block.IsComputed() {
		t.Errorf("%s must be Optional-only typed-pointer (not Computed)", name)
	}
	dataWo := block.Attributes["data_wo"]
	if !dataWo.IsOptional() || !dataWo.IsSensitive() || !dataWo.IsWriteOnly() || dataWo.IsComputed() {
		t.Errorf("%s.data_wo must be Optional+Sensitive+WriteOnly (never Computed)", name)
	}
	fn := block.Attributes["filename"]
	if !fn.IsRequired() {
		t.Errorf("%s.filename must be Required", name)
	}
	if wo := block.Attributes["wo_version"]; !wo.IsOptional() {
		t.Errorf("%s.wo_version must be Optional", name)
	}
	pw, hasPassword := block.Attributes["password_wo"]
	if expectPassword {
		if !hasPassword {
			t.Errorf("%s must expose password_wo", name)
		} else if !pw.IsOptional() || !pw.IsSensitive() || !pw.IsWriteOnly() || pw.IsComputed() {
			t.Errorf("%s.password_wo must be Optional+Sensitive+WriteOnly", name)
		}
	} else if hasPassword {
		t.Errorf("%s must NOT expose password_wo (server certificate is password-less)", name)
	}
}

func TestAdcsDataSource_Schema(t *testing.T) {
	d := NewAdcsDataSource()
	var resp datasource.SchemaResponse
	d.(*AdcsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("data source schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema
	for _, name := range []string{
		"id", "connector_mode", "display_name", "ca_name", "fqdn", "revocation_enabled",
		"adcs_url", "api_client_id", "server_certificate_details", "client_certificate_details", "connector_last_check_in",
	} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("data source missing attribute %q", name)
		}
	}
	if a := s.Attributes["id"]; !a.IsRequired() {
		t.Error("data source id must be Required")
	}
	// Data source must not expose any WriteOnly cert input blocks.
	for _, leaked := range []string{"server_certificate", "client_certificate"} {
		if _, ok := s.Attributes[leaked]; ok {
			t.Errorf("data source must not expose write-only input block %q", leaked)
		}
	}
}
