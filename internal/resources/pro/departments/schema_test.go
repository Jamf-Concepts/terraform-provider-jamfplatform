// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package departments

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestDepartmentsDataSource_Metadata(t *testing.T) {
	d := NewDepartmentsDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*DepartmentsDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_departments" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_departments", resp.TypeName)
	}
}

func TestDepartmentsDataSource_Schema(t *testing.T) {
	d := NewDepartmentsDataSource()
	var resp datasource.SchemaResponse
	d.(*DepartmentsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "timeouts", "filter", "departments"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	deps, ok := s.Attributes["departments"]
	if !ok {
		t.Fatal("missing departments attribute")
	}
	nested, ok := deps.(datasourceschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("departments should be ListNestedAttribute, got %T", deps)
	}
	for _, name := range []string{"id", "name"} {
		if _, ok := nested.NestedObject.Attributes[name]; !ok {
			t.Errorf("departments nested missing attribute %q", name)
		}
	}
}
