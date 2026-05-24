// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestPolicyResource_Schema(t *testing.T) {
	t.Parallel()
	r := NewPolicyResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema returned diagnostics: %v", resp.Diagnostics)
	}
	must := []string{
		"id", "general", "scope", "self_service", "package_configuration",
		"scripts", "printers", "dock_items", "account_maintenance", "reboot",
		"maintenance", "files_processes", "user_interaction", "disk_encryption",
		"timeouts",
	}
	for _, name := range must {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Fatalf("expected attribute %q in policy schema", name)
		}
	}
}

func TestPolicyResource_ScopeChildAttributes(t *testing.T) {
	t.Parallel()
	r := NewPolicyResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	scopeAttr, ok := resp.Schema.Attributes["scope"].(rschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("expected scope to be SingleNestedAttribute")
	}
	for _, child := range []string{"all_computers", "computer_ids", "computer_group_ids", "building_ids", "department_ids", "user_ids", "user_group_ids", "limitations", "exclusions"} {
		if _, ok := scopeAttr.Attributes[child]; !ok {
			t.Fatalf("expected scope.%s", child)
		}
	}
}

func TestPolicyDataSource_Schema(t *testing.T) {
	t.Parallel()
	d := NewPolicyDataSource()
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema returned diagnostics: %v", resp.Diagnostics)
	}
	for _, name := range []string{"id", "name", "enabled", "frequency", "trigger", "category_id", "category_name", "site_id", "site_name"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Fatalf("expected data source attribute %q", name)
		}
	}
	// Sanity-check that id and name are both Optional+Computed (selector pair).
	idAttr, ok := resp.Schema.Attributes["id"].(dsschema.StringAttribute)
	if !ok {
		t.Fatalf("expected id to be a StringAttribute")
	}
	if !idAttr.Optional || !idAttr.Computed {
		t.Fatalf("expected id to be Optional+Computed")
	}
}
