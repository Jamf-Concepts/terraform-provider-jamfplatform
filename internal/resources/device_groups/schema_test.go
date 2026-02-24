// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_groups

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestDeviceGroupsDataSource_Metadata(t *testing.T) {
	d := NewDeviceGroupsDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*DeviceGroupsDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_device_groups" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_device_groups", resp.TypeName)
	}
}

func TestDeviceGroupsDataSource_Schema(t *testing.T) {
	d := NewDeviceGroupsDataSource()
	req := datasource.SchemaRequest{}
	var resp datasource.SchemaResponse
	d.(*DeviceGroupsDataSource).Schema(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema

	if _, ok := s.Attributes["timeouts"]; !ok {
		t.Error("missing timeouts attribute")
	}

	if _, ok := s.Attributes["filter"]; !ok {
		t.Error("missing filter attribute")
	}

	deviceGroups, ok := s.Attributes["device_groups"]
	if !ok {
		t.Fatal("missing device_groups attribute")
	}
	nested, ok := deviceGroups.(datasourceschema.ListNestedAttribute)
	if !ok {
		t.Fatal("device_groups should be a ListNestedAttribute")
	}

	expectedAttrs := []string{"id", "name", "description", "device_type", "group_type", "member_count"}
	for _, name := range expectedAttrs {
		if _, ok := nested.NestedObject.Attributes[name]; !ok {
			t.Errorf("device_groups nested object missing attribute %q", name)
		}
	}
}
