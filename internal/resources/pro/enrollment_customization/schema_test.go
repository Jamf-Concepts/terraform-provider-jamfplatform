// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package enrollment_customization

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestEnrollmentCustomizationResource_Metadata(t *testing.T) {
	r := NewEnrollmentCustomizationResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*EnrollmentCustomizationResource).Metadata(context.Background(), req, &resp)
	if resp.TypeName != "jamfplatform_pro_enrollment_customization" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_enrollment_customization", resp.TypeName)
	}
}

func TestEnrollmentCustomizationResource_Schema(t *testing.T) {
	r := NewEnrollmentCustomizationResource()
	var resp resource.SchemaResponse
	r.(*EnrollmentCustomizationResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{
		"id", "display_name", "description", "site_id", "icon_source",
		"icon_source_hash", "branding_settings", "text_panes", "ldap_panes",
		"sso_panes", "timeouts",
	} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	if id := s.Attributes["id"]; id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only")
	}
	if dn := s.Attributes["display_name"]; !dn.IsRequired() {
		t.Errorf("display_name must be required")
	}
	if desc := s.Attributes["description"]; !desc.IsRequired() {
		t.Errorf("description must be required")
	}
	if hash := s.Attributes["icon_source_hash"]; hash.IsRequired() || !hash.IsComputed() {
		t.Errorf("icon_source_hash must be computed-only")
	}
	if bs := s.Attributes["branding_settings"]; !bs.IsRequired() {
		t.Errorf("branding_settings must be required")
	}
	if tp := s.Attributes["text_panes"]; tp.IsRequired() {
		t.Errorf("text_panes must be optional, not required")
	}
}

// TestEnrollmentCustomizationResource_SchemaValidate confirms the schema
// constructs cleanly — in particular that the icon_source <->
// branding_settings.icon_url ConflictsWith pair (which spans the top-level
// attribute and a nested attribute) is accepted by the framework's schema
// validator.
func TestEnrollmentCustomizationResource_SchemaValidate(t *testing.T) {
	r := NewEnrollmentCustomizationResource()
	var resp resource.SchemaResponse
	r.(*EnrollmentCustomizationResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	diags := resp.Schema.ValidateImplementation(context.Background())
	if diags.HasError() {
		t.Fatalf("schema implementation validation: %v", diags)
	}
}

func TestEnrollmentCustomizationDataSource_Metadata(t *testing.T) {
	d := NewEnrollmentCustomizationDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*EnrollmentCustomizationDataSource).Metadata(context.Background(), req, &resp)
	if resp.TypeName != "jamfplatform_pro_enrollment_customization" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_enrollment_customization", resp.TypeName)
	}
}

func TestEnrollmentCustomizationDataSource_Schema(t *testing.T) {
	d := NewEnrollmentCustomizationDataSource()
	var resp datasource.SchemaResponse
	d.(*EnrollmentCustomizationDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	for _, name := range []string{"id", "display_name", "description", "site_id", "branding_settings"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing data source attribute %q", name)
		}
	}
}

func TestEnrollmentCustomizationListResource_Metadata(t *testing.T) {
	r := NewEnrollmentCustomizationListResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*EnrollmentCustomizationListResource).Metadata(context.Background(), req, &resp)
	if resp.TypeName != "jamfplatform_pro_enrollment_customization" {
		t.Errorf("expected list type name %q, got %q", "jamfplatform_pro_enrollment_customization", resp.TypeName)
	}
}

func TestEnrollmentCustomizationListResource_SchemaHasFilter(t *testing.T) {
	r := NewEnrollmentCustomizationListResource()
	var resp list.ListResourceSchemaResponse
	r.(*EnrollmentCustomizationListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list resource missing filter attribute")
	}
}
