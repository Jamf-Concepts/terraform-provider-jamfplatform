// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package component

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestComponentDataSource_Metadata(t *testing.T) {
	d := NewComponentDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*ComponentDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_blueprints_component" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_blueprints_component", resp.TypeName)
	}
}

func TestComponentDataSource_Schema(t *testing.T) {
	d := NewComponentDataSource()
	req := datasource.SchemaRequest{}
	var resp datasource.SchemaResponse
	d.(*ComponentDataSource).Schema(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema

	id, ok := s.Attributes["id"]
	if !ok {
		t.Fatal("missing id attribute")
	}
	if !id.IsRequired() {
		t.Error("id should be required")
	}

	computedAttrs := []string{"identifier", "name", "description", "supported_os"}
	for _, name := range computedAttrs {
		attr, ok := s.Attributes[name]
		if !ok {
			t.Errorf("missing computed attribute %q", name)
			continue
		}
		if !attr.IsComputed() {
			t.Errorf("attribute %q should be computed", name)
		}
	}
}
