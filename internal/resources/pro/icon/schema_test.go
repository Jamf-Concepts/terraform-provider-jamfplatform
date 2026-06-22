// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package icon

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestIconResource_Metadata(t *testing.T) {
	r := NewIconResource()
	req := resource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp resource.MetadataResponse
	r.(*IconResource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_icon" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_icon", resp.TypeName)
	}
}

func TestIconResource_Schema(t *testing.T) {
	r := NewIconResource()
	var resp resource.SchemaResponse
	r.(*IconResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "icon_file_source", "source_hash", "url", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be computed-only, got required=%v computed=%v", id.IsRequired(), id.IsComputed())
	}

	iconFileSource := s.Attributes["icon_file_source"]
	if !iconFileSource.IsRequired() {
		t.Errorf("icon_file_source must be required, got required=%v optional=%v computed=%v", iconFileSource.IsRequired(), iconFileSource.IsOptional(), iconFileSource.IsComputed())
	}
	if iconFileSource.IsOptional() || iconFileSource.IsComputed() {
		t.Errorf("icon_file_source must not be optional or computed")
	}

	sourceHash := s.Attributes["source_hash"]
	if sourceHash.IsOptional() || sourceHash.IsRequired() {
		t.Errorf("source_hash must be computed-only, got required=%v optional=%v computed=%v", sourceHash.IsRequired(), sourceHash.IsOptional(), sourceHash.IsComputed())
	}
	if !sourceHash.IsComputed() {
		t.Errorf("source_hash must be computed")
	}

	url := s.Attributes["url"]
	if url.IsRequired() || !url.IsComputed() {
		t.Errorf("url must be computed-only, got required=%v computed=%v", url.IsRequired(), url.IsComputed())
	}
}
