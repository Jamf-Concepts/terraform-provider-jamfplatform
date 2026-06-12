// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package supervision_identity

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestSupervisionIdentityResource_Metadata(t *testing.T) {
	r := NewSupervisionIdentityResource()
	var resp resource.MetadataResponse
	r.(*SupervisionIdentityResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)

	if resp.TypeName != "jamfplatform_pro_supervision_identity" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_supervision_identity", resp.TypeName)
	}
}

func TestSupervisionIdentityResource_Schema(t *testing.T) {
	r := NewSupervisionIdentityResource()
	var resp resource.SchemaResponse
	r.(*SupervisionIdentityResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "display_name", "password", "certificate_data", "common_name", "expiration_date", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	if id := s.Attributes["id"]; id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	if dn := s.Attributes["display_name"]; !dn.IsRequired() {
		t.Errorf("display_name must be required")
	}

	pw := s.Attributes["password"]
	if !pw.IsRequired() {
		t.Errorf("password must be required")
	}
	if !pw.IsWriteOnly() {
		t.Errorf("password must be write-only")
	}
	if !pw.IsSensitive() {
		t.Errorf("password must be sensitive")
	}

	cert := s.Attributes["certificate_data"]
	if !cert.IsOptional() {
		t.Errorf("certificate_data must be optional")
	}
	if !cert.IsWriteOnly() {
		t.Errorf("certificate_data must be write-only")
	}
	if !cert.IsSensitive() {
		t.Errorf("certificate_data must be sensitive")
	}

	for _, name := range []string{"common_name", "expiration_date"} {
		a := s.Attributes[name]
		if a.IsRequired() || a.IsOptional() || !a.IsComputed() {
			t.Errorf("%s must be computed-only", name)
		}
	}
}

func TestSupervisionIdentityDataSource_Metadata(t *testing.T) {
	d := NewSupervisionIdentityDataSource()
	var resp datasource.MetadataResponse
	d.(*SupervisionIdentityDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)

	if resp.TypeName != "jamfplatform_pro_supervision_identity" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_supervision_identity", resp.TypeName)
	}
}

func TestSupervisionIdentityDataSource_Schema(t *testing.T) {
	d := NewSupervisionIdentityDataSource()
	var resp datasource.SchemaResponse
	d.(*SupervisionIdentityDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "display_name", "common_name", "expiration_date", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	// The data source must never expose secret material.
	for _, name := range []string{"password", "certificate_data"} {
		if _, ok := s.Attributes[name]; ok {
			t.Errorf("data source must not expose %q", name)
		}
	}
}

func TestSupervisionIdentityListResource_Metadata(t *testing.T) {
	r := NewSupervisionIdentityListResource()
	var resp resource.MetadataResponse
	r.(*SupervisionIdentityListResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)

	if resp.TypeName != "jamfplatform_pro_supervision_identity" {
		t.Errorf("expected list type name %q, got %q", "jamfplatform_pro_supervision_identity", resp.TypeName)
	}
}

func TestSupervisionIdentityListResource_Schema(t *testing.T) {
	r := NewSupervisionIdentityListResource()
	var resp list.ListResourceSchemaResponse
	r.(*SupervisionIdentityListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
