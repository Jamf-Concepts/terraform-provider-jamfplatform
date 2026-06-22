// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package advanced_mobile_device_search

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestResource_Metadata(t *testing.T) {
	r := NewAdvancedMobileDeviceSearchResource()
	var resp resource.MetadataResponse
	r.(*AdvancedMobileDeviceSearchResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_advanced_mobile_device_search" {
		t.Errorf("unexpected type name %q", resp.TypeName)
	}
}

func TestResource_Schema(t *testing.T) {
	r := NewAdvancedMobileDeviceSearchResource()
	var resp resource.SchemaResponse
	r.(*AdvancedMobileDeviceSearchResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	for _, name := range []string{"id", "name", "site_id", "criteria", "display_fields", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	// The Pro type carries only siteId (no site name on the wire), so site_name
	// is not modelled. Matched records / report tabs are not modelled either.
	if _, ok := s.Attributes["site_name"]; ok {
		t.Errorf("site_name must not be exposed (no site name on the Pro wire)")
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
	// criteria/display_fields are Optional+Computed on the full-replace endpoint so
	// omitting them preserves the current value (UseStateForUnknown); set [] to clear.
	for _, o := range []string{"criteria", "display_fields"} {
		a := s.Attributes[o]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%q must be Optional+Computed (omit=preserve), got optional=%v computed=%v", o, a.IsOptional(), a.IsComputed())
		}
	}
}

func TestDataSource_Schema_AndMetadata(t *testing.T) {
	d := NewAdvancedMobileDeviceSearchDataSource()
	var meta datasource.MetadataResponse
	d.(*AdvancedMobileDeviceSearchDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &meta)
	if meta.TypeName != "jamfplatform_pro_advanced_mobile_device_search" {
		t.Errorf("unexpected DS type name %q", meta.TypeName)
	}

	var resp datasource.SchemaResponse
	d.(*AdvancedMobileDeviceSearchDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("DS schema diagnostics: %v", resp.Diagnostics)
	}
	for _, sel := range []string{"id", "name"} {
		a := resp.Schema.Attributes[sel]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("DS %q must be optional+computed selector", sel)
		}
	}
	if got := d.(*AdvancedMobileDeviceSearchDataSource).ConfigValidators(context.Background()); len(got) != 1 {
		t.Errorf("expected 1 config validator (ExactlyOneOf), got %d", len(got))
	}
}

func TestListResource_Schema_AndMetadata(t *testing.T) {
	r := NewAdvancedMobileDeviceSearchListResource()
	var meta resource.MetadataResponse
	r.(*AdvancedMobileDeviceSearchListResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &meta)
	if meta.TypeName != "jamfplatform_pro_advanced_mobile_device_search" {
		t.Errorf("unexpected list type name %q", meta.TypeName)
	}
	var resp list.ListResourceSchemaResponse
	r.(*AdvancedMobileDeviceSearchListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("list schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
