// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_prestage_enrollment

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestMobileDevicePrestageEnrollmentResource_Metadata(t *testing.T) {
	r := NewMobileDevicePrestageEnrollmentResource()
	var resp resource.MetadataResponse
	r.(*MobileDevicePrestageEnrollmentResource).Metadata(context.Background(),
		resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if want := "jamfplatform_pro_mobile_device_prestage_enrollment"; resp.TypeName != want {
		t.Errorf("expected type name %q, got %q", want, resp.TypeName)
	}
}

func TestMobileDevicePrestageEnrollmentResource_Schema(t *testing.T) {
	r := NewMobileDevicePrestageEnrollmentResource()
	var resp resource.SchemaResponse
	r.(*MobileDevicePrestageEnrollmentResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	present := []string{
		"id", "display_name", "device_enrollment_program_instance_id",
		"default_prestage", "multi_user", "use_storage_quota_size",
		"storage_quota_size_megabytes", "temporary_session_only",
		"skip_setup_items", "names", "location_information",
		"purchasing_information", "scope_serial_numbers", "timeouts",
	}
	for _, name := range present {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	// account_settings is a computer-only concept; mobile prestages carry no
	// local-admin password / account-settings block (spike §6).
	if _, ok := s.Attributes["account_settings"]; ok {
		t.Errorf("account_settings must NOT exist on the mobile prestage schema")
	}

	if id := s.Attributes["id"]; id.IsRequired() || id.IsOptional() || !id.IsComputed() {
		t.Errorf("id must be computed-only")
	}
	if uuid := s.Attributes["profile_uuid"]; uuid.IsRequired() || uuid.IsOptional() || !uuid.IsComputed() {
		t.Errorf("profile_uuid must be computed-only")
	}
	if site := s.Attributes["site_id"]; site.IsRequired() || site.IsOptional() || !site.IsComputed() {
		t.Errorf("site_id must be computed-only")
	}
	// storage_quota_size_megabytes is read-only: Jamf Pro recalculates it on
	// every update, so it must be Computed-only (not settable) to stay idempotent.
	if sq := s.Attributes["storage_quota_size_megabytes"]; sq.IsRequired() || sq.IsOptional() || !sq.IsComputed() {
		t.Errorf("storage_quota_size_megabytes must be computed-only")
	}
	if dn := s.Attributes["display_name"]; !dn.IsRequired() {
		t.Errorf("display_name must be required")
	}

	diags := s.ValidateImplementation(context.Background())
	if diags.HasError() {
		t.Fatalf("ValidateImplementation diagnostics: %v", diags)
	}
}

func TestMobileDevicePrestageEnrollmentDataSource_Metadata(t *testing.T) {
	d := NewMobileDevicePrestageEnrollmentDataSource()
	var resp datasource.MetadataResponse
	d.(*MobileDevicePrestageEnrollmentDataSource).Metadata(context.Background(),
		datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if want := "jamfplatform_pro_mobile_device_prestage_enrollment"; resp.TypeName != want {
		t.Errorf("expected type name %q, got %q", want, resp.TypeName)
	}
}

func TestMobileDevicePrestageEnrollmentDataSource_Schema(t *testing.T) {
	d := NewMobileDevicePrestageEnrollmentDataSource()
	var resp datasource.SchemaResponse
	d.(*MobileDevicePrestageEnrollmentDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("data source schema diagnostics: %v", resp.Diagnostics)
	}
	diags := resp.Schema.ValidateImplementation(context.Background())
	if diags.HasError() {
		t.Fatalf("data source ValidateImplementation diagnostics: %v", diags)
	}
}

func TestMobileDevicePrestageEnrollmentListResource_Metadata(t *testing.T) {
	r := NewMobileDevicePrestageEnrollmentListResource()
	var resp resource.MetadataResponse
	r.(*MobileDevicePrestageEnrollmentListResource).Metadata(context.Background(),
		resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if want := "jamfplatform_pro_mobile_device_prestage_enrollment"; resp.TypeName != want {
		t.Errorf("expected list type name %q, got %q", want, resp.TypeName)
	}
}

func TestMobileDevicePrestageEnrollmentListResource_Schema(t *testing.T) {
	r := NewMobileDevicePrestageEnrollmentListResource()
	var resp list.ListResourceSchemaResponse
	r.(*MobileDevicePrestageEnrollmentListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("list resource schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list resource schema must expose a `filter` attribute")
	}
}
