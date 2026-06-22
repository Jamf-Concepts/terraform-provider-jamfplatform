// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package automated_device_enrollment

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestAutomatedDeviceEnrollmentResource_Metadata(t *testing.T) {
	r := NewAutomatedDeviceEnrollmentResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*AutomatedDeviceEnrollmentResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_automated_device_enrollment" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_automated_device_enrollment", resp.TypeName)
	}
}

func TestAutomatedDeviceEnrollmentResource_Schema(t *testing.T) {
	r := NewAutomatedDeviceEnrollmentResource()
	var resp resource.SchemaResponse
	r.(*AutomatedDeviceEnrollmentResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{
		"id", "name", "server_token", "server_token_wo_version", "token_file_name",
		"site_id", "supervision_identity_id", "admin_id", "org_name", "org_email",
		"org_phone", "org_address", "server_name", "server_uuid", "token_expiration_date",
		"timeouts",
	} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	name := s.Attributes["name"]
	if !name.IsRequired() {
		t.Errorf("name must be required")
	}

	tok := s.Attributes["server_token"]
	if !tok.IsRequired() {
		t.Errorf("server_token must be required")
	}

	ver := s.Attributes["server_token_wo_version"]
	if !ver.IsRequired() {
		t.Errorf("server_token_wo_version must be required")
	}
}

func TestAutomatedDeviceEnrollmentDataSource_Metadata(t *testing.T) {
	d := NewAutomatedDeviceEnrollmentDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*AutomatedDeviceEnrollmentDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_automated_device_enrollment" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_automated_device_enrollment", resp.TypeName)
	}
}

func TestAutomatedDeviceEnrollmentDataSource_Schema(t *testing.T) {
	d := NewAutomatedDeviceEnrollmentDataSource()
	var resp datasource.SchemaResponse
	d.(*AutomatedDeviceEnrollmentDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{
		"id", "name", "site_id", "supervision_identity_id", "admin_id",
		"org_name", "org_email", "org_phone", "org_address",
		"server_name", "server_uuid", "token_expiration_date", "timeouts",
	} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	// id and name are ExactlyOneOf — both Optional+Computed, neither solely Required.
	idAttr := s.Attributes["id"]
	if idAttr.IsRequired() {
		t.Errorf("data source id must not be required (ExactlyOneOf with name)")
	}
	if !idAttr.IsOptional() || !idAttr.IsComputed() {
		t.Errorf("data source id must be optional+computed, got optional=%v computed=%v", idAttr.IsOptional(), idAttr.IsComputed())
	}

	nameAttr := s.Attributes["name"]
	if nameAttr.IsRequired() {
		t.Errorf("data source name must not be required (ExactlyOneOf with id)")
	}
	if !nameAttr.IsOptional() || !nameAttr.IsComputed() {
		t.Errorf("data source name must be optional+computed, got optional=%v computed=%v", nameAttr.IsOptional(), nameAttr.IsComputed())
	}
}

func TestAutomatedDeviceEnrollmentListResource_Metadata(t *testing.T) {
	r := NewAutomatedDeviceEnrollmentListResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*AutomatedDeviceEnrollmentListResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_automated_device_enrollment" {
		t.Errorf("expected list type name %q, got %q", "jamfplatform_pro_automated_device_enrollment", resp.TypeName)
	}
}

func TestAutomatedDeviceEnrollmentListResource_Schema(t *testing.T) {
	r := NewAutomatedDeviceEnrollmentListResource()
	var resp list.ListResourceSchemaResponse
	r.(*AutomatedDeviceEnrollmentListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
