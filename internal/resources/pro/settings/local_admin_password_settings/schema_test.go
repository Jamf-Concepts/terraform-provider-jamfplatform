// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package local_admin_password_settings

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// TestLocalAdminPasswordSettingsResource_Metadata checks the resource type name.
func TestLocalAdminPasswordSettingsResource_Metadata(t *testing.T) {
	r := NewLocalAdminPasswordSettingsResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*LocalAdminPasswordSettingsResource).Metadata(context.Background(), req, &resp)
	if resp.TypeName != "jamfplatform_pro_local_admin_password_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_local_admin_password_settings", resp.TypeName)
	}
}

// TestLocalAdminPasswordSettingsResource_Schema asserts the three controls are
// Optional+Computed (full-replace endpoint, omit=preserve via UseStateForUnknown)
// and id is Computed.
func TestLocalAdminPasswordSettingsResource_Schema(t *testing.T) {
	r := NewLocalAdminPasswordSettingsResource()
	var resp resource.SchemaResponse
	r.(*LocalAdminPasswordSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	optionalComputed := []string{
		"laps_for_prestage_accounts_enabled",
		"rotation_interval",
		"rotation_after_viewing_interval",
	}
	for _, name := range optionalComputed {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Errorf("missing attribute %q", name)
			continue
		}
		if !attr.IsOptional() || !attr.IsComputed() {
			t.Errorf("attribute %q must be Optional+Computed (omit=preserve), got optional=%v computed=%v", name, attr.IsOptional(), attr.IsComputed())
		}
		if attr.IsRequired() {
			t.Errorf("attribute %q must not be Required", name)
		}
	}

	if id, ok := resp.Schema.Attributes["id"]; !ok || !id.IsComputed() {
		t.Errorf("id must be Computed")
	}
}

// TestLocalAdminPasswordSettingsResource_Schema_Enums asserts both interval
// attributes carry a OneOf validator.
func TestLocalAdminPasswordSettingsResource_Schema_Enums(t *testing.T) {
	r := NewLocalAdminPasswordSettingsResource()
	var resp resource.SchemaResponse
	r.(*LocalAdminPasswordSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	for _, name := range []string{"rotation_interval", "rotation_after_viewing_interval"} {
		attr, ok := resp.Schema.Attributes[name].(rschema.StringAttribute)
		if !ok {
			t.Fatalf("%s must be StringAttribute", name)
		}
		if len(attr.Validators) == 0 {
			t.Errorf("%s must declare at least one validator (OneOf)", name)
		}
	}
}

// TestValidRotationEnums asserts the enum slices hold exactly the documented
// values in UI dropdown order, and the duration tables agree with them.
func TestValidRotationEnums(t *testing.T) {
	wantInterval := []string{"Never", "7 days", "30 days", "60 days", "180 days"}
	if len(validRotationInterval) != len(wantInterval) {
		t.Fatalf("validRotationInterval has %d entries, want %d", len(validRotationInterval), len(wantInterval))
	}
	for i, v := range wantInterval {
		if validRotationInterval[i] != v {
			t.Errorf("validRotationInterval[%d] = %q, want %q", i, validRotationInterval[i], v)
		}
	}

	wantAfter := []string{"1 hour", "3 hours", "12 hours", "1 day", "3 days", "7 days"}
	if len(validRotationAfterViewingInterval) != len(wantAfter) {
		t.Fatalf("validRotationAfterViewingInterval has %d entries, want %d", len(validRotationAfterViewingInterval), len(wantAfter))
	}
	for i, v := range wantAfter {
		if validRotationAfterViewingInterval[i] != v {
			t.Errorf("validRotationAfterViewingInterval[%d] = %q, want %q", i, validRotationAfterViewingInterval[i], v)
		}
	}

	// Every non-"Never" rotation_interval label must have a duration mapping, and
	// every rotation_after_viewing label too.
	for _, v := range validRotationIntervalDurations {
		if _, ok := rotationIntervalDurationToValue[v]; !ok {
			t.Errorf("rotation_interval %q has no duration mapping", v)
		}
	}
	for _, v := range validRotationAfterViewingInterval {
		if _, ok := rotationAfterViewingToDuration[v]; !ok {
			t.Errorf("rotation_after_viewing %q has no duration mapping", v)
		}
	}

	// 7 days is the one shared, known-confirmed mapping (604800s).
	if rotationAfterViewingToDuration["7 days"] != 604800 || rotationIntervalDurationToValue["7 days"] != 604800 {
		t.Errorf("expected 7 days = 604800 in both tables")
	}
}
