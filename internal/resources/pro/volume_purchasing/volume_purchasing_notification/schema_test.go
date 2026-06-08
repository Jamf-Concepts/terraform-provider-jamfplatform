// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package volume_purchasing_notification

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

const wantTypeName = "jamfplatform_pro_volume_purchasing_notification"

func TestResource_Metadata(t *testing.T) {
	r := NewVolumePurchasingNotificationResource()
	var resp resource.MetadataResponse
	r.(*VolumePurchasingNotificationResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != wantTypeName {
		t.Errorf("type name = %q, want %q", resp.TypeName, wantTypeName)
	}
}

func TestResource_Schema(t *testing.T) {
	r := NewVolumePurchasingNotificationResource()
	var resp resource.SchemaResponse
	r.(*VolumePurchasingNotificationResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diags: %v", resp.Diagnostics)
	}
	s := resp.Schema
	for _, n := range []string{
		"id", "name", "enabled", "triggers", "location_ids",
		"internal_recipients", "external_recipients", "site_id", "timeouts",
	} {
		if _, ok := s.Attributes[n]; !ok {
			t.Errorf("missing attribute %q", n)
		}
	}
	if !s.Attributes["name"].IsRequired() {
		t.Error("name must be required")
	}
	if !s.Attributes["id"].IsComputed() || s.Attributes["id"].IsOptional() {
		t.Error("id must be computed-only")
	}
	// All four collections + enabled + site_id are Optional+Computed (full-replace,
	// empty set clears, server echoes defaults).
	for _, oc := range []string{"enabled", "triggers", "location_ids", "internal_recipients", "external_recipients", "site_id"} {
		a := s.Attributes[oc]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%q must be optional+computed", oc)
		}
	}
}

func TestResource_NoConfigValidators(t *testing.T) {
	r := NewVolumePurchasingNotificationResource()
	if _, ok := r.(resource.ResourceWithConfigValidators); ok {
		t.Error("resource must not implement ResourceWithConfigValidators (enum OneOf is attribute-level)")
	}
}

func TestDataSource_Schema_And_Validators(t *testing.T) {
	d := NewVolumePurchasingNotificationDataSource()
	var resp datasource.SchemaResponse
	d.(*VolumePurchasingNotificationDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ds schema diags: %v", resp.Diagnostics)
	}
	for _, sel := range []string{"id", "name"} {
		a := resp.Schema.Attributes[sel]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%q must be optional+computed selector", sel)
		}
	}
	for _, c := range []string{"enabled", "triggers", "location_ids", "internal_recipients", "external_recipients", "site_id"} {
		a := resp.Schema.Attributes[c]
		if !a.IsComputed() || a.IsOptional() || a.IsRequired() {
			t.Errorf("DS %q must be computed-only", c)
		}
	}
	if got := d.(*VolumePurchasingNotificationDataSource).ConfigValidators(context.Background()); len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestListResource_Schema(t *testing.T) {
	r := NewVolumePurchasingNotificationListResource()
	var resp list.ListResourceSchemaResponse
	r.(*VolumePurchasingNotificationListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("list schema diags: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Error("list schema missing filter")
	}
}
