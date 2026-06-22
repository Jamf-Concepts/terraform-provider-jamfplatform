// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_client

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestApiClientsDataSource_Metadata(t *testing.T) {
	d := NewApiClientsDataSource()
	var resp datasource.MetadataResponse
	d.(*ApiClientsDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_api_clients" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_api_clients", resp.TypeName)
	}
}

func TestApiClientsDataSource_Schema_NoSecret(t *testing.T) {
	d := NewApiClientsDataSource()
	var resp datasource.SchemaResponse
	d.(*ApiClientsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	for _, name := range []string{"id", "filter", "api_clients", "timeouts"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	nested := resp.Schema.Attributes["api_clients"]
	if nested == nil {
		t.Fatal("api_clients attribute missing")
	}
}
