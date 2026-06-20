// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package site

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestSiteResource_Metadata(t *testing.T) {
	r := NewSiteResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*SiteResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_site" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_site", resp.TypeName)
	}
}

func TestSiteResource_Schema(t *testing.T) {
	r := NewSiteResource()
	var resp resource.SchemaResponse
	r.(*SiteResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

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

func TestSiteDataSource_Metadata(t *testing.T) {
	d := NewSiteDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*SiteDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_site" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_site", resp.TypeName)
	}
}

func TestSiteDataSource_Schema(t *testing.T) {
	d := NewSiteDataSource()
	var resp datasource.SchemaResponse
	d.(*SiteDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "name", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	// Both id and name must be Optional+Computed — exactly one is supplied by user,
	// the other is filled from the SDK response.
	for _, sel := range []string{"id", "name"} {
		a := s.Attributes[sel]
		if !a.IsOptional() {
			t.Errorf("%q must be optional", sel)
		}
		if !a.IsComputed() {
			t.Errorf("%q must be computed (populated from response)", sel)
		}
	}
}

func TestSiteDataSource_ConfigValidators_ExactlyOneSelector(t *testing.T) {
	d := NewSiteDataSource().(*SiteDataSource)
	got := d.ConfigValidators(context.Background())
	if len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestSiteListResource_Metadata(t *testing.T) {
	r := NewSiteListResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*SiteListResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_site" {
		t.Errorf("expected list type name %q, got %q", "jamfplatform_pro_site", resp.TypeName)
	}
}

func TestSiteListResource_Schema(t *testing.T) {
	r := NewSiteListResource()
	var resp list.ListResourceSchemaResponse
	r.(*SiteListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
