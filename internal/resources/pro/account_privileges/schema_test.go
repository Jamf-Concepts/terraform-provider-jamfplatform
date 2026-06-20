// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package account_privileges

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestAccountPrivilegesDataSource_Metadata(t *testing.T) {
	d := NewAccountPrivilegesDataSource()
	var resp datasource.MetadataResponse
	d.(*AccountPrivilegesDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_account_privileges" {
		t.Errorf("got %q", resp.TypeName)
	}
}

func TestAccountPrivilegesDataSource_Schema(t *testing.T) {
	d := NewAccountPrivilegesDataSource()
	var resp datasource.SchemaResponse
	d.(*AccountPrivilegesDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	for _, name := range []string{"jamf_pro_server_objects", "jamf_pro_server_settings", "jamf_pro_server_actions", "casper_admin", "casper_remote", "casper_imaging", "recon", "all"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}
