// Copyright 2026 Jamf Software LLC.

package device

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestDeviceDataSource_Metadata(t *testing.T) {
	d := NewDeviceDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*DeviceDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_device" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_device", resp.TypeName)
	}
}

func TestDeviceDataSource_Schema(t *testing.T) {
	d := NewDeviceDataSource()
	req := datasource.SchemaRequest{}
	var resp datasource.SchemaResponse
	d.(*DeviceDataSource).Schema(context.Background(), req, &resp)

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

	computedAttrs := []string{"serial_number", "name", "model", "model_identifier", "operating_system_version"}
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
