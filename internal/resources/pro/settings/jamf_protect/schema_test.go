// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package jamf_protect

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestJamfProtectResource_Metadata(t *testing.T) {
	r := NewJamfProtectResource()
	var resp resource.MetadataResponse
	r.(*JamfProtectResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)

	if want := "jamfplatform_pro_jamf_protect"; resp.TypeName != want {
		t.Errorf("type name = %q, want %q", resp.TypeName, want)
	}
}

func TestJamfProtectResource_Schema(t *testing.T) {
	r := NewJamfProtectResource()
	var resp resource.SchemaResponse
	r.(*JamfProtectResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	want := []string{
		"id", "api_url", "client_id", "password", "password_wo_version",
		"auto_install", "registration_id", "api_client_name",
		"platform_plan_sync", "last_sync_time", "sync_status", "timeouts",
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

	// Registration fields: required.
	for _, name := range []string{"api_url", "client_id", "password_wo_version"} {
		if !s.Attributes[name].IsRequired() {
			t.Errorf("%s must be required", name)
		}
	}

	// password: Required + Sensitive + WriteOnly, never Computed.
	pw := s.Attributes["password"]
	if !pw.IsRequired() || !pw.IsSensitive() || !pw.IsWriteOnly() {
		t.Errorf("password must be Required + Sensitive + WriteOnly, got required=%v sensitive=%v writeOnly=%v",
			pw.IsRequired(), pw.IsSensitive(), pw.IsWriteOnly())
	}
	if pw.IsComputed() {
		t.Errorf("password must not be Computed (WriteOnly secret)")
	}

	// password_wo_version: the rotation companion is a regular attribute that
	// must round-trip through state — never WriteOnly or Sensitive.
	wo := s.Attributes["password_wo_version"]
	if wo.IsWriteOnly() || wo.IsSensitive() || wo.IsComputed() {
		t.Errorf("password_wo_version must be a plain Required attribute, got writeOnly=%v sensitive=%v computed=%v",
			wo.IsWriteOnly(), wo.IsSensitive(), wo.IsComputed())
	}

	// auto_install: Optional+Computed (server default false).
	ai := s.Attributes["auto_install"]
	if !ai.IsOptional() || !ai.IsComputed() || ai.IsRequired() {
		t.Errorf("auto_install must be Optional+Computed, got optional=%v computed=%v required=%v",
			ai.IsOptional(), ai.IsComputed(), ai.IsRequired())
	}

	// Server-derived echoes: computed-only.
	for _, name := range []string{"registration_id", "api_client_name", "platform_plan_sync", "last_sync_time", "sync_status"} {
		a := s.Attributes[name]
		if a.IsRequired() || a.IsOptional() || !a.IsComputed() {
			t.Errorf("%s must be computed-only, got required=%v optional=%v computed=%v", name, a.IsRequired(), a.IsOptional(), a.IsComputed())
		}
	}
}

func TestJamfProtectResource_IdentitySchema(t *testing.T) {
	r := NewJamfProtectResource()
	var resp resource.IdentitySchemaResponse
	r.(*JamfProtectResource).IdentitySchema(context.Background(), resource.IdentitySchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("identity schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Errorf("identity schema missing id attribute")
	}
}

func TestJamfProtectPlansDataSource_Metadata(t *testing.T) {
	d := NewJamfProtectPlansDataSource()
	var resp datasource.MetadataResponse
	d.(*JamfProtectPlansDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)

	if want := "jamfplatform_pro_jamf_protect_plans"; resp.TypeName != want {
		t.Errorf("type name = %q, want %q", resp.TypeName, want)
	}
}

func TestJamfProtectPlansDataSource_Schema(t *testing.T) {
	d := NewJamfProtectPlansDataSource()
	var resp datasource.SchemaResponse
	d.(*JamfProtectPlansDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	for _, name := range []string{"id", "filter", "sort", "plans", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if !s.Attributes["id"].IsComputed() {
		t.Errorf("data source id must be computed")
	}
	if !s.Attributes["plans"].IsComputed() {
		t.Errorf("plans must be computed")
	}
	for _, name := range []string{"filter", "sort"} {
		if !s.Attributes[name].IsOptional() {
			t.Errorf("%s must be optional", name)
		}
	}
	// Registration secrets must never leak onto the catalog data source.
	for _, name := range []string{"password", "password_wo_version", "client_id"} {
		if _, ok := s.Attributes[name]; ok {
			t.Errorf("data source must not expose %q", name)
		}
	}
}
