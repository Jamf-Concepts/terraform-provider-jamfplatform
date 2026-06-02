// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package advanced_computer_search

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestResource_Metadata(t *testing.T) {
	r := NewAdvancedComputerSearchResource()
	var resp resource.MetadataResponse
	r.(*AdvancedComputerSearchResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_advanced_computer_search" {
		t.Errorf("unexpected type name %q", resp.TypeName)
	}
}

func TestResource_Schema(t *testing.T) {
	r := NewAdvancedComputerSearchResource()
	var resp resource.SchemaResponse
	r.(*AdvancedComputerSearchResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	for _, name := range []string{
		"id", "name", "site_id", "site_name", "view_as",
		"sort_1", "sort_2", "sort_3", "criteria", "display_fields", "timeouts",
	} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	if id := s.Attributes["id"]; id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only")
	}
	if !s.Attributes["name"].IsRequired() {
		t.Errorf("name must be required")
	}
	// Optional+Computed with defaults.
	for _, oc := range []string{"site_id", "view_as"} {
		a := s.Attributes[oc]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%q must be optional+computed", oc)
		}
	}
	// site_name is computed-only (server-derived from mutable site_id; must NOT
	// be Optional, and must NOT carry UseStateForUnknown — see the derived-name
	// latent-bug note in the codebase).
	if sn := s.Attributes["site_name"]; sn.IsRequired() || sn.IsOptional() || !sn.IsComputed() {
		t.Errorf("site_name must be computed-only")
	}
	// Managed collections + sorts are optional-only (omit to clear).
	for _, o := range []string{"criteria", "display_fields", "sort_1", "sort_2", "sort_3"} {
		a := s.Attributes[o]
		if !a.IsOptional() || a.IsComputed() {
			t.Errorf("%q must be optional-only", o)
		}
	}
}

func TestDataSource_Schema_AndMetadata(t *testing.T) {
	d := NewAdvancedComputerSearchDataSource()
	var meta datasource.MetadataResponse
	d.(*AdvancedComputerSearchDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &meta)
	if meta.TypeName != "jamfplatform_pro_advanced_computer_search" {
		t.Errorf("unexpected DS type name %q", meta.TypeName)
	}

	var resp datasource.SchemaResponse
	d.(*AdvancedComputerSearchDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("DS schema diagnostics: %v", resp.Diagnostics)
	}
	for _, sel := range []string{"id", "name"} {
		a := resp.Schema.Attributes[sel]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("DS %q must be optional+computed selector", sel)
		}
	}
	if got := d.(*AdvancedComputerSearchDataSource).ConfigValidators(context.Background()); len(got) != 1 {
		t.Errorf("expected 1 config validator (ExactlyOneOf), got %d", len(got))
	}
}

func TestListResource_Schema_AndMetadata(t *testing.T) {
	r := NewAdvancedComputerSearchListResource()
	var meta resource.MetadataResponse
	r.(*AdvancedComputerSearchListResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &meta)
	if meta.TypeName != "jamfplatform_pro_advanced_computer_search" {
		t.Errorf("unexpected list type name %q", meta.TypeName)
	}
	var resp list.ListResourceSchemaResponse
	r.(*AdvancedComputerSearchListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("list schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
