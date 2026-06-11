// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package users

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestUsersDataSource_Metadata(t *testing.T) {
	d := NewUsersDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*UsersDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_users" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_users", resp.TypeName)
	}
}

func TestUsersDataSource_Schema(t *testing.T) {
	d := NewUsersDataSource()
	var resp datasource.SchemaResponse
	d.(*UsersDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "timeouts", "filter", "users"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	usersAttr, ok := s.Attributes["users"]
	if !ok {
		t.Fatal("missing users attribute")
	}
	nested, ok := usersAttr.(datasourceschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("users should be ListNestedAttribute, got %T", usersAttr)
	}
	for _, name := range []string{
		"id", "username", "full_name", "email_address", "phone_number",
		"position", "managed_apple_id", "enable_custom_photo_url", "custom_photo_url",
	} {
		if _, ok := nested.NestedObject.Attributes[name]; !ok {
			t.Errorf("users nested missing attribute %q", name)
		}
	}
}
