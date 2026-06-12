// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package account_group

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestAccountGroupResource_Metadata(t *testing.T) {
	r := NewAccountGroupResource()
	var resp resource.MetadataResponse
	r.(*AccountGroupResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_account_group" {
		t.Errorf("got %q", resp.TypeName)
	}
}

func TestAccountGroupResource_Schema(t *testing.T) {
	r := NewAccountGroupResource()
	var resp resource.SchemaResponse
	r.(*AccountGroupResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema
	for _, name := range []string{"id", "display_name", "access_level", "privilege_set", "site_id", "site_name", "ldap_server_id", "ldap_server_name", "members", "privileges", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if id := s.Attributes["id"]; id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only")
	}
	if !s.Attributes["display_name"].IsRequired() {
		t.Errorf("display_name must be required")
	}
	if !s.Attributes["access_level"].IsRequired() {
		t.Errorf("access_level must be required")
	}
	if !s.Attributes["privilege_set"].IsRequired() {
		t.Errorf("privilege_set must be required")
	}
	if sn := s.Attributes["site_name"]; sn.IsRequired() || sn.IsOptional() || !sn.IsComputed() {
		t.Errorf("site_name must be computed-only (derived)")
	}
	if !s.Attributes["members"].IsOptional() {
		t.Errorf("members must be optional")
	}
	if !s.Attributes["privileges"].IsOptional() {
		t.Errorf("privileges must be optional")
	}
}

func TestAccountGroupDataSource_Metadata(t *testing.T) {
	d := NewAccountGroupDataSource()
	var resp datasource.MetadataResponse
	d.(*AccountGroupDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_account_group" {
		t.Errorf("got %q", resp.TypeName)
	}
}

func TestAccountGroupDataSource_Schema(t *testing.T) {
	d := NewAccountGroupDataSource()
	var resp datasource.SchemaResponse
	d.(*AccountGroupDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["privileges"]; !ok {
		t.Errorf("DS missing privileges")
	}
}
