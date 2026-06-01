// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_installers

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestAppInstallersDataSource_Metadata(t *testing.T) {
	d := NewAppInstallersDataSource()
	var resp datasource.MetadataResponse
	d.(*AppInstallersDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_app_installers" {
		t.Errorf("expected plural type name, got %q", resp.TypeName)
	}
}

func TestAppInstallersDataSource_Schema(t *testing.T) {
	d := NewAppInstallersDataSource()
	var resp datasource.SchemaResponse
	d.(*AppInstallersDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	for _, name := range []string{"id", "name_substring", "deployments", "timeouts"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}
