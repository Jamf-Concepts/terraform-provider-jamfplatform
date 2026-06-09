// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package impact_alert_notification_settings

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// boolAttrNames is the set of Optional+Computed boolean toggles on the resource.
var boolAttrNames = []string{
	"deployable_objects_alert_enabled",
	"deployable_objects_confirmation_code_enabled",
	"scopeable_objects_alert_enabled",
	"scopeable_objects_confirmation_code_enabled",
}

func TestImpactAlertNotificationSettingsResource_Metadata(t *testing.T) {
	r := NewImpactAlertNotificationSettingsResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*ImpactAlertNotificationSettingsResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_impact_alert_notification_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_impact_alert_notification_settings", resp.TypeName)
	}
}

func TestImpactAlertNotificationSettingsResource_Schema(t *testing.T) {
	r := NewImpactAlertNotificationSettingsResource()
	var resp resource.SchemaResponse
	r.(*ImpactAlertNotificationSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	want := append([]string{"id", "timeouts"}, boolAttrNames...)
	for _, name := range want {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	for _, name := range boolAttrNames {
		a := s.Attributes[name]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%s must be optional+computed, got optional=%v computed=%v", name, a.IsOptional(), a.IsComputed())
		}
	}
}

func TestImpactAlertNotificationSettingsResource_IdentitySchema(t *testing.T) {
	r := NewImpactAlertNotificationSettingsResource()
	var resp resource.IdentitySchemaResponse
	r.(*ImpactAlertNotificationSettingsResource).IdentitySchema(context.Background(), resource.IdentitySchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("identity schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Errorf("identity schema missing id attribute")
	}
}

func TestImpactAlertNotificationSettingsResource_ConfigValidatorsWired(t *testing.T) {
	r := NewImpactAlertNotificationSettingsResource()
	validators := r.(*ImpactAlertNotificationSettingsResource).ConfigValidators(context.Background())
	if len(validators) == 0 {
		t.Errorf("expected at least one ConfigValidator (confirmation-code requires alert)")
	}
}

func TestImpactAlertNotificationSettingsDataSource_Metadata(t *testing.T) {
	d := NewImpactAlertNotificationSettingsDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*ImpactAlertNotificationSettingsDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_impact_alert_notification_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_impact_alert_notification_settings", resp.TypeName)
	}
}

func TestImpactAlertNotificationSettingsDataSource_Schema(t *testing.T) {
	d := NewImpactAlertNotificationSettingsDataSource()
	var resp datasource.SchemaResponse
	d.(*ImpactAlertNotificationSettingsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	want := append([]string{"id", "timeouts"}, boolAttrNames...)
	for _, name := range want {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if !s.Attributes["id"].IsComputed() {
		t.Errorf("data source id must be computed for singleton")
	}
}
