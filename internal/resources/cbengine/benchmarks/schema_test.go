// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package benchmarks

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestBenchmarksDataSource_Metadata(t *testing.T) {
	d := NewBenchmarksDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*BenchmarksDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_cbengine_benchmarks" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_cbengine_benchmarks", resp.TypeName)
	}
}

func TestBenchmarksDataSource_Schema(t *testing.T) {
	d := NewBenchmarksDataSource()
	req := datasource.SchemaRequest{}
	var resp datasource.SchemaResponse
	d.(*BenchmarksDataSource).Schema(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema

	if _, ok := s.Attributes["timeouts"]; !ok {
		t.Error("missing timeouts attribute")
	}

	benchmarks, ok := s.Attributes["benchmarks"]
	if !ok {
		t.Fatal("missing benchmarks attribute")
	}
	nested, ok := benchmarks.(datasourceschema.ListNestedAttribute)
	if !ok {
		t.Fatal("benchmarks should be a ListNestedAttribute")
	}

	expectedAttrs := []string{"id", "title", "description", "update_available", "sync_state", "target_device_groups"}
	for _, name := range expectedAttrs {
		if _, ok := nested.NestedObject.Attributes[name]; !ok {
			t.Errorf("benchmarks nested object missing attribute %q", name)
		}
	}
}
