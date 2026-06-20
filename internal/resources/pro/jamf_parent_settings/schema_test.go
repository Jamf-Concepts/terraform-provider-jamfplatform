// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package jamf_parent_settings

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// TestJamfParentSettingsResource_Metadata checks the resource type name.
func TestJamfParentSettingsResource_Metadata(t *testing.T) {
	r := NewJamfParentSettingsResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*JamfParentSettingsResource).Metadata(context.Background(), req, &resp)
	if resp.TypeName != "jamfplatform_pro_jamf_parent_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_jamf_parent_settings", resp.TypeName)
	}
}

// TestJamfParentSettingsResource_Schema asserts the §768.2 shape: every
// user-settable optional field is Optional+Computed (full-replace endpoint,
// omit=preserve via UseStateForUnknown), the API-required trio — timezone,
// device_group_id and restricted_times — are the Required carve-outs, and id
// is Computed.
func TestJamfParentSettingsResource_Schema(t *testing.T) {
	r := NewJamfParentSettingsResource()
	var resp resource.SchemaResponse
	r.(*JamfParentSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	optionalComputed := []string{
		"enabled",
		"allow_clear_passcode",
		"revoke_on_wipe_and_re_enroll",
		"safelisted_apps",
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

	// The API-required trio (§768 carve-out): timezoneId and restrictedTimes
	// omission is an HTTP 500, an omitted deviceGroupId decodes to 0 and 400s.
	for _, name := range []string{"timezone", "device_group_id", "restricted_times"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok || !attr.IsRequired() {
			t.Errorf("%s must be Required", name)
		}
	}

	if id, ok := resp.Schema.Attributes["id"]; !ok || !id.IsComputed() {
		t.Errorf("id must be Computed")
	}
}

// TestJamfParentSettingsResource_Schema_PlanModifiersPresent guards the §768
// anti-pattern: Optional+Computed on a full-replace endpoint without its
// UseStateForUnknown plan modifier is a silent wipe. Every converted attribute
// must carry exactly that modifier.
func TestJamfParentSettingsResource_Schema_PlanModifiersPresent(t *testing.T) {
	r := NewJamfParentSettingsResource()
	var resp resource.SchemaResponse
	r.(*JamfParentSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	for _, name := range []string{"enabled", "allow_clear_passcode", "revoke_on_wipe_and_re_enroll"} {
		if a, ok := resp.Schema.Attributes[name].(rschema.BoolAttribute); !ok || len(a.PlanModifiers) == 0 {
			t.Errorf("%s must carry boolplanmodifier.UseStateForUnknown", name)
		}
	}
	if a, ok := resp.Schema.Attributes["safelisted_apps"].(rschema.SetNestedAttribute); !ok || len(a.PlanModifiers) == 0 {
		t.Errorf("safelisted_apps must carry setplanmodifier.UseStateForUnknown")
	}
}

// TestJamfParentSettingsResource_Schema_RestrictedTimesValidators asserts the
// restricted_times plan-time gates are wired: the strict-UPPERCASE day-name
// key validator on the map, and the HH:MM:SS validator on both Required
// nested times.
func TestJamfParentSettingsResource_Schema_RestrictedTimesValidators(t *testing.T) {
	r := NewJamfParentSettingsResource()
	var resp resource.SchemaResponse
	r.(*JamfParentSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	m, ok := resp.Schema.Attributes["restricted_times"].(rschema.MapNestedAttribute)
	if !ok {
		t.Fatalf("restricted_times must be MapNestedAttribute")
	}
	if len(m.Validators) == 0 {
		t.Errorf("restricted_times must declare the uppercase day-name key validator")
	}
	for _, name := range []string{"begin_time", "end_time"} {
		attr, ok := m.NestedObject.Attributes[name].(rschema.StringAttribute)
		if !ok {
			t.Fatalf("restricted_times.%s must be StringAttribute", name)
		}
		if !attr.Required {
			t.Errorf("restricted_times.%s must be Required", name)
		}
		if len(attr.Validators) == 0 {
			t.Errorf("restricted_times.%s must declare the HH:MM:SS validator", name)
		}
	}
}

// TestJamfParentSettingsResource_Schema_SafelistValidators asserts the
// safelist hygiene validators are wired: unique bundle_id across the set and
// non-empty Required element fields.
func TestJamfParentSettingsResource_Schema_SafelistValidators(t *testing.T) {
	r := NewJamfParentSettingsResource()
	var resp resource.SchemaResponse
	r.(*JamfParentSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	set, ok := resp.Schema.Attributes["safelisted_apps"].(rschema.SetNestedAttribute)
	if !ok {
		t.Fatalf("safelisted_apps must be SetNestedAttribute")
	}
	if len(set.Validators) == 0 {
		t.Errorf("safelisted_apps must declare the unique-bundle_id validator")
	}
	for _, name := range []string{"name", "bundle_id"} {
		attr, ok := set.NestedObject.Attributes[name].(rschema.StringAttribute)
		if !ok {
			t.Fatalf("safelisted_apps.%s must be StringAttribute", name)
		}
		if !attr.Required {
			t.Errorf("safelisted_apps.%s must be Required", name)
		}
		if len(attr.Validators) == 0 {
			t.Errorf("safelisted_apps.%s must declare LengthAtLeast(1)", name)
		}
	}
}
