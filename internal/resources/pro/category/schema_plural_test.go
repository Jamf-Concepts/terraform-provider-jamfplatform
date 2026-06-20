// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package category

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestCategoriesDataSource_Metadata(t *testing.T) {
	d := NewCategoriesDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*CategoriesDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_categories" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_categories", resp.TypeName)
	}
}

func TestCategoriesDataSource_Schema(t *testing.T) {
	d := NewCategoriesDataSource()
	var resp datasource.SchemaResponse
	d.(*CategoriesDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "timeouts", "filter", "categories"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	cats, ok := s.Attributes["categories"]
	if !ok {
		t.Fatal("missing categories attribute")
	}
	nested, ok := cats.(datasourceschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("categories should be ListNestedAttribute, got %T", cats)
	}
	for _, name := range []string{"id", "name", "priority"} {
		if _, ok := nested.NestedObject.Attributes[name]; !ok {
			t.Errorf("categories nested missing attribute %q", name)
		}
	}
}
