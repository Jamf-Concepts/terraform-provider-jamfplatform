// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ibeacon

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestIbeaconResource_Metadata(t *testing.T) {
	r := NewIbeaconResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*IbeaconResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_ibeacon" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_ibeacon", resp.TypeName)
	}
}

func TestIbeaconResource_Schema(t *testing.T) {
	r := NewIbeaconResource()
	var resp resource.SchemaResponse
	r.(*IbeaconResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	required := []string{
		"id", "name", "uuid", "major", "minor",
		"include_any_major_value", "include_any_minor_value", "timeouts",
	}
	for _, name := range required {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	for _, req := range []string{"name", "uuid"} {
		if !s.Attributes[req].IsRequired() {
			t.Errorf("%q must be required", req)
		}
	}

	// Optional-only — nullable knobs the user owns.
	for _, opt := range []string{"major", "minor"} {
		a := s.Attributes[opt]
		if !a.IsOptional() {
			t.Errorf("%q must be optional", opt)
		}
		if a.IsComputed() {
			t.Errorf("%q must NOT be computed (Optional-only)", opt)
		}
	}

	// Each include_any_*_value is Optional + Computed (Default).
	for _, inc := range []string{"include_any_major_value", "include_any_minor_value"} {
		a := s.Attributes[inc]
		if !a.IsOptional() {
			t.Errorf("%q must be optional", inc)
		}
		if !a.IsComputed() {
			t.Errorf("%q must be computed (carries Default)", inc)
		}
	}
}

func TestIbeaconResource_ConfigValidators_HasCrossField(t *testing.T) {
	r := NewIbeaconResource().(*IbeaconResource)
	got := r.ConfigValidators(context.Background())
	if len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestIbeaconDataSource_Metadata(t *testing.T) {
	d := NewIbeaconDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*IbeaconDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_ibeacon" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_ibeacon", resp.TypeName)
	}
}

func TestIbeaconDataSource_Schema(t *testing.T) {
	d := NewIbeaconDataSource()
	var resp datasource.SchemaResponse
	d.(*IbeaconDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "name", "uuid", "major", "minor", "include_any_major_value", "include_any_minor_value", "timeouts"} {
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

func TestIbeaconDataSource_ConfigValidators_ExactlyOneSelector(t *testing.T) {
	d := NewIbeaconDataSource().(*IbeaconDataSource)
	got := d.ConfigValidators(context.Background())
	if len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestIbeaconListResource_Metadata(t *testing.T) {
	r := NewIbeaconListResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*IbeaconListResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_ibeacon" {
		t.Errorf("expected list type name %q, got %q", "jamfplatform_pro_ibeacon", resp.TypeName)
	}
}

func TestIbeaconListResource_Schema(t *testing.T) {
	r := NewIbeaconListResource()
	var resp list.ListResourceSchemaResponse
	r.(*IbeaconListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
