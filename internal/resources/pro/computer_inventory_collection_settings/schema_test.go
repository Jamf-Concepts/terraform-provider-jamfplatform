// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_inventory_collection_settings

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// allAttributeNames is the full top-level schema surface, asserted present on both the
// resource and data source schemas.
var allAttributeNames = []string{
	"id",
	"collect_local_user_accounts",
	"include_home_directory_sizes",
	"include_hidden_accounts",
	"collect_printers",
	"collect_active_services",
	"collect_synced_mobile_device_backup_dates",
	"collect_user_and_location_from_directory_service",
	"collect_package_receipts",
	"collect_available_software_updates",
	"collect_unmanaged_certificates",
	"monitor_beacon_regions",
	"allow_jamf_binary_user_and_location_changes",
	"collect_application_usage_information",
	"use_unix_user_paths",
	"include_software_id",
	"application_search_paths",
	"timeouts",
}

func TestComputerInventoryCollectionSettingsResource_Metadata(t *testing.T) {
	r := NewComputerInventoryCollectionSettingsResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*ComputerInventoryCollectionSettingsResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_computer_inventory_collection_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_computer_inventory_collection_settings", resp.TypeName)
	}
}

func TestComputerInventoryCollectionSettingsResource_Schema(t *testing.T) {
	r := NewComputerInventoryCollectionSettingsResource()
	var resp resource.SchemaResponse
	r.(*ComputerInventoryCollectionSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema
	for _, name := range allAttributeNames {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	// A representative preference toggle must be Optional+Computed.
	pref := s.Attributes["collect_local_user_accounts"]
	if !pref.IsOptional() || !pref.IsComputed() {
		t.Errorf("collect_local_user_accounts must be Optional+Computed, got optional=%v computed=%v", pref.IsOptional(), pref.IsComputed())
	}

	// include_software_id is server-managed: Computed-only.
	sw := s.Attributes["include_software_id"]
	if sw.IsOptional() || !sw.IsComputed() {
		t.Errorf("include_software_id must be Computed-only, got optional=%v computed=%v", sw.IsOptional(), sw.IsComputed())
	}

	// application_search_paths is Optional+Computed.
	paths := s.Attributes["application_search_paths"]
	if !paths.IsOptional() || !paths.IsComputed() {
		t.Errorf("application_search_paths must be Optional+Computed, got optional=%v computed=%v", paths.IsOptional(), paths.IsComputed())
	}
}

func TestComputerInventoryCollectionSettingsResource_IdentitySchema(t *testing.T) {
	r := NewComputerInventoryCollectionSettingsResource()
	var resp resource.IdentitySchemaResponse
	r.(*ComputerInventoryCollectionSettingsResource).IdentitySchema(context.Background(), resource.IdentitySchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("identity schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Errorf("identity schema missing id attribute")
	}
}

func TestComputerInventoryCollectionSettingsDataSource_Metadata(t *testing.T) {
	d := NewComputerInventoryCollectionSettingsDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*ComputerInventoryCollectionSettingsDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_computer_inventory_collection_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_computer_inventory_collection_settings", resp.TypeName)
	}
}

func TestComputerInventoryCollectionSettingsDataSource_Schema(t *testing.T) {
	d := NewComputerInventoryCollectionSettingsDataSource()
	var resp datasource.SchemaResponse
	d.(*ComputerInventoryCollectionSettingsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema
	for _, name := range allAttributeNames {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if !s.Attributes["id"].IsComputed() {
		t.Errorf("data source id must be computed for singleton")
	}
}
