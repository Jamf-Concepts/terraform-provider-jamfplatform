// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package venafi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestPkiVenafiResource_Metadata(t *testing.T) {
	r := NewPkiVenafiResource()
	var resp resource.MetadataResponse
	r.(*PkiVenafiResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_pki_venafi" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_pki_venafi", resp.TypeName)
	}
}

func TestPkiVenafiResource_Schema(t *testing.T) {
	r := NewPkiVenafiResource()
	var resp resource.SchemaResponse
	r.(*PkiVenafiResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{
		"id", "name", "proxy_address", "client_id", "revocation_enabled",
		"refresh_token_wo", "refresh_token_wo_version", "refresh_token_configured",
		"jamf_public_key", "jamf_public_key_rotation", "proxy_trust_store", "timeouts",
	} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	if !s.Attributes["name"].IsRequired() {
		t.Errorf("name must be required")
	}

	// id, jamf_public_key are computed.
	if id := s.Attributes["id"]; !id.IsComputed() || id.IsRequired() || id.IsOptional() {
		t.Errorf("id must be computed-only")
	}
	if jpk := s.Attributes["jamf_public_key"]; !jpk.IsComputed() || jpk.IsRequired() || jpk.IsOptional() {
		t.Errorf("jamf_public_key must be computed-only")
	}
	if rtc := s.Attributes["refresh_token_configured"]; !rtc.IsComputed() || rtc.IsRequired() || rtc.IsOptional() {
		t.Errorf("refresh_token_configured must be computed-only")
	}

	// Optional+Computed (omit=preserve) fields.
	for _, name := range []string{"proxy_address", "client_id", "revocation_enabled", "proxy_trust_store"} {
		a := s.Attributes[name]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%q must be optional+computed", name)
		}
	}

	// refresh_token_wo: optional + sensitive + write-only.
	rt := s.Attributes["refresh_token_wo"]
	if !rt.IsOptional() || !rt.IsSensitive() || !rt.IsWriteOnly() {
		t.Errorf("refresh_token_wo must be optional+sensitive+write-only, got optional=%v sensitive=%v writeonly=%v", rt.IsOptional(), rt.IsSensitive(), rt.IsWriteOnly())
	}

	// rotation triggers: optional, non-computed.
	for _, name := range []string{"refresh_token_wo_version", "jamf_public_key_rotation"} {
		a := s.Attributes[name]
		if !a.IsOptional() || a.IsComputed() {
			t.Errorf("%q must be optional, non-computed", name)
		}
	}
}

func TestPkiVenafiDataSource_Schema(t *testing.T) {
	d := NewPkiVenafiDataSource()
	var resp datasource.SchemaResponse
	d.(*PkiVenafiDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["refresh_token_wo"]; ok {
		t.Errorf("data source must not expose refresh_token_wo")
	}
	if !resp.Schema.Attributes["id"].IsRequired() {
		t.Errorf("data source id must be required")
	}
	for _, name := range []string{"name", "proxy_address", "client_id", "revocation_enabled", "refresh_token_configured", "jamf_public_key", "proxy_trust_store"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("data source missing attribute %q", name)
		}
	}
}
