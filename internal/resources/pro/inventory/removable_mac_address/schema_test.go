// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package removable_mac_address

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestRemovableMacAddressResource_Metadata(t *testing.T) {
	r := NewRemovableMacAddressResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*RemovableMacAddressResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_removable_mac_address" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_removable_mac_address", resp.TypeName)
	}
}

func TestRemovableMacAddressResource_Schema(t *testing.T) {
	r := NewRemovableMacAddressResource()
	var resp resource.SchemaResponse
	r.(*RemovableMacAddressResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "mac_address", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	mac := s.Attributes["mac_address"]
	if !mac.IsRequired() {
		t.Errorf("mac_address must be required")
	}
}

func TestRemovableMacAddressDataSource_Metadata(t *testing.T) {
	d := NewRemovableMacAddressDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*RemovableMacAddressDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_removable_mac_address" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_removable_mac_address", resp.TypeName)
	}
}

func TestRemovableMacAddressDataSource_Schema(t *testing.T) {
	d := NewRemovableMacAddressDataSource()
	var resp datasource.SchemaResponse
	d.(*RemovableMacAddressDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "mac_address", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	// Both id and mac_address must be Optional+Computed — exactly one is supplied by
	// user, the other is filled from the SDK response.
	for _, sel := range []string{"id", "mac_address"} {
		a := s.Attributes[sel]
		if !a.IsOptional() {
			t.Errorf("%q must be optional", sel)
		}
		if !a.IsComputed() {
			t.Errorf("%q must be computed (populated from response)", sel)
		}
	}
}

func TestRemovableMacAddressDataSource_ConfigValidators_ExactlyOneSelector(t *testing.T) {
	d := NewRemovableMacAddressDataSource().(*RemovableMacAddressDataSource)
	got := d.ConfigValidators(context.Background())
	if len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestRemovableMacAddressListResource_Metadata(t *testing.T) {
	r := NewRemovableMacAddressListResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*RemovableMacAddressListResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_removable_mac_address" {
		t.Errorf("expected list type name %q, got %q", "jamfplatform_pro_removable_mac_address", resp.TypeName)
	}
}

func TestRemovableMacAddressListResource_Schema(t *testing.T) {
	r := NewRemovableMacAddressListResource()
	var resp list.ListResourceSchemaResponse
	r.(*RemovableMacAddressListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
