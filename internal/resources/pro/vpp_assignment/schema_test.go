// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package vpp_assignment

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

const wantTypeName = "jamfplatform_pro_vpp_assignment"

func TestResource_Metadata(t *testing.T) {
	r := NewVPPAssignmentResource()
	var resp resource.MetadataResponse
	r.(*VPPAssignmentResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != wantTypeName {
		t.Errorf("type name = %q, want %q", resp.TypeName, wantTypeName)
	}
}

func TestResource_Schema(t *testing.T) {
	r := NewVPPAssignmentResource()
	var resp resource.SchemaResponse
	r.(*VPPAssignmentResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diags: %v", resp.Diagnostics)
	}
	s := resp.Schema
	for _, n := range []string{
		"id", "name", "vpp_admin_account_id", "vpp_admin_account_name",
		"ios_app_adam_ids", "mac_app_adam_ids", "ebook_adam_ids",
		"scope", "timeouts",
	} {
		if _, ok := s.Attributes[n]; !ok {
			t.Errorf("missing attribute %q", n)
		}
	}
	for _, req := range []string{"name", "vpp_admin_account_id"} {
		if !s.Attributes[req].IsRequired() {
			t.Errorf("%q must be required", req)
		}
	}
	// vpp_admin_account_id is mutable (NOT RequiresReplace) — a plain Required string.
	if a := s.Attributes["vpp_admin_account_name"]; !a.IsComputed() || a.IsOptional() || a.IsRequired() {
		t.Error("vpp_admin_account_name must be computed-only")
	}
	for _, c := range []string{"ios_app_adam_ids", "mac_app_adam_ids", "ebook_adam_ids"} {
		a := s.Attributes[c]
		if !a.IsOptional() || a.IsComputed() {
			t.Errorf("%q must be optional-only (opt-out, no Computed child)", c)
		}
	}
	if a := s.Attributes["scope"]; !a.IsOptional() {
		t.Error("scope must be optional")
	}
}

func TestResource_NoConfigValidators(t *testing.T) {
	// The all-flag conflict is attribute-level inside UserScopeAttributes; the
	// resource exposes no resource-level config validators.
	r := NewVPPAssignmentResource()
	if _, ok := r.(resource.ResourceWithConfigValidators); ok {
		t.Error("resource must not implement ResourceWithConfigValidators")
	}
}

func TestDataSource_Schema_And_Validators(t *testing.T) {
	d := NewVPPAssignmentDataSource()
	var resp datasource.SchemaResponse
	d.(*VPPAssignmentDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ds schema diags: %v", resp.Diagnostics)
	}
	for _, sel := range []string{"id", "name"} {
		a := resp.Schema.Attributes[sel]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%q must be optional+computed selector", sel)
		}
	}
	for _, c := range []string{"ios_apps", "mac_apps", "ebooks"} {
		a := resp.Schema.Attributes[c]
		if !a.IsComputed() || a.IsOptional() || a.IsRequired() {
			t.Errorf("DS %q must be computed-only", c)
		}
	}
	if got := d.(*VPPAssignmentDataSource).ConfigValidators(context.Background()); len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestListResource_Schema(t *testing.T) {
	r := NewVPPAssignmentListResource()
	var resp list.ListResourceSchemaResponse
	r.(*VPPAssignmentListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("list schema diags: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Error("list schema missing filter")
	}
}
