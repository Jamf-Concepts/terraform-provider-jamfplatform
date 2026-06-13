// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package jamf_connect

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestJamfConnectResource_Metadata(t *testing.T) {
	r := NewJamfConnectResource()
	var resp resource.MetadataResponse
	r.(*JamfConnectResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if want := "jamfplatform_pro_jamf_connect"; resp.TypeName != want {
		t.Errorf("type name = %q, want %q", resp.TypeName, want)
	}
}

func TestJamfConnectResource_Schema(t *testing.T) {
	r := NewJamfConnectResource()
	var resp resource.SchemaResponse
	r.(*JamfConnectResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	want := []string{
		"id", "profile_id", "config_profile_uuid", "auto_deployment_type",
		"version", "profile_name", "scope_description", "site_id", "timeouts",
	}
	for _, name := range want {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	// id: computed-only.
	if id := s.Attributes["id"]; id.IsRequired() || id.IsOptional() || !id.IsComputed() {
		t.Errorf("id must be computed-only")
	}
	// profile_id: the adoption key — required.
	if !s.Attributes["profile_id"].IsRequired() {
		t.Errorf("profile_id must be required")
	}
	// auto_deployment_type: Optional+Computed (NONE default).
	adt := s.Attributes["auto_deployment_type"]
	if !adt.IsOptional() || !adt.IsComputed() || adt.IsRequired() {
		t.Errorf("auto_deployment_type must be Optional+Computed, got optional=%v computed=%v required=%v",
			adt.IsOptional(), adt.IsComputed(), adt.IsRequired())
	}
	// version: Optional only (not Computed — driven by the bidirectional validator).
	v := s.Attributes["version"]
	if !v.IsOptional() || v.IsComputed() || v.IsRequired() {
		t.Errorf("version must be Optional-only, got optional=%v computed=%v required=%v",
			v.IsOptional(), v.IsComputed(), v.IsRequired())
	}
	// Server-derived display fields: computed-only.
	for _, name := range []string{"config_profile_uuid", "profile_name", "scope_description", "site_id"} {
		a := s.Attributes[name]
		if a.IsRequired() || a.IsOptional() || !a.IsComputed() {
			t.Errorf("%s must be computed-only, got required=%v optional=%v computed=%v", name, a.IsRequired(), a.IsOptional(), a.IsComputed())
		}
	}
}

func TestJamfConnectResource_IdentitySchema(t *testing.T) {
	r := NewJamfConnectResource()
	var resp resource.IdentitySchemaResponse
	r.(*JamfConnectResource).IdentitySchema(context.Background(), resource.IdentitySchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("identity schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Errorf("identity schema missing id attribute")
	}
}

func TestJamfConnectResource_ConfigValidatorsWired(t *testing.T) {
	r := NewJamfConnectResource().(*JamfConnectResource)
	vs := r.ConfigValidators(context.Background())
	if len(vs) != 1 {
		t.Fatalf("want 1 config validator, got %d", len(vs))
	}
	if _, ok := vs[0].(versionDeploymentTypeValidator); !ok {
		t.Errorf("expected versionDeploymentTypeValidator, got %T", vs[0])
	}
}

func TestJamfConnectDataSource_Metadata(t *testing.T) {
	d := NewJamfConnectDataSource()
	var resp datasource.MetadataResponse
	d.(*JamfConnectDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if want := "jamfplatform_pro_jamf_connect"; resp.TypeName != want {
		t.Errorf("type name = %q, want %q", resp.TypeName, want)
	}
}

func TestJamfConnectDataSource_Schema(t *testing.T) {
	d := NewJamfConnectDataSource()
	var resp datasource.SchemaResponse
	d.(*JamfConnectDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	for _, name := range []string{"config_profile_uuid", "profile_id", "profile_name"} {
		a := s.Attributes[name]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%s must be Optional+Computed (a lookup key), got optional=%v computed=%v", name, a.IsOptional(), a.IsComputed())
		}
	}
	for _, name := range []string{"auto_deployment_type", "version", "scope_description", "site_id"} {
		if !s.Attributes[name].IsComputed() {
			t.Errorf("%s must be computed", name)
		}
	}
}

func TestJamfConnectDataSource_ConfigValidatorsWired(t *testing.T) {
	d := NewJamfConnectDataSource().(*JamfConnectDataSource)
	vs := d.ConfigValidators(context.Background())
	if len(vs) != 1 {
		t.Fatalf("want 1 config validator, got %d", len(vs))
	}
}
