// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_software_title

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestPatchSoftwareTitleResource_Metadata(t *testing.T) {
	r := NewPatchSoftwareTitleResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*PatchSoftwareTitleResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_patch_software_title" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_patch_software_title", resp.TypeName)
	}
}

func TestPatchSoftwareTitleResource_Schema(t *testing.T) {
	r := NewPatchSoftwareTitleResource()
	var resp resource.SchemaResponse
	r.(*PatchSoftwareTitleResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	want := []string{"id", "name", "name_id", "source_id", "category_id", "category_name", "site_id", "site_name", "web_notification", "email_notification", "version_packages", "available_versions", "accept_extension_attributes", "extension_attributes", "timeouts"}
	for _, name := range want {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	for _, req := range []string{"name", "name_id", "source_id"} {
		if !s.Attributes[req].IsRequired() {
			t.Errorf("%q must be required", req)
		}
	}

	// category_id / site_id / both notification bools are Optional+Computed.
	for _, oc := range []string{"category_id", "site_id", "web_notification", "email_notification"} {
		a := s.Attributes[oc]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%q must be optional+computed, got optional=%v computed=%v", oc, a.IsOptional(), a.IsComputed())
		}
	}

	// category_name / site_name / available_versions are Computed-only.
	for _, c := range []string{"category_name", "site_name", "available_versions"} {
		a := s.Attributes[c]
		if a.IsOptional() || a.IsRequired() || !a.IsComputed() {
			t.Errorf("%q must be computed-only, got optional=%v required=%v computed=%v", c, a.IsOptional(), a.IsRequired(), a.IsComputed())
		}
	}

	// version_packages must be Optional-only (never Computed — avoids computed-map
	// plan quirks).
	vp := s.Attributes["version_packages"]
	if !vp.IsOptional() {
		t.Errorf("version_packages must be optional")
	}
	if vp.IsComputed() {
		t.Errorf("version_packages must NOT be computed (managed-subset map)")
	}

	// accept_extension_attributes is an Optional-only one-way action flag (never
	// Computed — it is a user intent input, not coupled to the computed list).
	accept := s.Attributes["accept_extension_attributes"]
	if !accept.IsOptional() || accept.IsComputed() || accept.IsRequired() {
		t.Errorf("accept_extension_attributes must be optional-only, got optional=%v computed=%v required=%v", accept.IsOptional(), accept.IsComputed(), accept.IsRequired())
	}

	// extension_attributes is Computed-only (no plan modifier) so it goes Unknown
	// when an accept flips accepted=false→true in the same apply.
	ea := s.Attributes["extension_attributes"]
	if ea.IsOptional() || ea.IsRequired() || !ea.IsComputed() {
		t.Errorf("extension_attributes must be computed-only, got optional=%v required=%v computed=%v", ea.IsOptional(), ea.IsRequired(), ea.IsComputed())
	}
}

func TestPatchSoftwareTitleDataSource_Metadata(t *testing.T) {
	d := NewPatchSoftwareTitleDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*PatchSoftwareTitleDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_patch_software_title" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_patch_software_title", resp.TypeName)
	}
}

func TestPatchSoftwareTitleDataSource_Schema(t *testing.T) {
	d := NewPatchSoftwareTitleDataSource()
	var resp datasource.SchemaResponse
	d.(*PatchSoftwareTitleDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	// id and name are the two mutually-exclusive selectors — Optional+Computed.
	for _, sel := range []string{"id", "name"} {
		a := s.Attributes[sel]
		if !a.IsOptional() {
			t.Errorf("%q must be optional (id-or-name selector)", sel)
		}
		if !a.IsComputed() {
			t.Errorf("%q must be computed (populated from response)", sel)
		}
	}

	// Everything else is Computed-only.
	for _, c := range []string{"name_id", "source_id", "category_id", "category_name", "site_id", "site_name", "web_notification", "email_notification", "version_packages", "available_versions"} {
		a, ok := s.Attributes[c]
		if !ok {
			t.Errorf("missing attribute %q", c)
			continue
		}
		if a.IsOptional() || a.IsRequired() || !a.IsComputed() {
			t.Errorf("%q must be computed-only, got optional=%v required=%v computed=%v", c, a.IsOptional(), a.IsRequired(), a.IsComputed())
		}
	}
}

func TestPatchSoftwareTitleDataSource_ConfigValidators_ExactlyOneSelector(t *testing.T) {
	d := NewPatchSoftwareTitleDataSource().(*PatchSoftwareTitleDataSource)
	got := d.ConfigValidators(context.Background())
	if len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestPatchSoftwareTitleListResource_Metadata(t *testing.T) {
	r := NewPatchSoftwareTitleListResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*PatchSoftwareTitleListResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_patch_software_title" {
		t.Errorf("expected list type name %q, got %q", "jamfplatform_pro_patch_software_title", resp.TypeName)
	}
}

func TestPatchSoftwareTitleListResource_Schema(t *testing.T) {
	r := NewPatchSoftwareTitleListResource()
	var resp list.ListResourceSchemaResponse
	r.(*PatchSoftwareTitleListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
