// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ldap_server

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestLdapServerResource_Metadata(t *testing.T) {
	r := NewLdapServerResource()
	var resp resource.MetadataResponse
	r.(*LdapServerResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_ldap_server" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_ldap_server", resp.TypeName)
	}
}

func TestLdapServerResource_Schema(t *testing.T) {
	r := NewLdapServerResource()
	var resp resource.SchemaResponse
	r.(*LdapServerResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	for _, name := range []string{"id", "connection_settings", "mappings_for_users", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing top-level attribute %q", name)
		}
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only")
	}

	conn, ok := s.Attributes["connection_settings"].(rschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("connection must be a SingleNestedAttribute")
	}
	if !conn.IsRequired() {
		t.Errorf("connection must be Required")
	}

	// display_name + hostname Required; directory_service Required.
	for _, name := range []string{"display_name", "directory_service", "hostname"} {
		if !conn.Attributes[name].IsRequired() {
			t.Errorf("connection.%s must be Required", name)
		}
	}

	// Server-defaulted connection fields are Optional+Computed.
	for _, name := range []string{"port", "use_ssl", "authentication_type", "connection_timeout", "search_timeout", "referral_response", "use_wildcards"} {
		a := conn.Attributes[name]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("connection.%s must be Optional+Computed, got optional=%v computed=%v", name, a.IsOptional(), a.IsComputed())
		}
	}

	// Server-managed echoes are Computed-only.
	for _, name := range []string{"is_enabled", "migrated_to_id", "certificates_used"} {
		a := conn.Attributes[name]
		if a.IsRequired() || a.IsOptional() || !a.IsComputed() {
			t.Errorf("connection.%s must be Computed-only, got required=%v optional=%v computed=%v", name, a.IsRequired(), a.IsOptional(), a.IsComputed())
		}
	}

	acct, ok := conn.Attributes["account"].(rschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("connection_settings.account must be a SingleNestedAttribute")
	}
	if !acct.IsOptional() {
		t.Errorf("connection_settings.account must be Optional (absent for anonymous binds)")
	}
	pw := acct.Attributes["password"]
	if !pw.IsSensitive() {
		t.Errorf("password must be Sensitive")
	}
	if !pw.(rschema.StringAttribute).WriteOnly {
		t.Errorf("password must be WriteOnly")
	}
	wo := acct.Attributes["password_wo_version"]
	if wo.IsRequired() || !wo.IsOptional() || wo.IsComputed() {
		t.Errorf("password_wo_version must be Optional-only Int64")
	}

	// mappings_for_users + sub-blocks are Optional-ONLY (not Computed): the
	// framework cannot decode an unknown object into a typed model pointer, so
	// Optional+Computed blocks fail at apply. Undeclared blocks are gated out of
	// state by the state builder instead.
	mfu, ok := s.Attributes["mappings_for_users"].(rschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("mappings_for_users must be a SingleNestedAttribute")
	}
	if !mfu.IsOptional() {
		t.Errorf("mappings_for_users must be Optional")
	}
	if mfu.IsComputed() {
		t.Errorf("mappings_for_users must NOT be Computed — framework cannot fit Unknown into a typed pointer model")
	}
	for _, name := range []string{"user_mappings", "user_group_mappings", "user_group_membership_mappings"} {
		sub, ok := mfu.Attributes[name].(rschema.SingleNestedAttribute)
		if !ok {
			t.Errorf("mappings_for_users.%s must be a SingleNestedAttribute", name)
			continue
		}
		if !sub.IsOptional() {
			t.Errorf("mappings_for_users.%s must be Optional", name)
		}
		if sub.IsComputed() {
			t.Errorf("mappings_for_users.%s must NOT be Computed", name)
		}
	}

	// Pin UI-aligned renames so a refactor cannot silently revert to wire names.
	um := mfu.Attributes["user_mappings"].(rschema.SingleNestedAttribute)
	for _, want := range []string{
		"object_class_limitation", "object_classes", "search_base", "search_scope",
		"user_id", "username", "real_name", "email_address", "append_to_email_results",
		"department", "building", "room", "phone", "position", "user_uuid",
	} {
		if _, ok := um.Attributes[want]; !ok {
			t.Errorf("user_mappings missing UI-aligned attribute %q", want)
		}
	}
	mm := mfu.Attributes["user_group_membership_mappings"].(rschema.SingleNestedAttribute)
	for _, want := range []string{
		"membership_location", "member_user_mapping", "group_membership_mapping",
		"use_dn", "use_ldap_compare", "recursive_lookups",
		"membership_calculation_optimization",
		"use_member_field_for_select_queries",
	} {
		if _, ok := mm.Attributes[want]; !ok {
			t.Errorf("user_group_membership_mappings missing UI-aligned attribute %q", want)
		}
	}
	// "Use the 'member' field for select membership queries" is a user checkbox
	// (User Object mode), so it is Optional+Computed — not a computed echo.
	umf := mm.Attributes["use_member_field_for_select_queries"]
	if !umf.IsOptional() || !umf.IsComputed() {
		t.Errorf("use_member_field_for_select_queries must be Optional+Computed")
	}
}

func TestLdapServerResource_ConfigValidators(t *testing.T) {
	r := NewLdapServerResource().(*LdapServerResource)
	if got := r.ConfigValidators(context.Background()); len(got) != 1 {
		t.Fatalf("expected 1 config validator (accountAuthConfigValidator), got %d", len(got))
	}
}

func TestLdapServerDataSource_Schema(t *testing.T) {
	d := NewLdapServerDataSource()
	var resp datasource.SchemaResponse
	d.(*LdapServerDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema
	for _, name := range []string{"id", "name", "connection_settings", "mappings_for_users", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	for _, sel := range []string{"id", "name"} {
		a := s.Attributes[sel]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%q must be Optional+Computed", sel)
		}
	}
}

func TestLdapServerDataSource_ConfigValidators(t *testing.T) {
	d := NewLdapServerDataSource().(*LdapServerDataSource)
	if got := d.ConfigValidators(context.Background()); len(got) != 1 {
		t.Fatalf("expected 1 config validator (ExactlyOneOf id/name), got %d", len(got))
	}
}

func TestLdapServerListResource_Metadata(t *testing.T) {
	r := NewLdapServerListResource()
	var resp resource.MetadataResponse
	r.(*LdapServerListResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_ldap_server" {
		t.Errorf("expected list type name %q, got %q", "jamfplatform_pro_ldap_server", resp.TypeName)
	}
}

func TestLdapServerListResource_Schema(t *testing.T) {
	r := NewLdapServerListResource()
	var resp list.ListResourceSchemaResponse
	r.(*LdapServerListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
