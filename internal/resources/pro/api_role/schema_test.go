// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_role

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestApiRoleResource_Metadata(t *testing.T) {
	r := NewApiRoleResource()
	var resp resource.MetadataResponse
	r.(*ApiRoleResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_api_role" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_api_role", resp.TypeName)
	}
}

func TestApiRoleResource_Schema(t *testing.T) {
	r := NewApiRoleResource()
	var resp resource.SchemaResponse
	r.(*ApiRoleResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "display_name", "privileges", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if id := s.Attributes["id"]; id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only")
	}
	if !s.Attributes["display_name"].IsRequired() {
		t.Errorf("display_name must be required")
	}
	if !s.Attributes["privileges"].IsRequired() {
		t.Errorf("privileges must be required")
	}
}

func TestApiRoleDataSource_Schema(t *testing.T) {
	d := NewApiRoleDataSource()
	var resp datasource.SchemaResponse
	d.(*ApiRoleDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if !resp.Schema.Attributes["id"].IsRequired() {
		t.Errorf("data source id must be required")
	}
	for _, name := range []string{"display_name", "privileges"} {
		if !resp.Schema.Attributes[name].IsComputed() {
			t.Errorf("data source %q must be computed", name)
		}
	}
}

func TestApiRoleListResource_Schema(t *testing.T) {
	r := NewApiRoleListResource()
	var resp list.ListResourceSchemaResponse
	r.(*ApiRoleListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
