// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_configuration_profile

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestResource_Schema(t *testing.T) {
	t.Parallel()
	r := NewResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema returned diagnostics: %v", resp.Diagnostics)
	}
	for _, name := range []string{"id", "general", "scope", "self_service", "timeouts"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Fatalf("expected attribute %q", name)
		}
	}
}

func TestResource_GeneralAttributes(t *testing.T) {
	t.Parallel()
	r := NewResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	g, ok := resp.Schema.Attributes["general"].(rschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("expected general to be SingleNestedAttribute")
	}
	for _, child := range []string{"id", "name", "description", "level", "distribution_method", "redeploy_on_update", "redeploy_days_before_certificate_expires", "uuid", "payloads", "category_id", "category_name", "site_id", "site_name"} {
		if _, ok := g.Attributes[child]; !ok {
			t.Fatalf("expected general.%s", child)
		}
	}
	// mobile does NOT have user_removable
	if _, ok := g.Attributes["user_removable"]; ok {
		t.Fatal("general.user_removable must not exist on mobile profiles")
	}
}

func TestResource_ScopeAttributes(t *testing.T) {
	t.Parallel()
	r := NewResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	s, ok := resp.Schema.Attributes["scope"].(rschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("expected scope to be SingleNestedAttribute")
	}
	for _, child := range []string{"targets", "limitations", "exclusions"} {
		if _, ok := s.Attributes[child]; !ok {
			t.Fatalf("expected scope.%s", child)
		}
	}
	targets, ok := s.Attributes["targets"].(rschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("expected scope.targets to be SingleNestedAttribute")
	}
	for _, child := range []string{"all_mobile_devices", "all_jss_users", "mobile_device_ids", "mobile_device_group_ids", "building_ids", "department_ids", "user_ids", "user_group_ids"} {
		if _, ok := targets.Attributes[child]; !ok {
			t.Fatalf("expected scope.targets.%s", child)
		}
	}
	// mobile does NOT have computer scope attrs
	for _, absent := range []string{"all_computers", "computer_ids", "computer_group_ids"} {
		if _, ok := targets.Attributes[absent]; ok {
			t.Fatalf("scope.targets.%s must not exist on mobile profiles", absent)
		}
	}
}

func TestResource_SelfServiceAttributes(t *testing.T) {
	t.Parallel()
	r := NewResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	ss, ok := resp.Schema.Attributes["self_service"].(rschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("expected self_service to be SingleNestedAttribute")
	}
	for _, child := range []string{"self_service_description", "feature_on_main_page", "removal_disallowed", "authorization_password", "categories"} {
		if _, ok := ss.Attributes[child]; !ok {
			t.Fatalf("expected self_service.%s", child)
		}
	}
	// mobile does NOT have notification or display_name attrs
	for _, absent := range []string{"self_service_display_name", "install_button_text", "display_notifications", "notification_location", "notification_subject", "notification_message"} {
		if _, ok := ss.Attributes[absent]; ok {
			t.Fatalf("self_service.%s must not exist on mobile profiles", absent)
		}
	}
}

func TestResource_PayloadsAttributeRequired(t *testing.T) {
	t.Parallel()
	r := NewResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	g := resp.Schema.Attributes["general"].(rschema.SingleNestedAttribute)
	p, ok := g.Attributes["payloads"].(rschema.StringAttribute)
	if !ok {
		t.Fatalf("expected general.payloads to be StringAttribute")
	}
	if !p.Required {
		t.Fatal("expected general.payloads to be Required")
	}
	// Payload diff suppression is implemented at the resource level via
	// ModifyPlan (so it can read and write three-way-compare references
	// in private state); the attribute itself carries no plan modifier.
	if _, ok := r.(resource.ResourceWithModifyPlan); !ok {
		t.Fatal("expected Resource to implement ResourceWithModifyPlan for payload diff suppression")
	}
}

func TestResource_AuthorizationPasswordSensitive(t *testing.T) {
	t.Parallel()
	r := NewResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	ss := resp.Schema.Attributes["self_service"].(rschema.SingleNestedAttribute)
	rp, ok := ss.Attributes["authorization_password"].(rschema.StringAttribute)
	if !ok {
		t.Fatalf("expected self_service.authorization_password to be StringAttribute")
	}
	if !rp.Sensitive {
		t.Fatal("authorization_password must be Sensitive")
	}
}
