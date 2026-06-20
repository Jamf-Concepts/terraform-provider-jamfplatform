// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package category

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestCategoryResource_Metadata(t *testing.T) {
	r := NewCategoryResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*CategoryResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_category" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_category", resp.TypeName)
	}
}

func TestCategoryResource_Schema(t *testing.T) {
	r := NewCategoryResource()
	var resp resource.SchemaResponse
	r.(*CategoryResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "name", "priority", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	name := s.Attributes["name"]
	if !name.IsRequired() {
		t.Errorf("name must be required")
	}

	prio := s.Attributes["priority"]
	if !prio.IsRequired() {
		t.Errorf("priority must be required")
	}
}

func TestCategoryDataSource_Metadata(t *testing.T) {
	d := NewCategoryDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*CategoryDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_category" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_category", resp.TypeName)
	}
}

func TestCategoryDataSource_Schema(t *testing.T) {
	d := NewCategoryDataSource()
	var resp datasource.SchemaResponse
	d.(*CategoryDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "name", "priority", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if !s.Attributes["id"].IsRequired() {
		t.Errorf("data source id must be required")
	}
}

func TestCategoryListResource_Metadata(t *testing.T) {
	r := NewCategoryListResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*CategoryListResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_category" {
		t.Errorf("expected list type name %q, got %q", "jamfplatform_pro_category", resp.TypeName)
	}
}

func TestCategoryListResource_Schema(t *testing.T) {
	r := NewCategoryListResource()
	var resp list.ListResourceSchemaResponse
	r.(*CategoryListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
