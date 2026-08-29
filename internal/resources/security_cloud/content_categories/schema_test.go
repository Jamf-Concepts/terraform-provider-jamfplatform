// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package content_categories

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestContentCategoriesDataSource_Metadata(t *testing.T) {
	d := NewContentCategoriesDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*ContentCategoriesDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_security_cloud_content_categories" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_security_cloud_content_categories", resp.TypeName)
	}
}

func TestContentCategoriesDataSource_Schema(t *testing.T) {
	d := NewContentCategoriesDataSource()
	var resp datasource.SchemaResponse
	d.(*ContentCategoriesDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "content_categories", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	categories, ok := s.Attributes["content_categories"].(dsschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("content_categories must be a ListNestedAttribute, got %T", s.Attributes["content_categories"])
	}
	for _, name := range []string{"id", "display_name", "name"} {
		if _, present := categories.NestedObject.Attributes[name]; !present {
			t.Errorf("content_categories missing nested attribute %q", name)
		}
	}
}

// TestContentCategoriesDataSource_IsEntirelyReadOnly pins the shape of a catalogue
// nobody manages: every attribute is Computed, and there is no argument to filter
// or select by because the endpoint accepts none.
func TestContentCategoriesDataSource_IsEntirelyReadOnly(t *testing.T) {
	d := NewContentCategoriesDataSource()
	var resp datasource.SchemaResponse
	d.(*ContentCategoriesDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	for name, attr := range resp.Schema.Attributes {
		if name == "timeouts" {
			continue
		}
		if attr.IsRequired() || attr.IsOptional() {
			t.Errorf("%s must be computed-only; the catalogue takes no arguments", name)
		}
	}
}

// TestContentCategoriesDataSource_PointsAtDisplayName is the one description this
// data source exists to carry. A category has two names and only one of them works
// as a Zero Trust Network Access app's category; a reader who picks `name` gets a
// server-side rejection naming neither field.
func TestContentCategoriesDataSource_PointsAtDisplayName(t *testing.T) {
	d := NewContentCategoriesDataSource()
	var resp datasource.SchemaResponse
	d.(*ContentCategoriesDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	categories := resp.Schema.Attributes["content_categories"].(dsschema.ListNestedAttribute)
	displayName := categories.NestedObject.Attributes["display_name"].GetMarkdownDescription()
	if !strings.Contains(displayName, "must match") {
		t.Errorf("display_name description must say it is the name to match, got: %s", displayName)
	}
	internal := categories.NestedObject.Attributes["name"].GetMarkdownDescription()
	if !strings.Contains(internal, "display_name") {
		t.Errorf("name description must redirect the reader to display_name, got: %s", internal)
	}
}
