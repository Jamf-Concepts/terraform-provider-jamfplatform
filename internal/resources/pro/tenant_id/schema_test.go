// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package tenant_id

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestTenantIDDataSource_Metadata(t *testing.T) {
	d := NewTenantIDDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*TenantIDDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_tenant_id" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_tenant_id", resp.TypeName)
	}
}

func TestTenantIDDataSource_Schema(t *testing.T) {
	d := NewTenantIDDataSource()
	var resp datasource.SchemaResponse
	d.(*TenantIDDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "tenant_id", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	// The data source takes no arguments: a provider instance is scoped to one
	// tenant, so there is nothing for a caller to select. An attribute that
	// became Required or Optional here would be a behaviour change worth
	// catching, not a refinement.
	for name, attr := range s.Attributes {
		if name == "timeouts" {
			continue
		}
		if attr.IsRequired() || attr.IsOptional() {
			t.Errorf("attribute %q must be computed-only, got required=%v optional=%v",
				name, attr.IsRequired(), attr.IsOptional())
		}
		if !attr.IsComputed() {
			t.Errorf("attribute %q must be computed", name)
		}
	}
}

// TestTenantIDDataSource_DescriptionIsUIAligned pins STYLE_GUIDE §User-facing
// descriptions are UI-aligned, not wire-aligned. The tenant identifier has no
// admin-UI screen of its own, which makes protocol vocabulary an easy thing to
// reach for when describing it.
func TestTenantIDDataSource_DescriptionIsUIAligned(t *testing.T) {
	d := NewTenantIDDataSource()
	var resp datasource.SchemaResponse
	d.(*TenantIDDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	// Only the description this package authors is in scope. The privileges
	// section is rendered by internal/common/permissions and shared verbatim
	// across every construct, so its wording is not this test's business.
	desc := strings.TrimSuffix(resp.Schema.MarkdownDescription, dataSourcePrivileges)
	if desc == resp.Schema.MarkdownDescription {
		t.Fatal("privileges section not found at the end of the description; the trim below is not doing anything")
	}

	for _, banned := range []string{"endpoint", "/v1/", "csa", "HTTP", "SDK"} {
		if strings.Contains(strings.ToLower(desc), strings.ToLower(banned)) {
			t.Errorf("description contains wire vocabulary %q:\n%s", banned, desc)
		}
	}
}
