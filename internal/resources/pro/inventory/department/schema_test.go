// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package department

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestDepartmentResource_Metadata(t *testing.T) {
	r := NewDepartmentResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*DepartmentResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_department" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_department", resp.TypeName)
	}
}

func TestDepartmentResource_Schema(t *testing.T) {
	r := NewDepartmentResource()
	var resp resource.SchemaResponse
	r.(*DepartmentResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "name", "timeouts"} {
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
}

func TestDepartmentDataSource_Metadata(t *testing.T) {
	d := NewDepartmentDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*DepartmentDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_department" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_department", resp.TypeName)
	}
}

func TestDepartmentDataSource_Schema(t *testing.T) {
	d := NewDepartmentDataSource()
	var resp datasource.SchemaResponse
	d.(*DepartmentDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "name", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if !s.Attributes["id"].IsRequired() {
		t.Errorf("data source id must be required")
	}
}

func TestDepartmentListResource_Metadata(t *testing.T) {
	r := NewDepartmentListResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*DepartmentListResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_department" {
		t.Errorf("expected list type name %q, got %q", "jamfplatform_pro_department", resp.TypeName)
	}
}

func TestDepartmentListResource_Schema(t *testing.T) {
	r := NewDepartmentListResource()
	var resp list.ListResourceSchemaResponse
	r.(*DepartmentListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
