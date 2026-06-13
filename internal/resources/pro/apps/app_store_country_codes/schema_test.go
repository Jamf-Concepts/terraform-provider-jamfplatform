// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_store_country_codes

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestAppStoreCountryCodesDataSource_Metadata(t *testing.T) {
	d := NewAppStoreCountryCodesDataSource()
	var resp datasource.MetadataResponse
	d.(*AppStoreCountryCodesDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_app_store_country_codes" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_app_store_country_codes", resp.TypeName)
	}
}

func TestAppStoreCountryCodesDataSource_Schema(t *testing.T) {
	d := NewAppStoreCountryCodesDataSource()
	var resp datasource.SchemaResponse
	d.(*AppStoreCountryCodesDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "search", "country_codes", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if search := s.Attributes["search"]; !search.IsOptional() {
		t.Errorf("search must be optional")
	}
	if cc := s.Attributes["country_codes"]; !cc.IsComputed() {
		t.Errorf("country_codes must be computed")
	}
}
