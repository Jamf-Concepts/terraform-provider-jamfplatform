// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package service_discovery_enrollment

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestServiceDiscoveryEnrollmentResource_Metadata(t *testing.T) {
	r := NewServiceDiscoveryEnrollmentResource()
	var resp resource.MetadataResponse
	r.(*ServiceDiscoveryEnrollmentResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_service_discovery_enrollment" {
		t.Errorf("type name = %q, want jamfplatform_pro_service_discovery_enrollment", resp.TypeName)
	}
}

func TestServiceDiscoveryEnrollmentResource_Schema(t *testing.T) {
	r := NewServiceDiscoveryEnrollmentResource()
	var resp resource.SchemaResponse
	r.(*ServiceDiscoveryEnrollmentResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	for _, name := range []string{"id", "well_known_setting", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if id := s.Attributes["id"]; id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only")
	}

	wks, ok := s.Attributes["well_known_setting"].(rschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("well_known_setting must be a ListNestedAttribute, got %T", s.Attributes["well_known_setting"])
	}
	if !wks.IsRequired() {
		t.Errorf("well_known_setting must be required")
	}

	nested := wks.NestedObject.Attributes
	if su := nested["server_uuid"]; !su.IsRequired() {
		t.Errorf("server_uuid must be required")
	}
	if et := nested["enrollment_type"]; !et.IsRequired() {
		t.Errorf("enrollment_type must be required")
	}
	on := nested["org_name"]
	if on == nil || on.IsRequired() || on.IsOptional() || !on.IsComputed() {
		t.Errorf("org_name must be computed-only")
	}
}

func TestServiceDiscoveryEnrollmentDataSource_Schema(t *testing.T) {
	d := NewServiceDiscoveryEnrollmentDataSource()
	var resp datasource.SchemaResponse
	d.(*ServiceDiscoveryEnrollmentDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("data source schema diagnostics: %v", resp.Diagnostics)
	}
	for _, name := range []string{"id", "well_known_setting", "timeouts"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("data source missing attribute %q", name)
		}
	}
}

func TestServiceDiscoveryEnrollmentResource_IdentitySchema(t *testing.T) {
	r := NewServiceDiscoveryEnrollmentResource()
	var resp resource.IdentitySchemaResponse
	r.(*ServiceDiscoveryEnrollmentResource).IdentitySchema(context.Background(), resource.IdentitySchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("identity schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Errorf("identity schema missing id attribute")
	}
}

// TestServiceDiscoveryEnrollmentResource_ImportRejectsNonSingleton verifies any id
// other than the fixed singleton id is rejected. The success path is exercised by
// the acceptance test's ImportStateId: "singleton" step.
func TestServiceDiscoveryEnrollmentResource_ImportRejectsNonSingleton(t *testing.T) {
	r := NewServiceDiscoveryEnrollmentResource().(*ServiceDiscoveryEnrollmentResource)

	var bad resource.ImportStateResponse
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "not-the-singleton"}, &bad)
	if !bad.Diagnostics.HasError() {
		t.Errorf("import with a non-singleton id should be rejected")
	}
}
