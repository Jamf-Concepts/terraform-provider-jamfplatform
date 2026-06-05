// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package class

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestResource_Metadata(t *testing.T) {
	r := NewClassResource()
	var resp resource.MetadataResponse
	r.(*ClassResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_class" {
		t.Errorf("unexpected type name %q", resp.TypeName)
	}
}

func TestResource_Schema(t *testing.T) {
	r := NewClassResource()
	var resp resource.SchemaResponse
	r.(*ClassResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	for _, name := range []string{
		"id", "name", "description", "site_id", "site_name", "source",
		"students", "teachers", "student_group_ids", "teacher_group_ids",
		"mobile_device_group_ids", "student_ids", "teacher_ids", "timeouts",
	} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	// Server-derived / out-of-scope fields must NOT be modelled.
	for _, absent := range []string{"mobile_devices", "mobile_device_group", "meeting_times", "apple_tvs", "restrictions", "home_screen"} {
		if _, ok := s.Attributes[absent]; ok {
			t.Errorf("attribute %q must not be exposed", absent)
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
	// site_name and source are computed-only server-derived values.
	for _, c := range []string{"site_name", "source"} {
		a := s.Attributes[c]
		if a.IsRequired() || a.IsOptional() || !a.IsComputed() {
			t.Errorf("%q must be computed-only", c)
		}
	}
	// description is optional+computed (server echoes empty default).
	if a := s.Attributes["description"]; !a.IsOptional() || !a.IsComputed() {
		t.Errorf("description must be optional+computed")
	}
	// Authored membership collections are optional-only (omit to clear).
	for _, o := range []string{"students", "teachers", "student_group_ids", "teacher_group_ids", "mobile_device_group_ids"} {
		a := s.Attributes[o]
		if !a.IsOptional() || a.IsComputed() {
			t.Errorf("%q must be optional-only", o)
		}
	}
	// Resolved-ID echoes are computed-only.
	for _, c := range []string{"student_ids", "teacher_ids"} {
		a := s.Attributes[c]
		if a.IsRequired() || a.IsOptional() || !a.IsComputed() {
			t.Errorf("%q must be computed-only", c)
		}
	}
}

func TestDataSource_Schema_AndMetadata(t *testing.T) {
	d := NewClassDataSource()
	var meta datasource.MetadataResponse
	d.(*ClassDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &meta)
	if meta.TypeName != "jamfplatform_pro_class" {
		t.Errorf("unexpected DS type name %q", meta.TypeName)
	}

	var resp datasource.SchemaResponse
	d.(*ClassDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("DS schema diagnostics: %v", resp.Diagnostics)
	}
	for _, sel := range []string{"id", "name"} {
		a := resp.Schema.Attributes[sel]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("DS %q must be optional+computed selector", sel)
		}
	}
	if got := d.(*ClassDataSource).ConfigValidators(context.Background()); len(got) != 1 {
		t.Errorf("expected 1 config validator (ExactlyOneOf), got %d", len(got))
	}
}

func TestListResource_Schema_AndMetadata(t *testing.T) {
	r := NewClassListResource()
	var meta resource.MetadataResponse
	r.(*ClassListResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &meta)
	if meta.TypeName != "jamfplatform_pro_class" {
		t.Errorf("unexpected list type name %q", meta.TypeName)
	}
	var resp list.ListResourceSchemaResponse
	r.(*ClassListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("list schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
