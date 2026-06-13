// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package managed_software_updates

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// computedOnlySubEnables is the set of server-managed, read-only boolean attributes.
var computedOnlySubEnables = []string{
	"dss_enabled",
	"recipe_enabled",
	"force_install_local_date_enabled",
	"custom_version_enabled",
}

func TestManagedSoftwareUpdateResource_Metadata(t *testing.T) {
	r := NewManagedSoftwareUpdateResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*ManagedSoftwareUpdateResource).Metadata(context.Background(), req, &resp)

	const want = "jamfplatform_pro_managed_software_update"
	if resp.TypeName != want {
		t.Errorf("expected type name %q, got %q", want, resp.TypeName)
	}
}

func TestManagedSoftwareUpdateResource_Schema(t *testing.T) {
	r := NewManagedSoftwareUpdateResource()
	var resp resource.SchemaResponse
	r.(*ManagedSoftwareUpdateResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	want := append([]string{"id", "enabled", "timeouts"}, computedOnlySubEnables...)
	for _, name := range want {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	enabled := s.Attributes["enabled"]
	if !enabled.IsOptional() || !enabled.IsComputed() {
		t.Errorf("enabled must be optional+computed, got optional=%v computed=%v", enabled.IsOptional(), enabled.IsComputed())
	}

	// The four sub-enables are server-managed and must be computed-only — never settable,
	// or a user could try to write a value the server ignores.
	for _, name := range computedOnlySubEnables {
		a := s.Attributes[name]
		if a.IsOptional() || a.IsRequired() || !a.IsComputed() {
			t.Errorf("%s must be computed-only, got optional=%v required=%v computed=%v", name, a.IsOptional(), a.IsRequired(), a.IsComputed())
		}
	}
}

func TestManagedSoftwareUpdateResource_IdentitySchema(t *testing.T) {
	r := NewManagedSoftwareUpdateResource()
	var resp resource.IdentitySchemaResponse
	r.(*ManagedSoftwareUpdateResource).IdentitySchema(context.Background(), resource.IdentitySchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("identity schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Errorf("identity schema missing id attribute")
	}
}
