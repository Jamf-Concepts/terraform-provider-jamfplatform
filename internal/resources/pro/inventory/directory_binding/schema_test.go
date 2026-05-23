// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package directory_binding

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestDirectoryBindingResource_Metadata(t *testing.T) {
	r := NewDirectoryBindingResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*DirectoryBindingResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_directory_binding" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_directory_binding", resp.TypeName)
	}
}

func TestDirectoryBindingResource_Schema(t *testing.T) {
	r := NewDirectoryBindingResource()
	var resp resource.SchemaResponse
	r.(*DirectoryBindingResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	required := []string{
		"id", "name", "priority", "type", "domain", "username", "password",
		"password_sha256", "computer_ou", "active_directory", "open_directory",
		"admitmac", "centrify", "timeouts",
	}
	for _, name := range required {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	// PowerBroker must NOT appear as a nested block — its identity is
	// conveyed entirely by `type`, and the input builder synthesises the
	// empty SDK struct from `type` alone.
	if _, ok := s.Attributes["powerbroker_identity_services"]; ok {
		t.Errorf("powerbroker_identity_services must not appear in the schema — PowerBroker carries no per-type config")
	}

	if !s.Attributes["name"].IsRequired() {
		t.Errorf("name must be required")
	}
	if !s.Attributes["type"].IsRequired() {
		t.Errorf("type must be required (drives the cross-field block validator)")
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	// priority is Optional+Computed so the server can fill in the default
	// when the user omits it.
	pr := s.Attributes["priority"]
	if !pr.IsOptional() || !pr.IsComputed() {
		t.Errorf("priority must be Optional+Computed, got optional=%v computed=%v", pr.IsOptional(), pr.IsComputed())
	}

	// password must be Sensitive — it is a write-only credential.
	pw := s.Attributes["password"]
	if !pw.IsSensitive() {
		t.Errorf("password must be Sensitive")
	}
	if !pw.IsOptional() {
		t.Errorf("password must be Optional")
	}

	// password_sha256 is the server-computed echo. Computed-only.
	ph := s.Attributes["password_sha256"]
	if ph.IsRequired() || ph.IsOptional() || !ph.IsComputed() {
		t.Errorf("password_sha256 must be Computed-only, got required=%v optional=%v computed=%v", ph.IsRequired(), ph.IsOptional(), ph.IsComputed())
	}

	// Each nested type block is Optional-only and is a
	// SingleNestedAttribute. NOT Computed: the framework cannot fit an
	// Unknown value into a typed `*StructModel` pointer (the model
	// would need to be `types.Object` to handle Unknown), so blocks
	// stay Optional and users supply at least an empty `<type>_block
	// = {}` to take management of the per-type config. Inner FIELDS
	// inside the block are Optional+Computed so the server can
	// populate defaults for fields the user omits. The cross-field
	// validator enforces "match `type` or be absent" at plan time.
	for _, name := range []string{"active_directory", "open_directory", "admitmac", "centrify"} {
		attr, ok := s.Attributes[name].(rschema.SingleNestedAttribute)
		if !ok {
			t.Errorf("%q must be a SingleNestedAttribute", name)
			continue
		}
		if !attr.IsOptional() {
			t.Errorf("%q must be Optional", name)
		}
		if attr.IsComputed() {
			t.Errorf("%q must NOT be Computed — framework cannot fit Unknown into typed pointer model", name)
		}
	}

	// Pin the UI-aligned renames so a future refactor cannot silently
	// revert to wire-form names. The whole point of these renames is
	// documentation (matching the Jamf Pro admin UI labels); only schema
	// tests catch a rename regression because the input/state builders
	// translate at the boundary and acc tests already wrap the UI labels.
	// See STYLE_GUIDE §Attribute names mirror the Jamf Pro admin UI.
	type expectedNested struct {
		block string
		names []string
	}
	expected := []expectedNested{
		{
			block: "active_directory",
			names: []string{
				"forest", "create_mobile_account", "require_confirmation",
				"force_local_home_directory", "use_unc_path", "network_protocol",
				"default_shell", "uid_attribute_mapping", "user_gid_attribute_mapping",
				"gid_attribute_mapping", "multiple_domains", "preferred_domain",
				"admin_groups",
			},
		},
		{
			block: "open_directory",
			names: []string{
				"encrypt_using_ssl", "perform_secure_bind",
				"use_for_authentication", "use_for_contacts",
			},
		},
		{
			block: "admitmac",
			names: []string{
				"require_confirmation", "home_location", "network_protocol",
				"default_shell", "mount_network_home", "place_home_folders",
				"uid_attribute_mapping", "user_gid_attribute_mapping",
				"gid_attribute_mapping", "admin_group", "cached_credentials",
				"add_user_to_local", "users_ou", "groups_ou", "printers_ou",
				"shared_folders_ou",
			},
		},
		{
			block: "centrify",
			names: []string{
				"workstation_mode", "overwrite_existing", "update_pam",
				"zone", "preferred_domain_server",
			},
		},
	}
	for _, e := range expected {
		nested, ok := s.Attributes[e.block].(rschema.SingleNestedAttribute)
		if !ok {
			continue
		}
		for _, want := range e.names {
			if _, ok := nested.Attributes[want]; !ok {
				t.Errorf("nested block %q is missing UI-aligned attribute %q — STYLE_GUIDE forbids reverting to the wire name", e.block, want)
			}
		}
	}
}

func TestDirectoryBindingResource_ConfigValidators(t *testing.T) {
	r := NewDirectoryBindingResource().(*DirectoryBindingResource)
	got := r.ConfigValidators(context.Background())
	if len(got) != 1 {
		t.Fatalf("expected 1 config validator (typeBlockConfigValidator), got %d", len(got))
	}
}

func TestDirectoryBindingDataSource_Metadata(t *testing.T) {
	d := NewDirectoryBindingDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*DirectoryBindingDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_directory_binding" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_directory_binding", resp.TypeName)
	}
}

func TestDirectoryBindingDataSource_Schema(t *testing.T) {
	d := NewDirectoryBindingDataSource()
	var resp datasource.SchemaResponse
	d.(*DirectoryBindingDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "name", "type", "password_sha256", "active_directory", "centrify", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	// Data source must NOT carry a plaintext `password` attribute — it is
	// read-only and the wire never returns the plaintext.
	if _, ok := s.Attributes["password"]; ok {
		t.Errorf("data source must not expose a `password` attribute — the wire returns only `password_sha256`")
	}

	for _, sel := range []string{"id", "name"} {
		a := s.Attributes[sel]
		if !a.IsOptional() {
			t.Errorf("%q must be optional", sel)
		}
		if !a.IsComputed() {
			t.Errorf("%q must be computed (populated from response)", sel)
		}
	}
}

func TestDirectoryBindingDataSource_ConfigValidators_ExactlyOneSelector(t *testing.T) {
	d := NewDirectoryBindingDataSource().(*DirectoryBindingDataSource)
	got := d.ConfigValidators(context.Background())
	if len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestDirectoryBindingListResource_Metadata(t *testing.T) {
	r := NewDirectoryBindingListResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*DirectoryBindingListResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_directory_binding" {
		t.Errorf("expected list type name %q, got %q", "jamfplatform_pro_directory_binding", resp.TypeName)
	}
}

func TestDirectoryBindingListResource_Schema(t *testing.T) {
	r := NewDirectoryBindingListResource()
	var resp list.ListResourceSchemaResponse
	r.(*DirectoryBindingListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
