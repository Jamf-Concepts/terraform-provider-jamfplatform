// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package jamf_pro_server_url

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestJamfProServerURLDataSource_Metadata(t *testing.T) {
	d := NewJamfProServerURLDataSource()
	var resp datasource.MetadataResponse
	d.(*JamfProServerURLDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_jamf_pro_server_url" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_jamf_pro_server_url", resp.TypeName)
	}
}

func TestJamfProServerURLDataSource_Schema(t *testing.T) {
	d := NewJamfProServerURLDataSource()
	var resp datasource.SchemaResponse
	d.(*JamfProServerURLDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	for _, name := range []string{"id", "url", "timeouts"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if !resp.Schema.Attributes["url"].IsComputed() {
		t.Errorf("url must be computed")
	}
	if resp.Schema.Attributes["url"].IsOptional() || resp.Schema.Attributes["url"].IsRequired() {
		t.Errorf("url must be Computed-only (no selector on a singleton data source)")
	}
}
