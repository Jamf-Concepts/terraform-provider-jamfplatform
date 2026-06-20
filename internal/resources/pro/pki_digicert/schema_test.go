// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pki_digicert

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestDigicertResource_Metadata(t *testing.T) {
	r := NewDigicertResource()
	var resp resource.MetadataResponse
	r.(*DigicertResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_pki_digicert" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_pki_digicert", resp.TypeName)
	}
}

func TestDigicertResource_Schema(t *testing.T) {
	r := NewDigicertResource()
	var resp resource.SchemaResponse
	r.(*DigicertResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "display_name", "host_name", "revocation_enabled", "client_certificate", "client_certificate_details", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	// display_name + host_name are Required (Jamf Pro mandates them on create).
	for _, name := range []string{"display_name", "host_name"} {
		if a := s.Attributes[name]; !a.IsRequired() {
			t.Errorf("%q must be required, got required=%v", name, a.IsRequired())
		}
	}
	// revocation_enabled is Optional+Computed (server-defaulted toggle).
	if a := s.Attributes["revocation_enabled"]; !a.IsOptional() || !a.IsComputed() {
		t.Errorf("revocation_enabled must be optional+computed, got optional=%v computed=%v", a.IsOptional(), a.IsComputed())
	}

	if id := s.Attributes["id"]; !id.IsComputed() || id.IsRequired() || id.IsOptional() {
		t.Errorf("id must be computed-only")
	}

	// Input block: Optional-only, not Computed (avoids the Optional+Computed nested trap).
	cc, ok := s.Attributes["client_certificate"].(rschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("client_certificate must be a SingleNestedAttribute")
	}
	if !cc.IsOptional() || cc.IsComputed() {
		t.Errorf("client_certificate must be optional-only, got optional=%v computed=%v", cc.IsOptional(), cc.IsComputed())
	}
	for _, name := range []string{"data_wo", "password_wo"} {
		a, ok := cc.Attributes[name]
		if !ok {
			t.Fatalf("client_certificate missing %q", name)
		}
		sa, ok := a.(rschema.StringAttribute)
		if !ok {
			t.Fatalf("%q must be a StringAttribute", name)
		}
		if !sa.WriteOnly || !sa.Sensitive || !sa.Optional {
			t.Errorf("%q must be optional+sensitive+writeonly, got writeonly=%v sensitive=%v optional=%v", name, sa.WriteOnly, sa.Sensitive, sa.Optional)
		}
	}
	if _, ok := cc.Attributes["wo_version"]; !ok {
		t.Errorf("client_certificate missing wo_version")
	}
	if fn := cc.Attributes["filename"]; !fn.IsRequired() {
		t.Errorf("client_certificate.filename must be required")
	}

	// Details block: Computed-only.
	det, ok := s.Attributes["client_certificate_details"].(rschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("client_certificate_details must be a SingleNestedAttribute")
	}
	if !det.IsComputed() || det.IsRequired() || det.IsOptional() {
		t.Errorf("client_certificate_details must be computed-only")
	}
	for _, name := range []string{"filename", "serial_number", "subject", "issuer", "expiration_date"} {
		a, ok := det.Attributes[name]
		if !ok {
			t.Fatalf("client_certificate_details missing %q", name)
		}
		if !a.IsComputed() {
			t.Errorf("client_certificate_details.%q must be computed", name)
		}
	}
}

func TestDigicertDataSource_Schema_NoSecret(t *testing.T) {
	d := NewDigicertDataSource()
	var resp datasource.SchemaResponse
	d.(*DigicertDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["client_certificate"]; ok {
		t.Errorf("data source must not expose the client_certificate input block")
	}
	if _, ok := resp.Schema.Attributes["client_certificate_details"].(dsschema.SingleNestedAttribute); !ok {
		t.Errorf("data source must expose client_certificate_details")
	}
	if !resp.Schema.Attributes["id"].IsRequired() {
		t.Errorf("data source id must be required")
	}
}
