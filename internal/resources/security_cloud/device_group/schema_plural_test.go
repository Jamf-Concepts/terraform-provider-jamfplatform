// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestDeviceGroupsDataSource_Metadata(t *testing.T) {
	d := NewDeviceGroupsDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*DeviceGroupsDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_security_cloud_device_groups" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_security_cloud_device_groups", resp.TypeName)
	}
}

func TestDeviceGroupsDataSource_Schema(t *testing.T) {
	s := pluralSchema(t)

	for _, name := range []string{"id", "device_groups", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	groups, ok := s.Attributes["device_groups"].(dsschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("device_groups must be a ListNestedAttribute, got %T", s.Attributes["device_groups"])
	}
	for _, name := range []string{"id", "name", "built_in"} {
		attr, present := groups.NestedObject.Attributes[name]
		if !present {
			t.Errorf("device_groups missing nested attribute %q", name)
			continue
		}
		if !attr.IsComputed() {
			t.Errorf("device_groups.%s must be computed", name)
		}
	}
	if !s.Attributes["device_groups"].IsComputed() {
		t.Errorf("device_groups must be computed")
	}
}

// TestDeviceGroupsDataSource_HasBuiltInFlag pins the reason this data source is
// shaped differently from every other plural one in the package. The implicit
// "Default Group" is reported rather than filtered out — otherwise the data source
// would disagree with the admin UI about how many groups exist — and `built_in`
// is what makes its null `id` explicable rather than a bug.
func TestDeviceGroupsDataSource_HasBuiltInFlag(t *testing.T) {
	groups, ok := pluralSchema(t).Attributes["device_groups"].(dsschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("device_groups must be a ListNestedAttribute")
	}
	if _, present := groups.NestedObject.Attributes["built_in"]; !present {
		t.Fatal("device_groups must expose built_in so the built-in group's null id is explicable")
	}
}

// pluralSchema builds the plural data source schema once per test.
func pluralSchema(t *testing.T) dsschema.Schema {
	t.Helper()
	d := NewDeviceGroupsDataSource()
	var resp datasource.SchemaResponse
	d.(*DeviceGroupsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}
