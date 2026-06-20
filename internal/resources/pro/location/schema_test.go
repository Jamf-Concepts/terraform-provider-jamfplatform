// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package location

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestVolumePurchasingLocationResource_Metadata(t *testing.T) {
	r := NewVolumePurchasingLocationResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*VolumePurchasingLocationResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_volume_purchasing_location" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_volume_purchasing_location", resp.TypeName)
	}
}

func TestVolumePurchasingLocationResource_Schema(t *testing.T) {
	r := NewVolumePurchasingLocationResource()
	var resp resource.SchemaResponse
	r.(*VolumePurchasingLocationResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{
		"id", "name", "service_token", "service_token_wo_version",
		"automatically_populate_purchased_content", "send_notification_when_no_longer_assigned",
		"auto_register_managed_users", "site_id", "site_name", "apple_id",
		"organization_name", "location_name", "country_code", "email",
		"token_expiration", "total_purchased_licenses", "total_used_licenses",
		"last_sync_time", "client_context_mismatch", "content", "timeouts",
	} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	name := s.Attributes["name"]
	if !name.IsRequired() {
		t.Errorf("name must be required")
	}

	tok := s.Attributes["service_token"]
	if !tok.IsRequired() {
		t.Errorf("service_token must be required")
	}

	ver := s.Attributes["service_token_wo_version"]
	if !ver.IsRequired() {
		t.Errorf("service_token_wo_version must be required")
	}

	content := s.Attributes["content"]
	if !content.IsComputed() {
		t.Errorf("content must be computed")
	}
}

func TestVolumePurchasingLocationDataSource_Metadata(t *testing.T) {
	d := NewVolumePurchasingLocationDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*VolumePurchasingLocationDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_volume_purchasing_location" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_volume_purchasing_location", resp.TypeName)
	}
}

func TestVolumePurchasingLocationDataSource_Schema(t *testing.T) {
	d := NewVolumePurchasingLocationDataSource()
	var resp datasource.SchemaResponse
	d.(*VolumePurchasingLocationDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{
		"id", "name", "automatically_populate_purchased_content",
		"send_notification_when_no_longer_assigned", "auto_register_managed_users",
		"site_id", "site_name", "apple_id", "organization_name", "location_name",
		"country_code", "email", "token_expiration", "total_purchased_licenses",
		"total_used_licenses", "last_sync_time", "client_context_mismatch",
		"content", "timeouts",
	} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	// id and name are ExactlyOneOf — both Optional+Computed, neither solely Required.
	idAttr := s.Attributes["id"]
	if idAttr.IsRequired() {
		t.Errorf("data source id must not be required (ExactlyOneOf with name)")
	}
	if !idAttr.IsOptional() || !idAttr.IsComputed() {
		t.Errorf("data source id must be optional+computed, got optional=%v computed=%v", idAttr.IsOptional(), idAttr.IsComputed())
	}

	nameAttr := s.Attributes["name"]
	if nameAttr.IsRequired() {
		t.Errorf("data source name must not be required (ExactlyOneOf with id)")
	}
	if !nameAttr.IsOptional() || !nameAttr.IsComputed() {
		t.Errorf("data source name must be optional+computed, got optional=%v computed=%v", nameAttr.IsOptional(), nameAttr.IsComputed())
	}
}

func TestVolumePurchasingLocationListResource_Metadata(t *testing.T) {
	r := NewVolumePurchasingLocationListResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*VolumePurchasingLocationListResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_volume_purchasing_location" {
		t.Errorf("expected list type name %q, got %q", "jamfplatform_pro_volume_purchasing_location", resp.TypeName)
	}
}

func TestVolumePurchasingLocationListResource_Schema(t *testing.T) {
	r := NewVolumePurchasingLocationListResource()
	var resp list.ListResourceSchemaResponse
	r.(*VolumePurchasingLocationListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
