// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package account

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAccountResource_Metadata(t *testing.T) {
	r := NewAccountResource()
	var resp resource.MetadataResponse
	r.(*AccountResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_account" {
		t.Errorf("got %q", resp.TypeName)
	}
}

func TestAccountResource_Schema(t *testing.T) {
	r := NewAccountResource()
	var resp resource.SchemaResponse
	r.(*AccountResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema
	for _, name := range []string{"id", "username", "full_name", "email_address", "access_level", "privilege_set", "access_status", "account_type", "ldap_server_id", "site_id", "force_password_change", "password", "password_wo_version", "privileges", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if id := s.Attributes["id"]; id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only")
	}
	if !s.Attributes["username"].IsRequired() {
		t.Errorf("username must be required")
	}
	if !s.Attributes["access_level"].IsRequired() {
		t.Errorf("access_level must be required")
	}
	if !s.Attributes["privilege_set"].IsRequired() {
		t.Errorf("privilege_set must be required")
	}
	pw := s.Attributes["password"]
	if !pw.IsOptional() || !pw.IsWriteOnly() {
		t.Errorf("password must be optional + write-only, got optional=%v writeonly=%v", pw.IsOptional(), pw.IsWriteOnly())
	}
}

func TestAccountResource_EnumTranslation(t *testing.T) {
	// Round-trip the wire/UI maps for the two non-obvious values.
	cases := map[string]string{"Group Access": "GroupBasedAccess", "Full Access": "FullAccess"}
	for ui, wire := range cases {
		if got := translate(accessLevelToWire, ui); got != wire {
			t.Errorf("accessLevelToWire[%q] = %q, want %q", ui, got, wire)
		}
		if got := translate(accessLevelFromWire, wire); got != ui {
			t.Errorf("accessLevelFromWire[%q] = %q, want %q", wire, got, ui)
		}
	}
	if got := translate(privilegeSetToWire, "Enrollment Only"); got != "ENROLLMENT" {
		t.Errorf("Enrollment Only -> %q, want ENROLLMENT", got)
	}
	if got := translate(privilegeSetFromWire, "ENROLLMENT"); got != "Enrollment Only" {
		t.Errorf("ENROLLMENT -> %q, want Enrollment Only", got)
	}
	// Unknown value passes through unchanged.
	if got := translate(accessLevelToWire, "Mystery"); got != "Mystery" {
		t.Errorf("unknown value should pass through, got %q", got)
	}
}

func TestCustPrivApplicable(t *testing.T) {
	if !custPrivApplicable(types.StringValue("Custom"), types.StringValue("Full Access")) {
		t.Error("Custom + Full Access should be applicable")
	}
	if custPrivApplicable(types.StringValue("Custom"), types.StringValue("Group Access")) {
		t.Error("Custom + Group Access should NOT be applicable")
	}
	if custPrivApplicable(types.StringValue("Auditor"), types.StringValue("Full Access")) {
		t.Error("Auditor should NOT be applicable")
	}
}

func TestAccountDataSource_Metadata(t *testing.T) {
	d := NewAccountDataSource()
	var resp datasource.MetadataResponse
	d.(*AccountDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_account" {
		t.Errorf("got %q", resp.TypeName)
	}
}

func TestAccountDataSource_Schema(t *testing.T) {
	d := NewAccountDataSource()
	var resp datasource.SchemaResponse
	d.(*AccountDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
}
