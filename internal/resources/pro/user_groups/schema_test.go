// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package user_groups

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestUserGroupsDataSource_Metadata(t *testing.T) {
	d := NewUserGroupsDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*UserGroupsDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_user_groups" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_user_groups", resp.TypeName)
	}
}

func TestUserGroupsDataSource_Schema(t *testing.T) {
	d := NewUserGroupsDataSource()
	var resp datasource.SchemaResponse
	d.(*UserGroupsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "timeouts", "filter", "user_groups"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	ugAttr, ok := s.Attributes["user_groups"]
	if !ok {
		t.Fatal("missing user_groups attribute")
	}
	nested, ok := ugAttr.(datasourceschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("user_groups should be ListNestedAttribute, got %T", ugAttr)
	}
	for _, name := range []string{"id", "name", "group_type", "notify_on_membership_change"} {
		if _, ok := nested.NestedObject.Attributes[name]; !ok {
			t.Errorf("user_groups nested missing attribute %q", name)
		}
	}
}

func TestGroupTypeFromIsSmart(t *testing.T) {
	tr := true
	if got := groupTypeFromIsSmart(&tr); got.ValueString() != "smart" {
		t.Errorf("expected smart, got %s", got.ValueString())
	}
	fa := false
	if got := groupTypeFromIsSmart(&fa); got.ValueString() != "static" {
		t.Errorf("expected static, got %s", got.ValueString())
	}
	if !groupTypeFromIsSmart(nil).IsNull() {
		t.Error("nil must yield null")
	}
}
