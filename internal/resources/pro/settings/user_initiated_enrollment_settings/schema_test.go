// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package user_initiated_enrollment_settings

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// TestResource_Metadata checks the resource type name.
func TestResource_Metadata(t *testing.T) {
	r := NewUserInitiatedEnrollmentSettingsResource()
	var resp resource.MetadataResponse
	r.(*UserInitiatedEnrollmentSettingsResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_user_initiated_enrollment_settings" {
		t.Errorf("unexpected type name %q", resp.TypeName)
	}
}

// TestResource_Schema_TopLevelAttributes asserts the toggle attributes are
// Optional+Computed and the deprecated type is Computed-only.
func TestResource_Schema_TopLevelAttributes(t *testing.T) {
	r := NewUserInitiatedEnrollmentSettingsResource()
	var resp resource.SchemaResponse
	r.(*UserInitiatedEnrollmentSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	optComputed := []string{
		"skip_certificate_installation", "restrict_reenrollment", "signing_mdm_profile_enabled",
		"enable_computer_enrollment", "create_management_account", "management_username",
		"hide_management_account", "allow_ssh_only_management_account", "ensure_ssh_running",
		"launch_self_service", "sign_quickadd_package", "account_driven_device_enrollment_macos",
		"profile_driven_enrollment_via_url_institutional", "profile_driven_enrollment_via_url_personal",
		"account_driven_user_enrollment", "account_driven_user_enrollment_visionos",
		"merge_managed_apple_account_usernames", "account_driven_device_enrollment_ios",
		"account_driven_device_enrollment_visionos",
	}
	for _, name := range optComputed {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Errorf("missing attribute %q", name)
			continue
		}
		if !attr.IsOptional() || !attr.IsComputed() {
			t.Errorf("attribute %q must be Optional+Computed", name)
		}
	}

	// personal_device_enrollment_type must be Computed-only (deprecated).
	pdet, ok := resp.Schema.Attributes["personal_device_enrollment_type"]
	if !ok || !pdet.IsComputed() || pdet.IsOptional() {
		t.Errorf("personal_device_enrollment_type must be Computed-only, not Optional")
	}

	if id, ok := resp.Schema.Attributes["id"]; !ok || !id.IsComputed() {
		t.Errorf("id must be Computed")
	}
}

// TestResource_Schema_CertBlocks asserts the cert sub-blocks are
// SingleNestedAttribute with WriteOnly keystore inputs and Computed details.
func TestResource_Schema_CertBlocks(t *testing.T) {
	r := NewUserInitiatedEnrollmentSettingsResource()
	var resp resource.SchemaResponse
	r.(*UserInitiatedEnrollmentSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	for _, block := range []string{"mdm_signing_certificate", "developer_certificate"} {
		cert, ok := resp.Schema.Attributes[block].(rschema.SingleNestedAttribute)
		if !ok {
			t.Fatalf("%s must be SingleNestedAttribute", block)
		}
		kf, ok := cert.Attributes["keystore_file"].(rschema.StringAttribute)
		if !ok || !kf.IsWriteOnly() || !kf.IsSensitive() {
			t.Errorf("%s.keystore_file must be WriteOnly+Sensitive", block)
		}
		kp, ok := cert.Attributes["keystore_password"].(rschema.StringAttribute)
		if !ok || !kp.IsWriteOnly() || !kp.IsSensitive() {
			t.Errorf("%s.keystore_password must be WriteOnly+Sensitive", block)
		}
		for _, computed := range []string{"subject", "serial_number"} {
			a, ok := cert.Attributes[computed]
			if !ok || !a.IsComputed() {
				t.Errorf("%s.%s must be Computed", block, computed)
			}
		}
	}
}

// TestResource_Schema_AccessGroup asserts the access_group set shape.
func TestResource_Schema_AccessGroup(t *testing.T) {
	r := NewUserInitiatedEnrollmentSettingsResource()
	var resp resource.SchemaResponse
	r.(*UserInitiatedEnrollmentSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	ag, ok := resp.Schema.Attributes["access_group"].(rschema.SetNestedAttribute)
	if !ok {
		t.Fatalf("access_group must be SetNestedAttribute")
	}
	id, ok := ag.NestedObject.Attributes["id"]
	if !ok || !id.IsComputed() {
		t.Errorf("access_group.id must be Computed")
	}
	// name + ldap_server_id are the user inputs; directory_service_group_id is
	// resolved by the provider (Computed, not user-supplied).
	for _, req := range []string{"ldap_server_id", "name"} {
		a, ok := ag.NestedObject.Attributes[req]
		if !ok || !a.IsRequired() {
			t.Errorf("access_group.%s must be Required", req)
		}
	}
	dsg, ok := ag.NestedObject.Attributes["directory_service_group_id"]
	if !ok || !dsg.IsComputed() || dsg.IsRequired() {
		t.Errorf("access_group.directory_service_group_id must be Computed (resolved from name), not Required")
	}
}

// TestDataSource_Metadata checks the DS type name.
func TestDataSource_Metadata(t *testing.T) {
	d := NewUserInitiatedEnrollmentSettingsDataSource()
	var resp datasource.MetadataResponse
	d.(*UserInitiatedEnrollmentSettingsDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_user_initiated_enrollment_settings" {
		t.Errorf("unexpected DS type name %q", resp.TypeName)
	}
}
