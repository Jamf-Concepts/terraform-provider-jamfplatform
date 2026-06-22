// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package return_to_service

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestResource_Metadata(t *testing.T) {
	r := NewReturnToServiceResource()
	var resp resource.MetadataResponse
	r.(*ReturnToServiceResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_return_to_service" {
		t.Errorf("unexpected type name %q", resp.TypeName)
	}
}

func TestResource_Schema(t *testing.T) {
	r := NewReturnToServiceResource()
	var resp resource.SchemaResponse
	r.(*ReturnToServiceResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	for _, name := range []string{"id", "display_name", "wifi_profile_id", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	if id := s.Attributes["id"]; id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only")
	}
	// Both writable fields are required by the server on every write, so neither
	// is Optional+Computed.
	for _, req := range []string{"display_name", "wifi_profile_id"} {
		a := s.Attributes[req]
		if !a.IsRequired() {
			t.Errorf("%q must be required", req)
		}
		if a.IsComputed() {
			t.Errorf("%q must not be computed", req)
		}
	}
}

func TestDataSource_Schema_AndMetadata(t *testing.T) {
	d := NewReturnToServiceDataSource()
	var meta datasource.MetadataResponse
	d.(*ReturnToServiceDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &meta)
	if meta.TypeName != "jamfplatform_pro_return_to_service" {
		t.Errorf("unexpected DS type name %q", meta.TypeName)
	}

	var resp datasource.SchemaResponse
	d.(*ReturnToServiceDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("DS schema diagnostics: %v", resp.Diagnostics)
	}
	for _, sel := range []string{"id", "display_name"} {
		a := resp.Schema.Attributes[sel]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("DS %q must be optional+computed selector", sel)
		}
	}
	if got := d.(*ReturnToServiceDataSource).ConfigValidators(context.Background()); len(got) != 1 {
		t.Errorf("expected 1 config validator (ExactlyOneOf), got %d", len(got))
	}
}

func TestListResource_Schema_AndMetadata(t *testing.T) {
	r := NewReturnToServiceListResource()
	var meta resource.MetadataResponse
	r.(*ReturnToServiceListResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &meta)
	if meta.TypeName != "jamfplatform_pro_return_to_service" {
		t.Errorf("unexpected list type name %q", meta.TypeName)
	}
	var resp list.ListResourceSchemaResponse
	r.(*ReturnToServiceListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("list schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
