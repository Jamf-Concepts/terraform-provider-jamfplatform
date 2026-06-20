// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package licensed_software

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

const wantTypeName = "jamfplatform_pro_licensed_software"

func TestLicensedSoftwareResource_Metadata(t *testing.T) {
	r := NewLicensedSoftwareResource()
	var resp resource.MetadataResponse
	r.(*LicensedSoftwareResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != wantTypeName {
		t.Errorf("expected type name %q, got %q", wantTypeName, resp.TypeName)
	}
}

func TestLicensedSoftwareResource_Schema(t *testing.T) {
	r := NewLicensedSoftwareResource()
	var resp resource.SchemaResponse
	r.(*LicensedSoftwareResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	for _, name := range []string{
		"id", "name", "publisher", "platform", "notes", "send_email_on_violation",
		"remove_titles_from_inventory_reports", "exclude_titles_purchased_from_app_store",
		"site_id", "site_name", "software_definitions", "licenses", "computers", "timeouts",
	} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing top-level attribute %q", name)
		}
	}

	// The legacy font / plug-in buckets are intentionally not modeled.
	for _, name := range []string{"font_definitions", "plugin_definitions"} {
		if _, ok := s.Attributes[name]; ok {
			t.Errorf("schema must NOT expose %q (server drops it on write)", name)
		}
	}

	if id := s.Attributes["id"]; id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only")
	}
	if n := s.Attributes["name"]; !n.IsRequired() {
		t.Errorf("name must be required")
	}
	if sn := s.Attributes["site_name"]; sn.IsRequired() || sn.IsOptional() || !sn.IsComputed() {
		t.Errorf("site_name must be computed-only (derived from site_id)")
	}
	if comp := s.Attributes["computers"]; comp.IsRequired() || comp.IsOptional() || !comp.IsComputed() {
		t.Errorf("computers must be computed-only")
	}

	// software_definitions: ordered list; compare_type optional+computed; name required.
	defs, ok := s.Attributes["software_definitions"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatalf("software_definitions must be a ListNestedAttribute")
	}
	if !defs.IsOptional() {
		t.Errorf("software_definitions must be optional")
	}
	if name := defs.NestedObject.Attributes["name"]; !name.IsRequired() {
		t.Errorf("software_definitions.name must be required")
	}
	if ct := defs.NestedObject.Attributes["compare_type"]; !ct.IsOptional() || !ct.IsComputed() {
		t.Errorf("software_definitions.compare_type must be optional+computed")
	}
	if ct := defs.NestedObject.Attributes["compare_type"].(schema.StringAttribute); len(ct.Validators) == 0 {
		t.Errorf("software_definitions.compare_type must carry the OneOf validator")
	}

	// licenses: ordered list with purchasing nested + computed attachments.
	lic, ok := s.Attributes["licenses"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatalf("licenses must be a ListNestedAttribute")
	}
	if !lic.IsOptional() {
		t.Errorf("licenses must be optional")
	}
	if lc := lic.NestedObject.Attributes["license_count"]; !lc.IsOptional() || !lc.IsComputed() {
		t.Errorf("licenses.license_count must be optional+computed")
	}
	att, ok := lic.NestedObject.Attributes["attachments"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatalf("licenses.attachments must be a ListNestedAttribute")
	}
	if att.IsRequired() || att.IsOptional() || !att.IsComputed() {
		t.Errorf("licenses.attachments must be computed-only")
	}

	pur, ok := lic.NestedObject.Attributes["purchasing"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("licenses.purchasing must be a SingleNestedAttribute")
	}
	if !pur.IsOptional() {
		t.Errorf("licenses.purchasing must be optional")
	}
	if lt := pur.Attributes["license_term"]; !lt.IsRequired() {
		t.Errorf("licenses.purchasing.license_term must be required")
	}
	for _, name := range []string{"po_date_epoch", "po_date_utc", "license_expires_epoch", "license_expires_utc"} {
		a, ok := pur.Attributes[name]
		if !ok {
			t.Errorf("purchasing missing %q", name)
			continue
		}
		if a.IsRequired() || a.IsOptional() || !a.IsComputed() {
			t.Errorf("purchasing.%s must be computed-only (server-derived echo)", name)
		}
	}
	// is_perpetual / is_annual are collapsed into license_term — they must not leak.
	for _, name := range []string{"is_perpetual", "is_annual"} {
		if _, ok := pur.Attributes[name]; ok {
			t.Errorf("purchasing must NOT expose %q (collapsed into license_term)", name)
		}
	}
}

func TestLicensedSoftwareDataSource_Metadata(t *testing.T) {
	d := NewLicensedSoftwareDataSource()
	var resp datasource.MetadataResponse
	d.(*LicensedSoftwareDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != wantTypeName {
		t.Errorf("expected type name %q, got %q", wantTypeName, resp.TypeName)
	}
}

func TestLicensedSoftwareDataSource_ConfigValidators(t *testing.T) {
	d := NewLicensedSoftwareDataSource().(*LicensedSoftwareDataSource)
	if got := d.ConfigValidators(context.Background()); len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestLicensedSoftwareListResource_Schema(t *testing.T) {
	r := NewLicensedSoftwareListResource()
	var resp list.ListResourceSchemaResponse
	r.(*LicensedSoftwareListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
