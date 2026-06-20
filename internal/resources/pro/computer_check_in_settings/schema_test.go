// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_check_in_settings

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// boolAttrNames is the set of Optional+Computed boolean toggles on the resource.
var boolAttrNames = []string{
	"create_startup_script",
	"startup_log",
	"startup_policies",
	"startup_ssh",
	"create_login_hook",
	"login_hook_log",
	"login_hook_policies",
	"allow_network_state_change_triggers",
}

func TestComputerCheckInSettingsResource_Metadata(t *testing.T) {
	r := NewComputerCheckInSettingsResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*ComputerCheckInSettingsResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_computer_check_in_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_computer_check_in_settings", resp.TypeName)
	}
}

func TestComputerCheckInSettingsResource_Schema(t *testing.T) {
	r := NewComputerCheckInSettingsResource()
	var resp resource.SchemaResponse
	r.(*ComputerCheckInSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	want := append([]string{"id", "check_in_frequency", "timeouts"}, boolAttrNames...)
	for _, name := range want {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	freq := s.Attributes["check_in_frequency"]
	if !freq.IsRequired() {
		t.Errorf("check_in_frequency must be required")
	}

	for _, name := range boolAttrNames {
		a := s.Attributes[name]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%s must be optional+computed, got optional=%v computed=%v", name, a.IsOptional(), a.IsComputed())
		}
	}
}

func TestComputerCheckInSettingsResource_IdentitySchema(t *testing.T) {
	r := NewComputerCheckInSettingsResource()
	var resp resource.IdentitySchemaResponse
	r.(*ComputerCheckInSettingsResource).IdentitySchema(context.Background(), resource.IdentitySchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("identity schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Errorf("identity schema missing id attribute")
	}
}

func TestComputerCheckInSettingsDataSource_Metadata(t *testing.T) {
	d := NewComputerCheckInSettingsDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*ComputerCheckInSettingsDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_computer_check_in_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_computer_check_in_settings", resp.TypeName)
	}
}

func TestComputerCheckInSettingsDataSource_Schema(t *testing.T) {
	d := NewComputerCheckInSettingsDataSource()
	var resp datasource.SchemaResponse
	d.(*ComputerCheckInSettingsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	want := append([]string{"id", "check_in_frequency", "timeouts"}, boolAttrNames...)
	for _, name := range want {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if !s.Attributes["id"].IsComputed() {
		t.Errorf("data source id must be computed for singleton")
	}
}
