// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package advanced_user_search

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestResource_Metadata(t *testing.T) {
	r := NewAdvancedUserSearchResource()
	var resp resource.MetadataResponse
	r.(*AdvancedUserSearchResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_advanced_user_search" {
		t.Errorf("unexpected type name %q", resp.TypeName)
	}
}

func TestResource_Schema(t *testing.T) {
	r := NewAdvancedUserSearchResource()
	var resp resource.SchemaResponse
	r.(*AdvancedUserSearchResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	for _, name := range []string{"id", "name", "site_id", "site_name", "criteria", "display_fields", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	// User searches have no view_as / sort columns.
	for _, absent := range []string{"view_as", "sort_1", "sort_2", "sort_3"} {
		if _, ok := s.Attributes[absent]; ok {
			t.Errorf("user search must NOT expose %q", absent)
		}
	}

	if id := s.Attributes["id"]; id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only")
	}
	if !s.Attributes["name"].IsRequired() {
		t.Errorf("name must be required")
	}
	if a := s.Attributes["site_id"]; !a.IsOptional() || !a.IsComputed() {
		t.Errorf("site_id must be optional+computed")
	}
	if sn := s.Attributes["site_name"]; sn.IsRequired() || sn.IsOptional() || !sn.IsComputed() {
		t.Errorf("site_name must be computed-only")
	}
	for _, o := range []string{"criteria", "display_fields"} {
		a := s.Attributes[o]
		if !a.IsOptional() || a.IsComputed() {
			t.Errorf("%q must be optional-only", o)
		}
	}
}

func TestDataSource_Schema_AndMetadata(t *testing.T) {
	d := NewAdvancedUserSearchDataSource()
	var meta datasource.MetadataResponse
	d.(*AdvancedUserSearchDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &meta)
	if meta.TypeName != "jamfplatform_pro_advanced_user_search" {
		t.Errorf("unexpected DS type name %q", meta.TypeName)
	}
	var resp datasource.SchemaResponse
	d.(*AdvancedUserSearchDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("DS schema diagnostics: %v", resp.Diagnostics)
	}
	for _, sel := range []string{"id", "name"} {
		a := resp.Schema.Attributes[sel]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("DS %q must be optional+computed selector", sel)
		}
	}
	if got := d.(*AdvancedUserSearchDataSource).ConfigValidators(context.Background()); len(got) != 1 {
		t.Errorf("expected 1 config validator, got %d", len(got))
	}
}

func TestListResource_Schema_AndMetadata(t *testing.T) {
	r := NewAdvancedUserSearchListResource()
	var meta resource.MetadataResponse
	r.(*AdvancedUserSearchListResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &meta)
	if meta.TypeName != "jamfplatform_pro_advanced_user_search" {
		t.Errorf("unexpected list type name %q", meta.TypeName)
	}
	var resp list.ListResourceSchemaResponse
	r.(*AdvancedUserSearchListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("list schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}

func TestValidOperators_ExcludesDateWindowOperators(t *testing.T) {
	for _, dropped := range []string{"in less than x days", "in more than x days"} {
		for _, op := range ValidOperators {
			if op == dropped {
				t.Errorf("user-search operator vocabulary must exclude %q", dropped)
			}
		}
	}
}
