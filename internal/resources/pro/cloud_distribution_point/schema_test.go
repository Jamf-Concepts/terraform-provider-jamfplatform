// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package cloud_distribution_point

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestCloudDistributionPointResource_Metadata(t *testing.T) {
	r := NewCloudDistributionPointResource()
	var resp resource.MetadataResponse
	r.(*CloudDistributionPointResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)

	if want := "jamfplatform_pro_cloud_distribution_point"; resp.TypeName != want {
		t.Errorf("type name = %q, want %q", resp.TypeName, want)
	}
}

func TestCloudDistributionPointResource_Schema(t *testing.T) {
	r := NewCloudDistributionPointResource()
	var resp resource.SchemaResponse
	r.(*CloudDistributionPointResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	want := []string{
		"id", "cdn_type", "master", "username", "password", "directory",
		"upload_url", "download_url", "cdn_url", "require_signed_urls",
		"key_pair_id", "private_key", "expiration_seconds",
		"secondary_auth_required", "secondary_auth_time_to_live",
		"secondary_auth_status_code", "has_connection_succeeded", "message",
		"inventory_id", "timeouts",
	}
	for _, name := range want {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	// id: computed-only.
	if id := s.Attributes["id"]; id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only")
	}

	// cdn_type: required.
	if ct := s.Attributes["cdn_type"]; !ct.IsRequired() {
		t.Errorf("cdn_type must be required")
	}

	// WriteOnly + Sensitive secrets, no _wo_version companions (API requires
	// them on every write — see model_types.go).
	for _, name := range []string{"password", "private_key"} {
		a := s.Attributes[name]
		if !a.IsOptional() || !a.IsSensitive() || !a.IsWriteOnly() {
			t.Errorf("%s must be Optional + Sensitive + WriteOnly, got optional=%v sensitive=%v writeOnly=%v",
				name, a.IsOptional(), a.IsSensitive(), a.IsWriteOnly())
		}
		if a.IsComputed() {
			t.Errorf("%s must not be Computed (WriteOnly secret)", name)
		}
	}
	if _, ok := s.Attributes["password_wo_version"]; ok {
		t.Errorf("password_wo_version must not exist — password is API-required and always sent")
	}

	// Server-derived echoes: computed-only.
	for _, name := range []string{"cdn_url", "secondary_auth_status_code", "has_connection_succeeded", "message", "inventory_id"} {
		a := s.Attributes[name]
		if a.IsRequired() || a.IsOptional() || !a.IsComputed() {
			t.Errorf("%s must be computed-only, got required=%v optional=%v computed=%v", name, a.IsRequired(), a.IsOptional(), a.IsComputed())
		}
	}
}

func TestCloudDistributionPointResource_ConfigValidatorsRegistered(t *testing.T) {
	r := NewCloudDistributionPointResource().(*CloudDistributionPointResource)
	if got := len(r.ConfigValidators(context.Background())); got != 1 {
		t.Errorf("expected 1 config validator, got %d", got)
	}
}

func TestCloudDistributionPointResource_IdentitySchema(t *testing.T) {
	r := NewCloudDistributionPointResource()
	var resp resource.IdentitySchemaResponse
	r.(*CloudDistributionPointResource).IdentitySchema(context.Background(), resource.IdentitySchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("identity schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Errorf("identity schema missing id attribute")
	}
}

func TestCloudDistributionPointDataSource_Schema(t *testing.T) {
	d := NewCloudDistributionPointDataSource()
	var resp datasource.SchemaResponse
	d.(*CloudDistributionPointDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema
	if !s.Attributes["id"].IsComputed() {
		t.Errorf("data source id must be computed")
	}
	// WriteOnly secrets must NOT appear on the data source (never returned).
	for _, name := range []string{"password", "private_key"} {
		if _, ok := s.Attributes[name]; ok {
			t.Errorf("data source must not expose secret %q", name)
		}
	}
}
