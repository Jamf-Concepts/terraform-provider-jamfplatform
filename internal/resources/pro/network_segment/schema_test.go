// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package network_segment

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestNetworkSegmentResource_Metadata(t *testing.T) {
	r := NewNetworkSegmentResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*NetworkSegmentResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_network_segment" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_network_segment", resp.TypeName)
	}
}

func TestNetworkSegmentResource_Schema(t *testing.T) {
	r := NewNetworkSegmentResource()
	var resp resource.SchemaResponse
	r.(*NetworkSegmentResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	required := []string{
		"id", "name", "starting_address", "ending_address",
		"building", "department", "override_buildings", "override_departments",
		"distribution_point", "distribution_server", "swu_server", "url",
		"timeouts",
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

	for _, req := range []string{"name", "starting_address", "ending_address"} {
		if !s.Attributes[req].IsRequired() {
			t.Errorf("%q must be required", req)
		}
	}

	// Optional-only — drift via Reconcile*Pointer.
	for _, opt := range []string{"building", "department", "override_buildings", "override_departments"} {
		a := s.Attributes[opt]
		if !a.IsOptional() {
			t.Errorf("%q must be optional", opt)
		}
		if a.IsComputed() {
			t.Errorf("%q must NOT be computed (Optional-only pattern)", opt)
		}
	}

	// Computed-only server-derived fields.
	for _, comp := range []string{"distribution_point", "distribution_server", "swu_server", "url"} {
		a := s.Attributes[comp]
		if a.IsRequired() || a.IsOptional() {
			t.Errorf("%q must be computed-only, got required=%v optional=%v", comp, a.IsRequired(), a.IsOptional())
		}
		if !a.IsComputed() {
			t.Errorf("%q must be computed", comp)
		}
	}
}

func TestNetworkSegmentDataSource_Metadata(t *testing.T) {
	d := NewNetworkSegmentDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*NetworkSegmentDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_network_segment" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_network_segment", resp.TypeName)
	}
}

func TestNetworkSegmentDataSource_Schema(t *testing.T) {
	d := NewNetworkSegmentDataSource()
	var resp datasource.SchemaResponse
	d.(*NetworkSegmentDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "name", "starting_address", "ending_address", "building", "department", "override_buildings", "override_departments", "distribution_point", "distribution_server", "swu_server", "url", "timeouts"} {
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

func TestNetworkSegmentDataSource_ConfigValidators_ExactlyOneSelector(t *testing.T) {
	d := NewNetworkSegmentDataSource().(*NetworkSegmentDataSource)
	got := d.ConfigValidators(context.Background())
	if len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestNetworkSegmentListResource_Metadata(t *testing.T) {
	r := NewNetworkSegmentListResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*NetworkSegmentListResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_network_segment" {
		t.Errorf("expected list type name %q, got %q", "jamfplatform_pro_network_segment", resp.TypeName)
	}
}

func TestNetworkSegmentListResource_Schema(t *testing.T) {
	r := NewNetworkSegmentListResource()
	var resp list.ListResourceSchemaResponse
	r.(*NetworkSegmentListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
