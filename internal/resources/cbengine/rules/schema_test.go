// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package rules

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestRulesDataSource_Metadata(t *testing.T) {
	d := NewRulesDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*RulesDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_cbengine_rules" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_cbengine_rules", resp.TypeName)
	}
}

func TestRulesDataSource_Schema(t *testing.T) {
	d := NewRulesDataSource()
	req := datasource.SchemaRequest{}
	var resp datasource.SchemaResponse
	d.(*RulesDataSource).Schema(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema

	baselineID, ok := s.Attributes["baseline_id"]
	if !ok {
		t.Fatal("missing baseline_id attribute")
	}
	if !baselineID.IsRequired() {
		t.Error("baseline_id should be required")
	}

	computedAttrs := []string{"sources", "rules"}
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

	rules, ok := s.Attributes["rules"]
	if !ok {
		t.Fatal("missing rules attribute")
	}
	nested, ok := rules.(datasourceschema.ListNestedAttribute)
	if !ok {
		t.Fatal("rules should be a ListNestedAttribute")
	}

	expectedRuleAttrs := []string{"id", "section_name", "enabled", "title"}
	for _, name := range expectedRuleAttrs {
		if _, ok := nested.NestedObject.Attributes[name]; !ok {
			t.Errorf("rules nested object missing attribute %q", name)
		}
	}
}
