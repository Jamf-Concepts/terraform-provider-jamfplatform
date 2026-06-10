// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package jamf_teacher_settings

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// TestJamfTeacherSettingsResource_Metadata checks the resource type name.
func TestJamfTeacherSettingsResource_Metadata(t *testing.T) {
	r := NewJamfTeacherSettingsResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*JamfTeacherSettingsResource).Metadata(context.Background(), req, &resp)
	if resp.TypeName != "jamfplatform_pro_jamf_teacher_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_jamf_teacher_settings", resp.TypeName)
	}
}

// TestJamfTeacherSettingsResource_Schema asserts the §768.2 shape: every
// user-settable optional field is Optional+Computed (full-replace endpoint,
// omit=preserve via UseStateForUnknown), the API-required timezone is the
// Required carve-out, and id is Computed.
func TestJamfTeacherSettingsResource_Schema(t *testing.T) {
	r := NewJamfTeacherSettingsResource()
	var resp resource.SchemaResponse
	r.(*JamfTeacherSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	optionalComputed := []string{
		"enabled",
		"restrictions_end_time",
		"maximum_restriction_time_seconds",
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

	// timezone is mandatory on every PUT (omission -> HTTP 500), so it is the
	// §768 API-required carve-out — Required, not Optional+Computed.
	if tz, ok := resp.Schema.Attributes["timezone"]; !ok || !tz.IsRequired() {
		t.Errorf("timezone must be Required")
	}

	if id, ok := resp.Schema.Attributes["id"]; !ok || !id.IsComputed() {
		t.Errorf("id must be Computed")
	}
}

// TestJamfTeacherSettingsResource_Schema_PlanModifiersPresent guards the §768
// anti-pattern: Optional+Computed on a full-replace endpoint without its
// UseStateForUnknown plan modifier is a silent wipe. Every converted attribute
// must carry exactly that modifier.
func TestJamfTeacherSettingsResource_Schema_PlanModifiersPresent(t *testing.T) {
	r := NewJamfTeacherSettingsResource()
	var resp resource.SchemaResponse
	r.(*JamfTeacherSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	if a, ok := resp.Schema.Attributes["enabled"].(rschema.BoolAttribute); !ok || len(a.PlanModifiers) == 0 {
		t.Errorf("enabled must carry boolplanmodifier.UseStateForUnknown")
	}
	if a, ok := resp.Schema.Attributes["restrictions_end_time"].(rschema.StringAttribute); !ok || len(a.PlanModifiers) == 0 {
		t.Errorf("restrictions_end_time must carry stringplanmodifier.UseStateForUnknown")
	}
	if a, ok := resp.Schema.Attributes["maximum_restriction_time_seconds"].(rschema.Int64Attribute); !ok || len(a.PlanModifiers) == 0 {
		t.Errorf("maximum_restriction_time_seconds must carry int64planmodifier.UseStateForUnknown")
	}
	if a, ok := resp.Schema.Attributes["safelisted_apps"].(rschema.SetNestedAttribute); !ok || len(a.PlanModifiers) == 0 {
		t.Errorf("safelisted_apps must carry setplanmodifier.UseStateForUnknown")
	}
}

// TestJamfTeacherSettingsResource_Schema_SafelistValidators asserts the
// safelist hygiene validators are wired: unique bundle_id across the set and
// non-empty Required element fields.
func TestJamfTeacherSettingsResource_Schema_SafelistValidators(t *testing.T) {
	r := NewJamfTeacherSettingsResource()
	var resp resource.SchemaResponse
	r.(*JamfTeacherSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
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
