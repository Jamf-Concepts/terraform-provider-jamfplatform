// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pkg

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestPackageResource_Metadata(t *testing.T) {
	r := NewPackageResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*PackageResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_package" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_package", resp.TypeName)
	}
}

func TestPackageResource_Schema(t *testing.T) {
	r := NewPackageResource()
	var resp resource.SchemaResponse
	r.(*PackageResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema

	// Required attrs (UI-aligned per §3.Q5).
	for _, name := range []string{"display_name", "file_name"} {
		a, ok := s.Attributes[name]
		if !ok {
			t.Errorf("missing attribute %q", name)
			continue
		}
		if !a.IsRequired() {
			t.Errorf("attribute %q must be Required", name)
		}
	}

	// Computed-only attrs (server-derived, no user input path).
	computedOnly := []string{
		"id", "manifest", "manifest_file_name", "size", "install_language",
		"parent_package_id", "self_healing_action", "self_heal_notify",
		"cloud_transfer_status", "indexed", "format",
	}
	for _, name := range computedOnly {
		a, ok := s.Attributes[name]
		if !ok {
			t.Errorf("missing attribute %q", name)
			continue
		}
		if a.IsOptional() || a.IsRequired() {
			t.Errorf("attribute %q must be Computed-only (got optional=%v required=%v)", name, a.IsOptional(), a.IsRequired())
		}
		if !a.IsComputed() {
			t.Errorf("attribute %q must be Computed", name)
		}
	}

	// Optional+Computed attrs (server-defaulted; user may set).
	optionalComputed := []string{
		"category_id", "info", "notes", "priority", "fill_user_template",
		"fill_existing_users", "reboot_required", "os_requirements",
		"available_in_software_update", "sha3_512", "sha256", "md5",
		"hash_type", "hash_value",
	}
	for _, name := range optionalComputed {
		a, ok := s.Attributes[name]
		if !ok {
			t.Errorf("missing attribute %q", name)
			continue
		}
		if !a.IsOptional() {
			t.Errorf("attribute %q must be Optional", name)
		}
		if !a.IsComputed() {
			t.Errorf("attribute %q must be Computed", name)
		}
	}

	// Pure-Optional inputs (no server echo).
	for _, name := range []string{"package_file_source", "package_file_source_checksum", "manifest_file_source"} {
		a, ok := s.Attributes[name]
		if !ok {
			t.Errorf("missing attribute %q", name)
			continue
		}
		if !a.IsOptional() {
			t.Errorf("attribute %q must be Optional", name)
		}
		if a.IsComputed() {
			t.Errorf("attribute %q must NOT be Computed — provider-internal input", name)
		}
	}

	if _, ok := s.Attributes["timeouts"]; !ok {
		t.Errorf("missing timeouts attribute")
	}
}

func TestPackageResource_IdentitySchema(t *testing.T) {
	r := NewPackageResource()
	var resp resource.IdentitySchemaResponse
	r.(*PackageResource).IdentitySchema(context.Background(), resource.IdentitySchemaRequest{}, &resp)

	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Errorf("identity schema missing id attribute")
	}
}

func TestPackageDataSource_Metadata(t *testing.T) {
	d := NewPackageDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*PackageDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_package" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_package", resp.TypeName)
	}
}

func TestPackageDataSource_Schema(t *testing.T) {
	d := NewPackageDataSource()
	var resp datasource.SchemaResponse
	d.(*PackageDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "display_name"} {
		a := s.Attributes[name]
		if !a.IsOptional() {
			t.Errorf("%q must be optional selector", name)
		}
		if !a.IsComputed() {
			t.Errorf("%q must be computed (populated from response)", name)
		}
	}

	// A handful of fields confirming the rest is wired.
	for _, name := range []string{"file_name", "manifest", "sha3_512", "cloud_transfer_status"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
}

func TestPackageDataSource_ConfigValidators(t *testing.T) {
	d := NewPackageDataSource().(*PackageDataSource)
	got := d.ConfigValidators(context.Background())
	if len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestPackageListResource_Metadata(t *testing.T) {
	r := NewPackageListResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*PackageListResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_package" {
		t.Errorf("expected list type name %q, got %q", "jamfplatform_pro_package", resp.TypeName)
	}
}

func TestPackageListResource_Schema(t *testing.T) {
	r := NewPackageListResource()
	var resp list.ListResourceSchemaResponse
	r.(*PackageListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
