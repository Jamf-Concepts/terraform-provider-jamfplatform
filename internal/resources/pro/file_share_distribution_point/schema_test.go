// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package file_share_distribution_point

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestResource_Metadata(t *testing.T) {
	r := NewFileShareDistributionPointResource()
	var resp resource.MetadataResponse
	r.(*FileShareDistributionPointResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != "jamfplatform_pro_file_share_distribution_point" {
		t.Errorf("got type name %q", resp.TypeName)
	}
}

func TestResource_Schema(t *testing.T) {
	r := NewFileShareDistributionPointResource()
	var resp resource.SchemaResponse
	r.(*FileShareDistributionPointResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	want := []string{
		"id", "name", "server_name", "file_sharing_connection_type",
		"principal", "backup_distribution_point_id", "enable_load_balancing",
		"share_name", "port", "workgroup",
		"read_write_username", "read_write_password", "read_write_password_wo_version",
		"read_only_username", "read_only_password", "read_only_password_wo_version",
		"https_enabled", "https_port", "https_context", "https_security_type",
		"https_username", "https_password", "https_password_wo_version",
		"timeouts",
	}
	for _, n := range want {
		if _, ok := s.Attributes[n]; !ok {
			t.Errorf("missing attribute %q", n)
		}
	}

	// ssh_* and local_path_to_share are intentionally NOT exposed — not in the
	// admin UI (maintainer decision Q3).
	for _, n := range []string{"ssh_username", "ssh_password", "local_path_to_share"} {
		if _, ok := s.Attributes[n]; ok {
			t.Errorf("attribute %q must NOT be exposed (not in the admin UI)", n)
		}
	}

	for _, n := range []string{"name", "server_name", "file_sharing_connection_type"} {
		if !s.Attributes[n].IsRequired() {
			t.Errorf("%q must be Required", n)
		}
	}

	id := s.Attributes["id"]
	if id.IsRequired() || !id.IsComputed() {
		t.Errorf("id must be Computed-only")
	}

	// The three passwords must be WriteOnly + Sensitive + Optional.
	for _, n := range []string{"read_write_password", "read_only_password", "https_password"} {
		a, ok := s.Attributes[n].(rschema.StringAttribute)
		if !ok {
			t.Errorf("%q must be a StringAttribute", n)
			continue
		}
		if !a.WriteOnly || !a.Sensitive || !a.Optional {
			t.Errorf("%q must be WriteOnly+Sensitive+Optional, got writeOnly=%v sensitive=%v optional=%v", n, a.WriteOnly, a.Sensitive, a.Optional)
		}
		if a.Computed {
			t.Errorf("%q must not be Computed (WriteOnly never echoed)", n)
		}
	}

	// Each *_wo_version is Optional-only Int64.
	for _, n := range []string{"read_write_password_wo_version", "read_only_password_wo_version", "https_password_wo_version"} {
		a := s.Attributes[n]
		if !a.IsOptional() || a.IsComputed() || a.IsRequired() {
			t.Errorf("%q must be Optional-only Int64", n)
		}
	}

	// Server-defaulted / user-string optionals are Optional+Computed (omit=preserve).
	for _, n := range []string{
		"principal", "backup_distribution_point_id", "enable_load_balancing",
		"share_name", "port", "workgroup", "read_write_username", "read_only_username",
		"https_enabled", "https_port", "https_context", "https_security_type", "https_username",
	} {
		a := s.Attributes[n]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%q must be Optional+Computed, got optional=%v computed=%v", n, a.IsOptional(), a.IsComputed())
		}
	}
}

func TestResource_ConfigValidators(t *testing.T) {
	r := NewFileShareDistributionPointResource().(*FileShareDistributionPointResource)
	if got := r.ConfigValidators(context.Background()); len(got) != 2 {
		t.Fatalf("expected 2 config validators, got %d", len(got))
	}
}

func TestDataSource_Schema(t *testing.T) {
	d := NewFileShareDistributionPointDataSource()
	var resp datasource.SchemaResponse
	d.(*FileShareDistributionPointDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	for _, n := range []string{"id", "name", "server_name", "file_sharing_connection_type", "timeouts"} {
		if _, ok := s.Attributes[n]; !ok {
			t.Errorf("missing data source attribute %q", n)
		}
	}
	// No plaintext passwords or wo_versions on the read-only data source.
	for _, n := range []string{
		"read_write_password", "read_only_password", "https_password",
		"read_write_password_wo_version", "read_only_password_wo_version", "https_password_wo_version",
	} {
		if _, ok := s.Attributes[n]; ok {
			t.Errorf("data source must not expose %q", n)
		}
	}
	for _, sel := range []string{"id", "name"} {
		a := s.Attributes[sel]
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%q must be Optional+Computed lookup key", sel)
		}
	}
}

func TestListResource_Schema(t *testing.T) {
	r := NewFileShareDistributionPointListResource()
	var resp list.ListResourceSchemaResponse
	r.(*FileShareDistributionPointListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
