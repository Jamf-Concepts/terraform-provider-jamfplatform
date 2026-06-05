// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_enrollment_profile

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

const wantTypeName = "jamfplatform_pro_mobile_device_enrollment_profile"

func TestResource_Metadata(t *testing.T) {
	r := NewEnrollmentProfileResource()
	var resp resource.MetadataResponse
	r.(*EnrollmentProfileResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != wantTypeName {
		t.Errorf("type name = %q, want %q", resp.TypeName, wantTypeName)
	}
}

func TestResource_Schema(t *testing.T) {
	r := NewEnrollmentProfileResource()
	var resp resource.SchemaResponse
	r.(*EnrollmentProfileResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diags: %v", resp.Diagnostics)
	}
	s := resp.Schema
	for _, n := range []string{"id", "name", "description", "site_id", "site_name", "invitation", "uuid", "location", "purchasing", "attachments", "timeouts"} {
		if _, ok := s.Attributes[n]; !ok {
			t.Errorf("missing attribute %q", n)
		}
	}
	if !s.Attributes["name"].IsRequired() {
		t.Error("name must be required")
	}
	for _, c := range []string{"site_name", "invitation", "uuid"} {
		a := s.Attributes[c]
		if !a.IsComputed() || a.IsOptional() || a.IsRequired() {
			t.Errorf("%q must be computed-only", c)
		}
	}
	if a := s.Attributes["attachments"]; !a.IsComputed() || a.IsOptional() {
		t.Error("attachments must be computed-only (read-only)")
	}
	if a := s.Attributes["site_id"]; !a.IsOptional() || !a.IsComputed() {
		t.Error("site_id must be optional+computed")
	}
}

func TestDataSource_Schema_And_Validators(t *testing.T) {
	d := NewEnrollmentProfileDataSource()
	var resp datasource.SchemaResponse
	d.(*EnrollmentProfileDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ds schema diags: %v", resp.Diagnostics)
	}
	for _, sel := range []string{"id", "name", "invitation"} {
		a := resp.Schema.Attributes[sel]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%q must be optional+computed selector", sel)
		}
	}
	if got := d.(*EnrollmentProfileDataSource).ConfigValidators(context.Background()); len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestListResource_Schema(t *testing.T) {
	r := NewEnrollmentProfileListResource()
	var resp list.ListResourceSchemaResponse
	r.(*EnrollmentProfileListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("list schema diags: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Error("list schema missing filter")
	}
}
