// Copyright 2026 Jamf Software LLC.

package components

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestComponentsDataSource_Metadata(t *testing.T) {
	d := NewComponentsDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*ComponentsDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_blueprints_components" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_blueprints_components", resp.TypeName)
	}
}

func TestComponentsDataSource_Schema(t *testing.T) {
	d := NewComponentsDataSource()
	req := datasource.SchemaRequest{}
	var resp datasource.SchemaResponse
	d.(*ComponentsDataSource).Schema(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema

	components, ok := s.Attributes["components"]
	if !ok {
		t.Fatal("missing components attribute")
	}
	nested, ok := components.(datasourceschema.ListNestedAttribute)
	if !ok {
		t.Fatal("components should be a ListNestedAttribute")
	}

	expectedAttrs := []string{"identifier", "name", "description", "supported_os"}
	for _, name := range expectedAttrs {
		if _, ok := nested.NestedObject.Attributes[name]; !ok {
			t.Errorf("components nested object missing attribute %q", name)
		}
	}
}
