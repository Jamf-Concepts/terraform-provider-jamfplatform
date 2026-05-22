// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package dock_item

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestDockItemResource_Metadata(t *testing.T) {
	r := NewDockItemResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*DockItemResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_dock_item" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_dock_item", resp.TypeName)
	}
}

func TestDockItemResource_Schema(t *testing.T) {
	r := NewDockItemResource()
	var resp resource.SchemaResponse
	r.(*DockItemResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	required := []string{"id", "name", "type", "path", "contents", "timeouts"}
	for _, name := range required {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	// contents is server-derived: Computed only, NOT Optional. Mirrors the
	// "PLIST File is read-only" UI affordance.
	contents := s.Attributes["contents"]
	if contents.IsOptional() {
		t.Errorf("contents must NOT be optional — it is server-derived and read-only")
	}
	if !contents.IsComputed() {
		t.Errorf("contents must be computed")
	}

	for _, req := range []string{"name", "type", "path"} {
		if !s.Attributes[req].IsRequired() {
			t.Errorf("%q must be required", req)
		}
	}
}

func TestDockItemDataSource_Metadata(t *testing.T) {
	d := NewDockItemDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*DockItemDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_dock_item" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_dock_item", resp.TypeName)
	}
}

func TestDockItemDataSource_Schema(t *testing.T) {
	d := NewDockItemDataSource()
	var resp datasource.SchemaResponse
	d.(*DockItemDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "name", "type", "path", "contents", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

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

func TestDockItemDataSource_ConfigValidators_ExactlyOneSelector(t *testing.T) {
	d := NewDockItemDataSource().(*DockItemDataSource)
	got := d.ConfigValidators(context.Background())
	if len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestDockItemListResource_Metadata(t *testing.T) {
	r := NewDockItemListResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*DockItemListResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_dock_item" {
		t.Errorf("expected list type name %q, got %q", "jamfplatform_pro_dock_item", resp.TypeName)
	}
}

func TestDockItemListResource_Schema(t *testing.T) {
	r := NewDockItemListResource()
	var resp list.ListResourceSchemaResponse
	r.(*DockItemListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
