// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_extension_attribute

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestComputerEAResource_Metadata(t *testing.T) {
	r := NewComputerExtensionAttributeResource()
	var resp resource.MetadataResponse
	r.(*ComputerExtensionAttributeResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_computer_extension_attribute" {
		t.Errorf("unexpected type name %q", resp.TypeName)
	}
}

func TestComputerEAResource_Schema(t *testing.T) {
	r := NewComputerExtensionAttributeResource()
	var resp resource.SchemaResponse
	r.(*ComputerExtensionAttributeResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema
	for _, name := range []string{"id", "name", "description", "data_type", "input_type", "inventory_display", "enabled", "script", "popup_menu_choices", "directory_service_attribute", "allow_multiple_values", "manage_existing_data", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if id := s.Attributes["id"]; id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only")
	}
	for _, req := range []string{"name", "data_type", "input_type", "inventory_display"} {
		if !s.Attributes[req].IsRequired() {
			t.Errorf("%s must be required", req)
		}
	}
}

func TestComputerEAResource_ConfigValidatorsPresent(t *testing.T) {
	r := NewComputerExtensionAttributeResource().(*ComputerExtensionAttributeResource)
	if len(r.ConfigValidators(context.Background())) == 0 {
		t.Errorf("expected at least one config validator")
	}
}

func TestComputerEADataSource_Schema(t *testing.T) {
	d := NewComputerExtensionAttributeDataSource()
	var resp datasource.SchemaResponse
	d.(*ComputerExtensionAttributeDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	for _, name := range []string{"id", "name", "data_type", "input_type", "inventory_display", "script", "popup_menu_choices"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("data source missing attribute %q", name)
		}
	}
}

func TestComputerEAListResource_Schema(t *testing.T) {
	r := NewComputerExtensionAttributeListResource()
	var resp list.ListResourceSchemaResponse
	r.(*ComputerExtensionAttributeListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
