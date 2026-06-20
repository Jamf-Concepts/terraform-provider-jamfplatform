// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package printer

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestPrinterResource_Metadata(t *testing.T) {
	r := NewPrinterResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*PrinterResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_printer" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_printer", resp.TypeName)
	}
}

func TestPrinterResource_Schema(t *testing.T) {
	r := NewPrinterResource()
	var resp resource.SchemaResponse
	r.(*PrinterResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	required := []string{
		"id", "name", "category", "uri", "cups_name", "location", "model",
		"info", "notes", "make_default", "use_generic", "ppd", "ppd_path",
		"ppd_contents", "shared", "os_requirements", "timeouts",
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

	if !s.Attributes["name"].IsRequired() {
		t.Errorf("name must be required")
	}

	// ppd_path is Optional+Computed because the server auto-fills it under
	// use_generic=true (the bundled Generic.ppd path).
	pp := s.Attributes["ppd_path"]
	if !pp.IsOptional() || !pp.IsComputed() {
		t.Errorf("ppd_path must be Optional+Computed, got optional=%v computed=%v", pp.IsOptional(), pp.IsComputed())
	}

	// ppd_contents is Optional+Computed and uses the trimmedStringType
	// custom type. Computed is required so the framework is permitted to
	// surface a value that differs from the user's config (the server
	// strips trailing whitespace). Semantic equality on the custom type
	// suppresses the resulting diff. Not Sensitive: PPD files are driver
	// descriptors, not secrets.
	pc := s.Attributes["ppd_contents"]
	if pc.IsSensitive() {
		t.Errorf("ppd_contents must NOT be Sensitive — PPD files are not secrets")
	}
	if !pc.IsOptional() || !pc.IsComputed() {
		t.Errorf("ppd_contents must be Optional+Computed (needs Computed so trimmedStringType semantic equality can normalise server-side whitespace strip), got optional=%v computed=%v", pc.IsOptional(), pc.IsComputed())
	}

	// use_generic and make_default both have Default(...) so they must be
	// Optional+Computed.
	for _, name := range []string{"use_generic", "make_default", "shared"} {
		a := s.Attributes[name]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%q must be Optional+Computed, got optional=%v computed=%v", name, a.IsOptional(), a.IsComputed())
		}
	}
}

func TestPrinterResource_ConfigValidators(t *testing.T) {
	r := NewPrinterResource().(*PrinterResource)
	got := r.ConfigValidators(context.Background())
	if len(got) != 1 {
		t.Fatalf("expected 1 config validator (use_generic PPD trio), got %d", len(got))
	}
}

func TestPrinterDataSource_Metadata(t *testing.T) {
	d := NewPrinterDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*PrinterDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_printer" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_printer", resp.TypeName)
	}
}

func TestPrinterDataSource_Schema(t *testing.T) {
	d := NewPrinterDataSource()
	var resp datasource.SchemaResponse
	d.(*PrinterDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "name", "category", "ppd_path", "shared", "os_requirements", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	for _, sel := range []string{"id", "name"} {
		a := s.Attributes[sel]
		if !a.IsOptional() {
			t.Errorf("%q must be optional", sel)
		}
		if !a.IsComputed() {
			t.Errorf("%q must be computed (populated from response)", sel)
		}
	}
}

func TestPrinterDataSource_ConfigValidators_ExactlyOneSelector(t *testing.T) {
	d := NewPrinterDataSource().(*PrinterDataSource)
	got := d.ConfigValidators(context.Background())
	if len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestPrinterListResource_Metadata(t *testing.T) {
	r := NewPrinterListResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*PrinterListResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_printer" {
		t.Errorf("expected list type name %q, got %q", "jamfplatform_pro_printer", resp.TypeName)
	}
}

func TestPrinterListResource_Schema(t *testing.T) {
	r := NewPrinterListResource()
	var resp list.ListResourceSchemaResponse
	r.(*PrinterListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
