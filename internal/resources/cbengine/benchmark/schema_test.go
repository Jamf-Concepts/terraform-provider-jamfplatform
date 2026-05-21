// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package benchmark

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// --- Resource ---

func TestBenchmarkResource_Metadata(t *testing.T) {
	r := NewBenchmarkResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*BenchmarkResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_cbengine_benchmark" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_cbengine_benchmark", resp.TypeName)
	}
}

func TestBenchmarkResource_Schema(t *testing.T) {
	r := NewBenchmarkResource()
	req := resource.SchemaRequest{}
	var resp resource.SchemaResponse
	r.(*BenchmarkResource).Schema(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	if s.Version != 0 {
		t.Errorf("expected schema version 0, got %d", s.Version)
	}

	requiredAttrs := []string{"title", "sources", "rules", "enforcement_mode"}
	for _, name := range requiredAttrs {
		attr, ok := s.Attributes[name]
		if !ok {
			t.Errorf("missing required attribute %q", name)
			continue
		}
		if !attr.IsRequired() {
			t.Errorf("attribute %q should be required", name)
		}
	}

	computedAttrs := []string{"id", "tenant_id", "deleted", "update_available", "last_updated_at"}
	for _, name := range computedAttrs {
		attr, ok := s.Attributes[name]
		if !ok {
			t.Errorf("missing computed attribute %q", name)
			continue
		}
		if !attr.IsComputed() {
			t.Errorf("attribute %q should be computed", name)
		}
	}

	optionalAttrs := []string{"description", "source_baseline_id", "timeouts", "target_device_group", "target_device_groups"}
	for _, name := range optionalAttrs {
		attr, ok := s.Attributes[name]
		if !ok {
			t.Errorf("missing optional attribute %q", name)
			continue
		}
		if !attr.IsOptional() {
			t.Errorf("attribute %q should be optional", name)
		}
	}

	singular, _ := s.Attributes["target_device_group"].(resourceschema.StringAttribute)
	if singular.DeprecationMessage == "" {
		t.Errorf("target_device_group should carry a DeprecationMessage")
	}
}

func TestBenchmarkResource_ConfigValidators(t *testing.T) {
	r := NewBenchmarkResource().(*BenchmarkResource)
	got := r.ConfigValidators(context.Background())
	if len(got) != 1 {
		t.Fatalf("expected 1 resource-level config validator, got %d", len(got))
	}
}

func TestBenchmarkResource_SchemaRulesStructure(t *testing.T) {
	r := NewBenchmarkResource()
	req := resource.SchemaRequest{}
	var resp resource.SchemaResponse
	r.(*BenchmarkResource).Schema(context.Background(), req, &resp)

	rules, ok := resp.Schema.Attributes["rules"]
	if !ok {
		t.Fatal("missing rules attribute")
	}
	nested, ok := rules.(resourceschema.ListNestedAttribute)
	if !ok {
		t.Fatal("rules should be a ListNestedAttribute")
	}

	expectedRuleAttrs := []string{"id", "enabled", "section_name", "title", "description", "odv_value"}
	for _, name := range expectedRuleAttrs {
		if _, ok := nested.NestedObject.Attributes[name]; !ok {
			t.Errorf("rules nested object missing attribute %q", name)
		}
	}
}

func TestBenchmarkResource_SchemaSourcesStructure(t *testing.T) {
	r := NewBenchmarkResource()
	req := resource.SchemaRequest{}
	var resp resource.SchemaResponse
	r.(*BenchmarkResource).Schema(context.Background(), req, &resp)

	sources, ok := resp.Schema.Attributes["sources"]
	if !ok {
		t.Fatal("missing sources attribute")
	}
	nested, ok := sources.(resourceschema.ListNestedAttribute)
	if !ok {
		t.Fatal("sources should be a ListNestedAttribute")
	}

	for _, name := range []string{"branch", "revision"} {
		if _, ok := nested.NestedObject.Attributes[name]; !ok {
			t.Errorf("sources nested object missing attribute %q", name)
		}
	}
}

// --- Data Source ---

func TestBenchmarkDataSource_Metadata(t *testing.T) {
	d := NewBenchmarkDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*BenchmarkDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_cbengine_benchmark" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_cbengine_benchmark", resp.TypeName)
	}
}

func TestBenchmarkDataSource_Schema(t *testing.T) {
	d := NewBenchmarkDataSource()
	req := datasource.SchemaRequest{}
	var resp datasource.SchemaResponse
	d.(*BenchmarkDataSource).Schema(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	optionalAttrs := []string{"id", "title"}
	for _, name := range optionalAttrs {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Errorf("missing optional attribute %q", name)
			continue
		}
		if !attr.IsOptional() {
			t.Errorf("attribute %q should be optional", name)
		}
	}

	computedAttrs := []string{"benchmark_id", "tenant_id", "description", "sources", "rules"}
	for _, name := range computedAttrs {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Errorf("missing computed attribute %q", name)
			continue
		}
		if !attr.IsComputed() {
			t.Errorf("attribute %q should be computed", name)
		}
	}
}

// --- List Resource ---

func TestBenchmarkListResource_Metadata(t *testing.T) {
	r := NewBenchmarkListResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*BenchmarkListResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_cbengine_benchmark" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_cbengine_benchmark", resp.TypeName)
	}
}
