// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package vpp_invitation

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

const wantTypeName = "jamfplatform_pro_vpp_invitation"

func TestResource_Metadata(t *testing.T) {
	r := NewVPPInvitationResource()
	var resp resource.MetadataResponse
	r.(*VPPInvitationResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != wantTypeName {
		t.Errorf("type name = %q, want %q", resp.TypeName, wantTypeName)
	}
}

func TestResource_Schema(t *testing.T) {
	r := NewVPPInvitationResource()
	var resp resource.SchemaResponse
	r.(*VPPInvitationResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diags: %v", resp.Diagnostics)
	}
	s := resp.Schema
	for _, n := range []string{
		"id", "name", "vpp_account_id", "distribution_method", "auto_register_managed_users",
		"sender_name", "sender_email_address", "subject", "message", "require_login",
		"scope", "invitation_usages", "timeouts",
	} {
		if _, ok := s.Attributes[n]; !ok {
			t.Errorf("missing attribute %q", n)
		}
	}
	for _, req := range []string{"name", "vpp_account_id", "distribution_method"} {
		if !s.Attributes[req].IsRequired() {
			t.Errorf("%q must be required", req)
		}
	}
	if a := s.Attributes["auto_register_managed_users"]; !a.IsOptional() || !a.IsComputed() {
		t.Error("auto_register_managed_users must be optional+computed")
	}
	if a := s.Attributes["invitation_usages"]; !a.IsComputed() || a.IsOptional() || a.IsRequired() {
		t.Error("invitation_usages must be computed-only (read-only)")
	}
	if a := s.Attributes["scope"]; !a.IsOptional() {
		t.Error("scope must be optional")
	}
}

func TestResource_ConfigValidators(t *testing.T) {
	r := NewVPPInvitationResource()
	if got := r.(*VPPInvitationResource).ConfigValidators(context.Background()); len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestDataSource_Schema_And_Validators(t *testing.T) {
	d := NewVPPInvitationDataSource()
	var resp datasource.SchemaResponse
	d.(*VPPInvitationDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ds schema diags: %v", resp.Diagnostics)
	}
	for _, sel := range []string{"id", "name"} {
		a := resp.Schema.Attributes[sel]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%q must be optional+computed selector", sel)
		}
	}
	if got := d.(*VPPInvitationDataSource).ConfigValidators(context.Background()); len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestListResource_Schema(t *testing.T) {
	r := NewVPPInvitationListResource()
	var resp list.ListResourceSchemaResponse
	r.(*VPPInvitationListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("list schema diags: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Error("list schema missing filter")
	}
}
