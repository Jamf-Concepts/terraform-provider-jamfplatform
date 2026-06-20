// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_client

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestApiClientResource_Metadata(t *testing.T) {
	r := NewApiClientResource()
	var resp resource.MetadataResponse
	r.(*ApiClientResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_api_client" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_api_client", resp.TypeName)
	}
}

func TestApiClientResource_Schema(t *testing.T) {
	r := NewApiClientResource()
	var resp resource.SchemaResponse
	r.(*ApiClientResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "display_name", "api_roles", "enabled", "access_token_lifetime_seconds", "client_id", "app_type", "credential_rotation", "client_secret", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if !s.Attributes["display_name"].IsRequired() {
		t.Errorf("display_name must be required")
	}
	if !s.Attributes["api_roles"].IsRequired() {
		t.Errorf("api_roles must be required")
	}

	secret := s.Attributes["client_secret"]
	if !secret.IsComputed() || !secret.IsSensitive() {
		t.Errorf("client_secret must be computed + sensitive, got computed=%v sensitive=%v", secret.IsComputed(), secret.IsSensitive())
	}
	if secret.IsRequired() || secret.IsOptional() {
		t.Errorf("client_secret must be computed-only (not user-settable)")
	}

	if appType := s.Attributes["app_type"]; !appType.IsComputed() || appType.IsRequired() || appType.IsOptional() {
		t.Errorf("app_type must be computed-only")
	}

	rotation := s.Attributes["credential_rotation"]
	if !rotation.IsOptional() || rotation.IsComputed() {
		t.Errorf("credential_rotation must be optional, non-computed")
	}

	for _, name := range []string{"enabled", "access_token_lifetime_seconds"} {
		a := s.Attributes[name]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%q must be optional+computed", name)
		}
	}
}

func TestApiClientDataSource_Schema_NoSecret(t *testing.T) {
	d := NewApiClientDataSource()
	var resp datasource.SchemaResponse
	d.(*ApiClientDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["client_secret"]; ok {
		t.Errorf("data source must not expose client_secret")
	}
	if !resp.Schema.Attributes["id"].IsRequired() {
		t.Errorf("data source id must be required")
	}
}

func TestApiClientListResource_Schema(t *testing.T) {
	r := NewApiClientListResource()
	var resp list.ListResourceSchemaResponse
	r.(*ApiClientListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
