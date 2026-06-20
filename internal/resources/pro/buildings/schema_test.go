// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package buildings

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestBuildingsDataSource_Metadata(t *testing.T) {
	d := NewBuildingsDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*BuildingsDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_buildings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_buildings", resp.TypeName)
	}
}

func TestBuildingsDataSource_Schema(t *testing.T) {
	d := NewBuildingsDataSource()
	var resp datasource.SchemaResponse
	d.(*BuildingsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "timeouts", "filter", "buildings"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	bldgs, ok := s.Attributes["buildings"]
	if !ok {
		t.Fatal("missing buildings attribute")
	}
	nested, ok := bldgs.(datasourceschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("buildings should be ListNestedAttribute, got %T", bldgs)
	}
	for _, name := range []string{
		"id",
		"name",
		"city",
		"country",
		"state_province",
		"street_address_1",
		"street_address_2",
		"zip_postal_code",
	} {
		if _, ok := nested.NestedObject.Attributes[name]; !ok {
			t.Errorf("buildings nested missing attribute %q", name)
		}
	}
}
