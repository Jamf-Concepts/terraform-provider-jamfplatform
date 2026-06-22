// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package user

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestUserDataSource_Metadata(t *testing.T) {
	d := NewUserDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*UserDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_user" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_user", resp.TypeName)
	}
}

func TestUserDataSource_Schema(t *testing.T) {
	d := NewUserDataSource()
	var resp datasource.SchemaResponse
	d.(*UserDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{
		"id", "username", "full_name", "email_address", "phone_number",
		"position", "managed_apple_id", "enable_custom_photo_url", "custom_photo_url", "timeouts",
	} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}

func TestUserDataSource_ConfigValidators(t *testing.T) {
	d := NewUserDataSource().(*UserDataSource)
	if got := len(d.ConfigValidators(context.Background())); got != 1 {
		t.Errorf("expected exactly one config validator (id/username mutual exclusivity), got %d", got)
	}
}
