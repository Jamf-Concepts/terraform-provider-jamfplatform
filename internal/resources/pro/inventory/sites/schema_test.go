// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sites

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestSitesDataSource_Metadata(t *testing.T) {
	d := NewSitesDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*SitesDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_sites" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_sites", resp.TypeName)
	}
}

func TestSitesDataSource_Schema(t *testing.T) {
	d := NewSitesDataSource()
	var resp datasource.SchemaResponse
	d.(*SitesDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "timeouts", "filter", "sites"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	sitesAttr, ok := s.Attributes["sites"]
	if !ok {
		t.Fatal("missing sites attribute")
	}
	nested, ok := sitesAttr.(datasourceschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("sites should be ListNestedAttribute, got %T", sitesAttr)
	}
	for _, name := range []string{"id", "name"} {
		if _, ok := nested.NestedObject.Attributes[name]; !ok {
			t.Errorf("sites nested missing attribute %q", name)
		}
	}
}
