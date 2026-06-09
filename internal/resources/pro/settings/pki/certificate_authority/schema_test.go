// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package certificate_authority

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestCertificateAuthorityDataSource_Metadata(t *testing.T) {
	d := NewCertificateAuthorityDataSource()
	var resp datasource.MetadataResponse
	d.(*CertificateAuthorityDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)

	if resp.TypeName != "jamfplatform_pro_pki_certificate_authority" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_pki_certificate_authority", resp.TypeName)
	}
}

func TestCertificateAuthorityDataSource_Schema(t *testing.T) {
	d := NewCertificateAuthorityDataSource()
	var resp datasource.SchemaResponse
	d.(*CertificateAuthorityDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	computed := []string{
		"subject_x500_principal", "issuer_x500_principal", "serial_number", "version",
		"not_after", "not_before", "key_usage", "key_usage_extended",
		"sha1_fingerprint", "sha256_fingerprint",
		"signature_algorithm", "signature_algorithm_oid", "signature_value", "pem",
	}
	for _, name := range computed {
		a, ok := s.Attributes[name]
		if !ok {
			t.Errorf("missing attribute %q", name)
			continue
		}
		if a.IsRequired() || a.IsOptional() {
			t.Errorf("attribute %q must be computed-only", name)
		}
		if !a.IsComputed() {
			t.Errorf("attribute %q must be computed", name)
		}
	}

	// id is the only Optional input (also Computed for the active-CA case).
	id, ok := s.Attributes["id"]
	if !ok {
		t.Fatalf("missing id attribute")
	}
	if !id.IsOptional() || !id.IsComputed() {
		t.Errorf("id must be Optional+Computed")
	}

	if _, ok := s.Attributes["timeouts"]; !ok {
		t.Errorf("missing timeouts attribute")
	}
}
