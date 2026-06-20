// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package script

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestScriptResource_Metadata(t *testing.T) {
	r := NewScriptResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*ScriptResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_script" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_script", resp.TypeName)
	}
}

func TestScriptResource_Schema(t *testing.T) {
	r := NewScriptResource()
	var resp resource.SchemaResponse
	r.(*ScriptResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	required := []string{
		"id", "name", "category_id", "category_name", "info", "notes",
		"os_requirements", "priority",
		"parameter_4", "parameter_5", "parameter_6", "parameter_7",
		"parameter_8", "parameter_9", "parameter_10", "parameter_11",
		"script_contents", "timeouts",
	}
	for _, name := range required {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	name := s.Attributes["name"]
	if !name.IsRequired() {
		t.Errorf("name must be required")
	}

	priority := s.Attributes["priority"]
	if !priority.IsOptional() || !priority.IsComputed() {
		t.Errorf("priority must be optional+computed, got optional=%v computed=%v", priority.IsOptional(), priority.IsComputed())
	}

	categoryName := s.Attributes["category_name"]
	if categoryName.IsRequired() || categoryName.IsOptional() || !categoryName.IsComputed() {
		t.Errorf("category_name must be computed-only, got required=%v optional=%v computed=%v",
			categoryName.IsRequired(), categoryName.IsOptional(), categoryName.IsComputed())
	}
}

func TestScriptDataSource_Metadata(t *testing.T) {
	d := NewScriptDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*ScriptDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_script" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_script", resp.TypeName)
	}
}

func TestScriptDataSource_Schema(t *testing.T) {
	d := NewScriptDataSource()
	var resp datasource.SchemaResponse
	d.(*ScriptDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	if _, ok := s.Attributes["id"]; !ok {
		t.Fatalf("missing id attribute")
	}
	if !s.Attributes["id"].IsRequired() {
		t.Errorf("data source id must be required")
	}
	for _, name := range []string{"name", "category_id", "category_name", "priority", "script_contents", "parameter_4", "parameter_11"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}

func TestScriptListResource_Metadata(t *testing.T) {
	r := NewScriptListResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*ScriptListResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_script" {
		t.Errorf("expected list type name %q, got %q", "jamfplatform_pro_script", resp.TypeName)
	}
}

func TestScriptListResource_Schema(t *testing.T) {
	r := NewScriptListResource()
	var resp list.ListResourceSchemaResponse
	r.(*ScriptListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
