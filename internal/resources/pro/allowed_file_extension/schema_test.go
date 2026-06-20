// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package allowed_file_extension

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestAllowedFileExtensionResource_Metadata(t *testing.T) {
	r := NewAllowedFileExtensionResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*AllowedFileExtensionResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_allowed_file_extension" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_allowed_file_extension", resp.TypeName)
	}
}

func TestAllowedFileExtensionResource_Schema(t *testing.T) {
	r := NewAllowedFileExtensionResource()
	var resp resource.SchemaResponse
	r.(*AllowedFileExtensionResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "extension", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	ext := s.Attributes["extension"]
	if !ext.IsRequired() {
		t.Errorf("extension must be required")
	}
}

func TestNoSurroundingWhitespaceRegex(t *testing.T) {
	// The validator must reject leading/trailing whitespace (the server trims it, which
	// would otherwise drift) while leaving case, dots, and internal characters alone.
	accept := []string{"jpg", "JPG", ".tfdot", "tar.gz", "a"}
	reject := []string{"", " jpg", "jpg ", " jpg ", "\tjpg", "jpg\n"}

	for _, v := range accept {
		if !noSurroundingWhitespace.MatchString(v) {
			t.Errorf("expected %q to be accepted", v)
		}
	}
	for _, v := range reject {
		if noSurroundingWhitespace.MatchString(v) {
			t.Errorf("expected %q to be rejected", v)
		}
	}
}

func TestAllowedFileExtensionDataSource_Metadata(t *testing.T) {
	d := NewAllowedFileExtensionDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*AllowedFileExtensionDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_allowed_file_extension" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_allowed_file_extension", resp.TypeName)
	}
}

func TestAllowedFileExtensionDataSource_Schema(t *testing.T) {
	d := NewAllowedFileExtensionDataSource()
	var resp datasource.SchemaResponse
	d.(*AllowedFileExtensionDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "extension", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	// Both id and extension must be Optional+Computed — exactly one is supplied by the
	// user, the other is filled from the SDK response.
	for _, sel := range []string{"id", "extension"} {
		a := s.Attributes[sel]
		if !a.IsOptional() {
			t.Errorf("%q must be optional", sel)
		}
		if !a.IsComputed() {
			t.Errorf("%q must be computed (populated from response)", sel)
		}
	}
}

func TestAllowedFileExtensionDataSource_ConfigValidators_ExactlyOneSelector(t *testing.T) {
	d := NewAllowedFileExtensionDataSource().(*AllowedFileExtensionDataSource)
	got := d.ConfigValidators(context.Background())
	if len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestAllowedFileExtensionListResource_Metadata(t *testing.T) {
	r := NewAllowedFileExtensionListResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*AllowedFileExtensionListResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_allowed_file_extension" {
		t.Errorf("expected list type name %q, got %q", "jamfplatform_pro_allowed_file_extension", resp.TypeName)
	}
}

func TestAllowedFileExtensionListResource_Schema(t *testing.T) {
	r := NewAllowedFileExtensionListResource()
	var resp list.ListResourceSchemaResponse
	r.(*AllowedFileExtensionListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
