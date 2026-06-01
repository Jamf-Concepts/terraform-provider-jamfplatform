// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_roles

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestApiRolesDataSource_Metadata(t *testing.T) {
	d := NewApiRolesDataSource()
	var resp datasource.MetadataResponse
	d.(*ApiRolesDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_api_roles" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_api_roles", resp.TypeName)
	}
}

func TestApiRolesDataSource_Schema(t *testing.T) {
	d := NewApiRolesDataSource()
	var resp datasource.SchemaResponse
	d.(*ApiRolesDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	for _, name := range []string{"id", "filter", "api_roles", "timeouts"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}
