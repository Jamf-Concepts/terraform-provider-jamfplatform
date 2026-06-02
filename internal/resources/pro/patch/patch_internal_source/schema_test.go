// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_internal_source

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestPatchInternalSourceDataSource_Metadata(t *testing.T) {
	d := NewPatchInternalSourceDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*PatchInternalSourceDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_patch_internal_source" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_patch_internal_source", resp.TypeName)
	}
}

func TestPatchInternalSourceDataSource_Schema(t *testing.T) {
	d := NewPatchInternalSourceDataSource()
	var resp datasource.SchemaResponse
	d.(*PatchInternalSourceDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "name", "enabled", "endpoint", "available_titles", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	// id and name are the Optional+Computed selectors.
	for _, sel := range []string{"id", "name"} {
		a := s.Attributes[sel]
		if !a.IsOptional() {
			t.Errorf("%q must be optional", sel)
		}
		if !a.IsComputed() {
			t.Errorf("%q must be computed (populated from response)", sel)
		}
	}

	// Remaining attributes are Computed-only — surfaced from the response.
	for _, c := range []string{"enabled", "endpoint", "available_titles"} {
		a := s.Attributes[c]
		if a.IsOptional() || a.IsRequired() {
			t.Errorf("%q must be computed-only, got optional=%v required=%v", c, a.IsOptional(), a.IsRequired())
		}
		if !a.IsComputed() {
			t.Errorf("%q must be computed", c)
		}
	}
}

func TestPatchInternalSourceDataSource_ConfigValidators_ExactlyOneSelector(t *testing.T) {
	d := NewPatchInternalSourceDataSource().(*PatchInternalSourceDataSource)
	got := d.ConfigValidators(context.Background())
	if len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}
