// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_request_form_field

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

const wantTypeName = "jamfplatform_pro_app_request_form_field"

func TestAppRequestFormFieldResource_Metadata(t *testing.T) {
	r := NewAppRequestFormFieldResource()
	var resp resource.MetadataResponse
	r.(*AppRequestFormFieldResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != wantTypeName {
		t.Errorf("expected type name %q, got %q", wantTypeName, resp.TypeName)
	}
}

func TestAppRequestFormFieldResource_Schema(t *testing.T) {
	r := NewAppRequestFormFieldResource()
	var resp resource.SchemaResponse
	r.(*AppRequestFormFieldResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "title", "description", "priority", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	if id := s.Attributes["id"]; id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}
	if title := s.Attributes["title"]; !title.IsRequired() {
		t.Errorf("title must be required")
	}
	if priority := s.Attributes["priority"]; !priority.IsRequired() {
		t.Errorf("priority must be required")
	}
	if desc := s.Attributes["description"]; !desc.IsOptional() || desc.IsRequired() {
		t.Errorf("description must be optional-only, got optional=%v required=%v", desc.IsOptional(), desc.IsRequired())
	}
}

func TestAppRequestFormFieldDataSource_Metadata(t *testing.T) {
	d := NewAppRequestFormFieldDataSource()
	var resp datasource.MetadataResponse
	d.(*AppRequestFormFieldDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != wantTypeName {
		t.Errorf("expected type name %q, got %q", wantTypeName, resp.TypeName)
	}
}

func TestAppRequestFormFieldDataSource_Schema(t *testing.T) {
	d := NewAppRequestFormFieldDataSource()
	var resp datasource.SchemaResponse
	d.(*AppRequestFormFieldDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, sel := range []string{"id", "title"} {
		a := s.Attributes[sel]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%q must be optional+computed", sel)
		}
	}
	for _, c := range []string{"description", "priority"} {
		if a := s.Attributes[c]; !a.IsComputed() || a.IsOptional() || a.IsRequired() {
			t.Errorf("%q must be computed-only", c)
		}
	}
}

func TestAppRequestFormFieldDataSource_ConfigValidators(t *testing.T) {
	d := NewAppRequestFormFieldDataSource().(*AppRequestFormFieldDataSource)
	if got := d.ConfigValidators(context.Background()); len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestAppRequestFormFieldListResource_Metadata(t *testing.T) {
	r := NewAppRequestFormFieldListResource()
	var resp resource.MetadataResponse
	r.(*AppRequestFormFieldListResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != wantTypeName {
		t.Errorf("expected list type name %q, got %q", wantTypeName, resp.TypeName)
	}
}

func TestAppRequestFormFieldListResource_Schema(t *testing.T) {
	r := NewAppRequestFormFieldListResource()
	var resp list.ListResourceSchemaResponse
	r.(*AppRequestFormFieldListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
