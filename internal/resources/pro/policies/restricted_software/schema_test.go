// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package restricted_software

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

const wantTypeName = "jamfplatform_pro_restricted_software"

func TestRestrictedSoftwareResource_Metadata(t *testing.T) {
	r := NewRestrictedSoftwareResource()
	var resp resource.MetadataResponse
	r.(*RestrictedSoftwareResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != wantTypeName {
		t.Errorf("expected type name %q, got %q", wantTypeName, resp.TypeName)
	}
}

func TestRestrictedSoftwareResource_Schema(t *testing.T) {
	r := NewRestrictedSoftwareResource()
	var resp resource.SchemaResponse
	r.(*RestrictedSoftwareResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "general", "scope", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing top-level attribute %q", name)
		}
	}

	if id := s.Attributes["id"]; id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only")
	}
	if g := s.Attributes["general"]; !g.IsRequired() {
		t.Errorf("general must be required")
	}
	if sc := s.Attributes["scope"]; !sc.IsOptional() {
		t.Errorf("scope must be optional")
	}

	general, ok := s.Attributes["general"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("general must be a SingleNestedAttribute")
	}
	for _, name := range []string{"name", "process_name"} {
		a, ok := general.Attributes[name]
		if !ok {
			t.Errorf("general missing %q", name)
			continue
		}
		if !a.IsRequired() {
			t.Errorf("general.%s must be required", name)
		}
	}
	for _, name := range []string{"restrict_exact_process_name", "send_email_notification_on_violation", "kill_process", "delete_application", "display_message", "site_id"} {
		a, ok := general.Attributes[name]
		if !ok {
			t.Errorf("general missing %q", name)
			continue
		}
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("general.%s must be optional+computed", name)
		}
	}
	for _, name := range []string{"id", "site_name"} {
		a, ok := general.Attributes[name]
		if !ok {
			t.Errorf("general missing %q", name)
			continue
		}
		if a.IsRequired() || a.IsOptional() || !a.IsComputed() {
			t.Errorf("general.%s must be computed-only", name)
		}
	}

	scopeAttr, ok := s.Attributes["scope"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("scope must be a SingleNestedAttribute")
	}
	// Targets and exclusions are siblings under scope; no limitations tab.
	for _, name := range []string{"targets", "exclusions"} {
		if _, ok := scopeAttr.Attributes[name]; !ok {
			t.Errorf("scope missing %q", name)
		}
	}
	if _, ok := scopeAttr.Attributes["limitations"]; ok {
		t.Errorf("scope must NOT expose \"limitations\" (restricted software is a limited subset)")
	}

	targetsAttr, ok := scopeAttr.Attributes["targets"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("scope.targets must be a SingleNestedAttribute")
	}
	for _, name := range []string{"all_computers", "computer_ids", "computer_group_ids", "building_ids", "department_ids"} {
		if _, ok := targetsAttr.Attributes[name]; !ok {
			t.Errorf("scope.targets missing %q", name)
		}
	}
	// LIMITED subset: no target users / all_jss_users.
	for _, name := range []string{"all_jss_users", "user_ids", "user_group_ids"} {
		if _, ok := targetsAttr.Attributes[name]; ok {
			t.Errorf("scope.targets must NOT expose %q (restricted software is a limited subset)", name)
		}
	}
	if ac := targetsAttr.Attributes["all_computers"]; len(ac.(schema.BoolAttribute).Validators) == 0 {
		t.Errorf("scope.targets.all_computers must carry the conflict validator")
	}

	excl, ok := scopeAttr.Attributes["exclusions"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("scope.exclusions must be a SingleNestedAttribute")
	}
	for _, name := range []string{"computer_ids", "computer_group_ids", "building_ids", "department_ids", "directory_service_or_local_user_names"} {
		if _, ok := excl.Attributes[name]; !ok {
			t.Errorf("scope.exclusions missing %q", name)
		}
	}
	for _, name := range []string{"network_segment_ids", "user_group_ids", "ibeacon_ids", "directory_service_user_group_names"} {
		if _, ok := excl.Attributes[name]; ok {
			t.Errorf("scope.exclusions must NOT expose %q", name)
		}
	}
}

func TestRestrictedSoftwareDataSource_Metadata(t *testing.T) {
	d := NewRestrictedSoftwareDataSource()
	var resp datasource.MetadataResponse
	d.(*RestrictedSoftwareDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != wantTypeName {
		t.Errorf("expected type name %q, got %q", wantTypeName, resp.TypeName)
	}
}

func TestRestrictedSoftwareDataSource_ConfigValidators(t *testing.T) {
	d := NewRestrictedSoftwareDataSource().(*RestrictedSoftwareDataSource)
	if got := d.ConfigValidators(context.Background()); len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestRestrictedSoftwareListResource_Schema(t *testing.T) {
	r := NewRestrictedSoftwareListResource()
	var resp list.ListResourceSchemaResponse
	r.(*RestrictedSoftwareListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
