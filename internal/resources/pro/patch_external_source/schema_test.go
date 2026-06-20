// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_external_source

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestPatchExternalSourceResource_Metadata(t *testing.T) {
	r := NewPatchExternalSourceResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*PatchExternalSourceResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_patch_external_source" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_patch_external_source", resp.TypeName)
	}
}

func TestPatchExternalSourceResource_Schema(t *testing.T) {
	r := NewPatchExternalSourceResource()
	var resp resource.SchemaResponse
	r.(*PatchExternalSourceResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "name", "enabled", "host_name", "port", "ssl_enabled", "certificate_validation_enabled", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	name := s.Attributes["name"]
	if !name.IsRequired() {
		t.Errorf("name must be required")
	}

	// enabled / ssl_enabled / certificate_validation_enabled are Optional+Computed
	// (server-defaulted toggles surfaced through the Computed slot).
	for _, b := range []string{"enabled", "ssl_enabled", "certificate_validation_enabled"} {
		a := s.Attributes[b]
		if !a.IsOptional() {
			t.Errorf("%q must be optional", b)
		}
		if !a.IsComputed() {
			t.Errorf("%q must be computed", b)
		}
	}

	// host_name is Required: the server mandates a non-blank host on create.
	// port is Optional-only (not Computed) so 0/empty collapses to null.
	host := s.Attributes["host_name"]
	if !host.IsRequired() {
		t.Errorf("host_name must be required, got required=%v", host.IsRequired())
	}
	port := s.Attributes["port"]
	if !port.IsOptional() || port.IsComputed() {
		t.Errorf("port must be optional-only, got optional=%v computed=%v", port.IsOptional(), port.IsComputed())
	}
	// port must carry an AtLeast(1) validator: an explicit 0 in config would
	// drift against the server's empty-port echo (which the read collapses to
	// null), so 0/negative are rejected at plan time.
	portAttr, ok := port.(schema.Int64Attribute)
	if !ok {
		t.Fatalf("port attribute is not an Int64Attribute")
	}
	if len(portAttr.Validators) == 0 {
		t.Errorf("port must carry at least one validator (AtLeast(1))")
	}
}

func TestPatchExternalSourceDataSource_Metadata(t *testing.T) {
	d := NewPatchExternalSourceDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*PatchExternalSourceDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_patch_external_source" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_patch_external_source", resp.TypeName)
	}
}

func TestPatchExternalSourceDataSource_Schema(t *testing.T) {
	d := NewPatchExternalSourceDataSource()
	var resp datasource.SchemaResponse
	d.(*PatchExternalSourceDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "name", "enabled", "host_name", "port", "ssl_enabled", "certificate_validation_enabled", "available_titles", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	// available_titles is a Computed-only nested catalog list.
	titles := s.Attributes["available_titles"]
	if titles.IsOptional() || titles.IsRequired() || !titles.IsComputed() {
		t.Errorf("available_titles must be computed-only, got optional=%v required=%v computed=%v", titles.IsOptional(), titles.IsRequired(), titles.IsComputed())
	}

	// id and name are the Optional+Computed selectors — exactly one is supplied
	// by the user, the other is filled from the SDK response.
	for _, sel := range []string{"id", "name"} {
		a := s.Attributes[sel]
		if !a.IsOptional() {
			t.Errorf("%q must be optional", sel)
		}
		if !a.IsComputed() {
			t.Errorf("%q must be computed (populated from response)", sel)
		}
	}

	// All remaining attributes are Computed-only — surfaced from the response.
	for _, c := range []string{"enabled", "host_name", "port", "ssl_enabled", "certificate_validation_enabled"} {
		a := s.Attributes[c]
		if a.IsOptional() || a.IsRequired() {
			t.Errorf("%q must be computed-only, got optional=%v required=%v", c, a.IsOptional(), a.IsRequired())
		}
		if !a.IsComputed() {
			t.Errorf("%q must be computed", c)
		}
	}
}

func TestPatchExternalSourceDataSource_ConfigValidators_ExactlyOneSelector(t *testing.T) {
	d := NewPatchExternalSourceDataSource().(*PatchExternalSourceDataSource)
	got := d.ConfigValidators(context.Background())
	if len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestPatchExternalSourceListResource_Metadata(t *testing.T) {
	r := NewPatchExternalSourceListResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*PatchExternalSourceListResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_patch_external_source" {
		t.Errorf("expected list type name %q, got %q", "jamfplatform_pro_patch_external_source", resp.TypeName)
	}
}

func TestPatchExternalSourceListResource_Schema(t *testing.T) {
	r := NewPatchExternalSourceListResource()
	var resp list.ListResourceSchemaResponse
	r.(*PatchExternalSourceListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
