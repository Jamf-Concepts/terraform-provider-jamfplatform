// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package user_group

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestUserGroupResource_Metadata(t *testing.T) {
	r := NewUserGroupResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*UserGroupResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_user_group" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_user_group", resp.TypeName)
	}
}

func TestUserGroupResource_Schema(t *testing.T) {
	r := NewUserGroupResource()
	var resp resource.SchemaResponse
	r.(*UserGroupResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	required := []string{
		"id", "name", "group_type", "notify_on_membership_change",
		"site_id", "site_name", "members", "member_count",
		"criteria", "timeouts",
	}
	for _, name := range required {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	// id is computed-only.
	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only")
	}

	// name and group_type are required.
	for _, n := range []string{"name", "group_type"} {
		if !s.Attributes[n].IsRequired() {
			t.Errorf("%q must be required", n)
		}
	}

	// Optional+Computed with defaults.
	for _, oc := range []string{"notify_on_membership_change", "site_id"} {
		a := s.Attributes[oc]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%q must be optional+computed", oc)
		}
	}

	// Computed-only server-derived.
	for _, c := range []string{"site_name", "member_count"} {
		a := s.Attributes[c]
		if a.IsRequired() || a.IsOptional() {
			t.Errorf("%q must be computed-only", c)
		}
		if !a.IsComputed() {
			t.Errorf("%q must be computed", c)
		}
	}

	// Optional-only managed lists.
	for _, o := range []string{"members", "criteria"} {
		a := s.Attributes[o]
		if !a.IsOptional() {
			t.Errorf("%q must be optional", o)
		}
	}
}

func TestUserGroupDataSource_Metadata(t *testing.T) {
	d := NewUserGroupDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*UserGroupDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_user_group" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_user_group", resp.TypeName)
	}
}

func TestUserGroupDataSource_Schema(t *testing.T) {
	d := NewUserGroupDataSource()
	var resp datasource.SchemaResponse
	d.(*UserGroupDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "name", "group_type", "notify_on_membership_change", "site_id", "site_name", "member_count", "criteria", "users", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	// id and name are both Optional+Computed (selectors).
	for _, sel := range []string{"id", "name"} {
		a := s.Attributes[sel]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%q must be optional+computed", sel)
		}
	}
}

func TestUserGroupDataSource_ConfigValidators_ExactlyOneSelector(t *testing.T) {
	d := NewUserGroupDataSource().(*UserGroupDataSource)
	got := d.ConfigValidators(context.Background())
	if len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestUserGroupListResource_Metadata(t *testing.T) {
	r := NewUserGroupListResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*UserGroupListResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_user_group" {
		t.Errorf("expected list type name %q, got %q", "jamfplatform_pro_user_group", resp.TypeName)
	}
}

func TestUserGroupListResource_Schema(t *testing.T) {
	r := NewUserGroupListResource()
	var resp list.ListResourceSchemaResponse
	r.(*UserGroupListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
