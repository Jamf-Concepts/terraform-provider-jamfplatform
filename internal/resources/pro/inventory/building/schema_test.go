// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package building

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// optionalAddressAttributes enumerates every Optional address field on the building
// resource schema. Plain Optional (no Computed) so clear-on-omit via Pro's lossy PUT
// works — Optional+Computed+UseStateForUnknown would silently freeze the old value
// when a user removes the field from config.
var optionalAddressAttributes = []string{
	"city",
	"country",
	"state_province",
	"street_address_1",
	"street_address_2",
	"zip_postal_code",
}

func TestBuildingResource_Metadata(t *testing.T) {
	r := NewBuildingResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*BuildingResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_building" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_building", resp.TypeName)
	}
}

func TestBuildingResource_Schema(t *testing.T) {
	r := NewBuildingResource()
	var resp resource.SchemaResponse
	r.(*BuildingResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	expected := []string{"id", "name", "timeouts"}
	expected = append(expected, optionalAddressAttributes...)
	for _, name := range expected {
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

	for _, attrName := range optionalAddressAttributes {
		a := s.Attributes[attrName]
		if !a.IsOptional() {
			t.Errorf("%q must be optional", attrName)
		}
		if a.IsComputed() {
			t.Errorf("%q must NOT be computed — Optional+Computed defeats Pro clear-on-omit", attrName)
		}
	}
}

func TestBuildingDataSource_Metadata(t *testing.T) {
	d := NewBuildingDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*BuildingDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_building" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_building", resp.TypeName)
	}
}

func TestBuildingDataSource_Schema(t *testing.T) {
	d := NewBuildingDataSource()
	var resp datasource.SchemaResponse
	d.(*BuildingDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	expected := []string{"id", "name", "timeouts"}
	expected = append(expected, optionalAddressAttributes...)
	for _, name := range expected {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if !s.Attributes["id"].IsRequired() {
		t.Errorf("data source id must be required")
	}
}

func TestBuildingListResource_Metadata(t *testing.T) {
	r := NewBuildingListResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*BuildingListResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_building" {
		t.Errorf("expected list type name %q, got %q", "jamfplatform_pro_building", resp.TypeName)
	}
}

func TestBuildingListResource_Schema(t *testing.T) {
	r := NewBuildingListResource()
	var resp list.ListResourceSchemaResponse
	r.(*BuildingListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
