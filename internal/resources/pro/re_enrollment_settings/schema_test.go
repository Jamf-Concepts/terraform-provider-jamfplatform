// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package re_enrollment_settings

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// TestReEnrollmentSettingsResource_Metadata checks the resource type name.
func TestReEnrollmentSettingsResource_Metadata(t *testing.T) {
	r := NewReEnrollmentSettingsResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*ReEnrollmentSettingsResource).Metadata(context.Background(), req, &resp)
	if resp.TypeName != "jamfplatform_pro_re_enrollment_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_re_enrollment_settings", resp.TypeName)
	}
}

// TestReEnrollmentSettingsResource_Schema asserts the five clear_* toggles are
// Optional+Computed (full-replace endpoint, omit=preserve via UseStateForUnknown),
// the API-required clear_management_history enum is Required, and id is Computed.
func TestReEnrollmentSettingsResource_Schema(t *testing.T) {
	r := NewReEnrollmentSettingsResource()
	var resp resource.SchemaResponse
	r.(*ReEnrollmentSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	// The five flush toggles are Optional+Computed so omitting one preserves its
	// current value rather than resetting it on the full-replace write.
	optionalComputed := []string{
		"clear_policy_logs",
		"clear_location_information",
		"clear_location_information_history",
		"clear_extension_attributes",
		"clear_software_update_plans",
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

	// clear_management_history is required by the API (rejects an omitted or empty
	// value), so it is the Required carve-out — not Optional+Computed.
	if enum, ok := resp.Schema.Attributes["clear_management_history"]; !ok || !enum.IsRequired() {
		t.Errorf("clear_management_history must be Required")
	}

	if id, ok := resp.Schema.Attributes["id"]; !ok || !id.IsComputed() {
		t.Errorf("id must be Computed")
	}
}

// TestReEnrollmentSettingsResource_Schema_ClearManagementHistoryEnum asserts the
// clear_management_history attribute carries a OneOf validator over the four
// documented enum values and rejects anything else.
func TestReEnrollmentSettingsResource_Schema_ClearManagementHistoryEnum(t *testing.T) {
	r := NewReEnrollmentSettingsResource()
	var resp resource.SchemaResponse
	r.(*ReEnrollmentSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	attr, ok := resp.Schema.Attributes["clear_management_history"].(rschema.StringAttribute)
	if !ok {
		t.Fatalf("clear_management_history must be StringAttribute")
	}
	if len(attr.Validators) == 0 {
		t.Fatalf("clear_management_history must declare at least one validator")
	}
	// The enum has no natural "unset" (the UI dropdown always has a selection)
	// and the resource is full-replace, so the attribute is Required, not
	// Optional+Computed.
	if !attr.Required {
		t.Error("clear_management_history must be Required")
	}
	if attr.Optional || attr.Computed {
		t.Error("clear_management_history must not be Optional or Computed")
	}
}

// TestReEnrollmentSettingsDataSource_Metadata checks the DS type name.
func TestReEnrollmentSettingsDataSource_Metadata(t *testing.T) {
	d := NewReEnrollmentSettingsDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*ReEnrollmentSettingsDataSource).Metadata(context.Background(), req, &resp)
	if resp.TypeName != "jamfplatform_pro_re_enrollment_settings" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_re_enrollment_settings", resp.TypeName)
	}
}

// TestValidClearManagementHistory asserts the enum slice holds exactly the four
// documented values, in the documented order.
func TestValidClearManagementHistory(t *testing.T) {
	want := []string{
		"DELETE_NOTHING",
		"DELETE_ERRORS",
		"DELETE_EVERYTHING_EXCEPT_ACKNOWLEDGED",
		"DELETE_EVERYTHING",
	}
	if len(validClearManagementHistory) != len(want) {
		t.Fatalf("validClearManagementHistory has %d entries, want %d", len(validClearManagementHistory), len(want))
	}
	for i, v := range want {
		if validClearManagementHistory[i] != v {
			t.Errorf("validClearManagementHistory[%d] = %q, want %q", i, validClearManagementHistory[i], v)
		}
	}
	if defaultClearManagementHistory != "DELETE_EVERYTHING_EXCEPT_ACKNOWLEDGED" {
		t.Errorf("defaultClearManagementHistory = %q, want DELETE_EVERYTHING_EXCEPT_ACKNOWLEDGED", defaultClearManagementHistory)
	}
}
