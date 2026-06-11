// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package macos_onboarding

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestOnboardingResource_Metadata(t *testing.T) {
	r := NewOnboardingResource()
	var resp resource.MetadataResponse
	r.(*OnboardingResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_macos_onboarding" {
		t.Errorf("type name = %q, want jamfplatform_pro_macos_onboarding", resp.TypeName)
	}
}

func TestOnboardingResource_Schema(t *testing.T) {
	r := NewOnboardingResource()
	var resp resource.SchemaResponse
	r.(*OnboardingResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	for _, name := range []string{"id", "enabled", "onboarding_items", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if id := s.Attributes["id"]; id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only")
	}
	if e := s.Attributes["enabled"]; !e.IsRequired() {
		t.Errorf("enabled must be required")
	}
	if oi := s.Attributes["onboarding_items"]; !oi.IsRequired() {
		t.Errorf("onboarding_items must be required")
	}
}

func TestOnboardingResource_IdentitySchema(t *testing.T) {
	r := NewOnboardingResource()
	var resp resource.IdentitySchemaResponse
	r.(*OnboardingResource).IdentitySchema(context.Background(), resource.IdentitySchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("identity schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Errorf("identity schema missing id attribute")
	}
}

// TestOnboardingResource_ImportRejectsNonSingleton verifies any id other than the
// fixed singleton id is rejected with a clear diagnostic. The success path (which
// calls ImportStatePassthroughID against a schema-wired State) is exercised by the
// acceptance test's ImportStateId: "singleton" step.
func TestOnboardingResource_ImportRejectsNonSingleton(t *testing.T) {
	r := NewOnboardingResource().(*OnboardingResource)

	var bad resource.ImportStateResponse
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "not-the-singleton"}, &bad)
	if !bad.Diagnostics.HasError() {
		t.Errorf("import with a non-singleton id should be rejected")
	}
}

func TestOnboardingDataSource_Metadata(t *testing.T) {
	d := NewOnboardingDataSource()
	var resp datasource.MetadataResponse
	d.(*OnboardingDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_macos_onboarding" {
		t.Errorf("type name = %q, want jamfplatform_pro_macos_onboarding", resp.TypeName)
	}
}

func TestOnboardingDataSource_Schema(t *testing.T) {
	d := NewOnboardingDataSource()
	var resp datasource.SchemaResponse
	d.(*OnboardingDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	for _, name := range []string{"id", "enabled", "onboarding_items", "timeouts"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}

func TestOnboardingEligibleItemsDataSource_Metadata(t *testing.T) {
	d := NewOnboardingEligibleItemsDataSource()
	var resp datasource.MetadataResponse
	d.(*OnboardingEligibleItemsDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_macos_onboarding_eligible_items" {
		t.Errorf("type name = %q, want jamfplatform_pro_macos_onboarding_eligible_items", resp.TypeName)
	}
}

func TestOnboardingEligibleItemsDataSource_Schema(t *testing.T) {
	d := NewOnboardingEligibleItemsDataSource()
	var resp datasource.SchemaResponse
	d.(*OnboardingEligibleItemsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	for _, name := range []string{"id", "entity_type", "items"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if et := resp.Schema.Attributes["entity_type"]; !et.IsRequired() {
		t.Errorf("entity_type must be required")
	}
}

// TestValidEntityTypes pins the accepted self_service_entity_type vocabulary so an
// accidental enum change is caught.
func TestValidEntityTypes(t *testing.T) {
	want := map[string]bool{"OS_X_POLICY": true, "OS_X_CONFIG_PROFILE": true, "OS_X_MAC_APP": true, "OS_X_APP_INSTALLER": true}
	if len(validEntityTypes) != len(want) {
		t.Fatalf("validEntityTypes = %v, want %d values", validEntityTypes, len(want))
	}
	for _, v := range validEntityTypes {
		if !want[v] {
			t.Errorf("unexpected entity type %q", v)
		}
	}
}
