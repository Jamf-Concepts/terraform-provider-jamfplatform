// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_role_privileges

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestApiRolePrivilegesDataSource_Metadata(t *testing.T) {
	d := NewApiRolePrivilegesDataSource()
	var resp datasource.MetadataResponse
	d.(*ApiRolePrivilegesDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_api_role_privileges" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_api_role_privileges", resp.TypeName)
	}
}

func TestApiRolePrivilegesDataSource_Schema(t *testing.T) {
	d := NewApiRolePrivilegesDataSource()
	var resp datasource.SchemaResponse
	d.(*ApiRolePrivilegesDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	for _, name := range []string{"id", "search", "privileges", "timeouts"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if !resp.Schema.Attributes["search"].IsOptional() {
		t.Errorf("search must be optional")
	}
	if !resp.Schema.Attributes["privileges"].IsComputed() {
		t.Errorf("privileges must be computed")
	}
}
