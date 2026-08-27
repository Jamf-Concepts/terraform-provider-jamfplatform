// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package dns_zone

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestDNSZonesDataSource_Metadata(t *testing.T) {
	d := NewDNSZonesDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*DNSZonesDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_security_cloud_dns_zones" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_security_cloud_dns_zones", resp.TypeName)
	}
}

func TestDNSZonesDataSource_Schema(t *testing.T) {
	d := NewDNSZonesDataSource()
	var resp datasource.SchemaResponse
	d.(*DNSZonesDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "dns_zones", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	zones, ok := s.Attributes["dns_zones"].(dsschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("dns_zones must be a ListNestedAttribute, got %T", s.Attributes["dns_zones"])
	}
	for _, name := range []string{"id", "name", "domains", "name_servers"} {
		if _, present := zones.NestedObject.Attributes[name]; !present {
			t.Errorf("dns_zones missing nested attribute %q", name)
		}
	}
	if !s.Attributes["dns_zones"].IsComputed() {
		t.Errorf("dns_zones must be computed")
	}
}
