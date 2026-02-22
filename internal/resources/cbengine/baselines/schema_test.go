// Copyright 2026 Jamf Software LLC.

package baselines

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestBaselinesDataSource_Metadata(t *testing.T) {
	d := NewBaselinesDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*BaselinesDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_cbengine_baselines" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_cbengine_baselines", resp.TypeName)
	}
}

func TestBaselinesDataSource_Schema(t *testing.T) {
	d := NewBaselinesDataSource()
	req := datasource.SchemaRequest{}
	var resp datasource.SchemaResponse
	d.(*BaselinesDataSource).Schema(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema

	if _, ok := s.Attributes["timeouts"]; !ok {
		t.Error("missing timeouts attribute")
	}

	baselines, ok := s.Attributes["baselines"]
	if !ok {
		t.Fatal("missing baselines attribute")
	}
	nested, ok := baselines.(datasourceschema.ListNestedAttribute)
	if !ok {
		t.Fatal("baselines should be a ListNestedAttribute")
	}

	expectedAttrs := []string{"id", "baseline_id", "title", "description", "rule_count"}
	for _, name := range expectedAttrs {
		if _, ok := nested.NestedObject.Attributes[name]; !ok {
			t.Errorf("baselines nested object missing attribute %q", name)
		}
	}
}
