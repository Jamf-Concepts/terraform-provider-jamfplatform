// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_request_settings

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
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
