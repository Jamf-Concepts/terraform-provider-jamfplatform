// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package blueprints

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestBlueprintsDataSource_Metadata(t *testing.T) {
	d := NewBlueprintsDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*BlueprintsDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_blueprints" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_blueprints", resp.TypeName)
	}
}

func TestBlueprintsDataSource_Schema(t *testing.T) {
	d := NewBlueprintsDataSource()
	req := datasource.SchemaRequest{}
	var resp datasource.SchemaResponse
	d.(*BlueprintsDataSource).Schema(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema

	search, ok := s.Attributes["search"]
	if !ok {
		t.Fatal("missing search attribute")
	}
	if !search.IsOptional() {
		t.Error("search should be optional")
	}

	blueprints, ok := s.Attributes["blueprints"]
	if !ok {
		t.Fatal("missing blueprints attribute")
	}
	nested, ok := blueprints.(datasourceschema.ListNestedAttribute)
	if !ok {
		t.Fatal("blueprints should be a ListNestedAttribute")
	}

	expectedAttrs := []string{"id", "name", "description", "created", "updated", "deployment_state"}
	for _, name := range expectedAttrs {
		if _, ok := nested.NestedObject.Attributes[name]; !ok {
			t.Errorf("blueprints nested object missing attribute %q", name)
		}
	}
}
