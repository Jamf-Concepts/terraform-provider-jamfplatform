// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_request_settings

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const wantTypeName = "jamfplatform_pro_app_request_settings"

func TestAppRequestSettingsResource_Metadata(t *testing.T) {
	r := NewAppRequestSettingsResource()
	var resp resource.MetadataResponse
	r.(*AppRequestSettingsResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != wantTypeName {
		t.Errorf("expected type name %q, got %q", wantTypeName, resp.TypeName)
	}
}

func TestAppRequestSettingsResource_Schema(t *testing.T) {
	r := NewAppRequestSettingsResource()
	var resp resource.SchemaResponse
	r.(*AppRequestSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "enabled", "app_store_locale", "approver_emails", "requester_user_group_id", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	if emails := s.Attributes["approver_emails"]; !emails.IsRequired() {
		t.Errorf("approver_emails must be required")
	}
	for _, oc := range []string{"enabled", "app_store_locale", "requester_user_group_id"} {
		a := s.Attributes[oc]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%q must be optional+computed (omit=preserve), got optional=%v computed=%v", oc, a.IsOptional(), a.IsComputed())
		}
	}
}

// TestApproverEmails_AcceptsAnEmptySet guards against reintroducing a minimum-size
// validator. App Request settings are a per-tenant singleton that always exists, and a
// tenant with App Requests switched off has no approvers — the server holds that state
// happily, only the Jamf Pro admin UI insists on one. A plan-time minimum made that
// tenant inexpressible, so `terraform plan -generate-config-out` emitted configuration
// this provider then rejected and the settings could not be adopted at all.
func TestApproverEmails_AcceptsAnEmptySet(t *testing.T) {
	r := NewAppRequestSettingsResource()
	var resp resource.SchemaResponse
	r.(*AppRequestSettingsResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	emails, ok := resp.Schema.Attributes["approver_emails"].(schema.SetAttribute)
	if !ok {
		t.Fatalf("approver_emails is not a SetAttribute")
	}

	empty, diags := types.SetValue(types.StringType, []attr.Value{})
	if diags.HasError() {
		t.Fatalf("building an empty set: %v", diags)
	}

	for _, v := range emails.Validators {
		var vResp validator.SetResponse
		v.ValidateSet(context.Background(), validator.SetRequest{
			Path:        path.Root("approver_emails"),
			ConfigValue: empty,
		}, &vResp)
		if vResp.Diagnostics.HasError() {
			t.Errorf("an empty approver_emails set must validate cleanly: %v", vResp.Diagnostics)
		}
	}
}
