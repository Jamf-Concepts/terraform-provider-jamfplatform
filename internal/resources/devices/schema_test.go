// Copyright 2026 Jamf Software LLC.

package devices

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestDevicesDataSource_Metadata(t *testing.T) {
	d := NewDevicesDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*DevicesDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_devices" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_devices", resp.TypeName)
	}
}

func TestDevicesDataSource_Schema(t *testing.T) {
	d := NewDevicesDataSource()
	req := datasource.SchemaRequest{}
	var resp datasource.SchemaResponse
	d.(*DevicesDataSource).Schema(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema

	if _, ok := s.Attributes["filter"]; !ok {
		t.Error("missing filter attribute")
	}

	devices, ok := s.Attributes["devices"]
	if !ok {
		t.Fatal("missing devices attribute")
	}
	nested, ok := devices.(datasourceschema.ListNestedAttribute)
	if !ok {
		t.Fatal("devices should be a ListNestedAttribute")
	}

	expectedAttrs := []string{"id", "serial_number", "name", "model", "model_identifier", "operating_system_version"}
	for _, name := range expectedAttrs {
		if _, ok := nested.NestedObject.Attributes[name]; !ok {
			t.Errorf("devices nested object missing attribute %q", name)
		}
	}
}
