// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_gateway

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestGatewaysDataSource_Metadata(t *testing.T) {
	d := NewGatewaysDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*GatewaysDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_security_cloud_ztna_gateways" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_security_cloud_ztna_gateways", resp.TypeName)
	}
}

func TestGatewaysDataSource_Schema(t *testing.T) {
	d := NewGatewaysDataSource()
	var resp datasource.SchemaResponse
	d.(*GatewaysDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "gateways", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	gateways, ok := s.Attributes["gateways"].(dsschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("gateways must be a ListNestedAttribute, got %T", s.Attributes["gateways"])
	}
	for _, name := range []string{"id", "name", "datacenter", "contact", "enabled", "ipsec", "status"} {
		if _, present := gateways.NestedObject.Attributes[name]; !present {
			t.Errorf("gateways missing nested attribute %q", name)
		}
	}
	if !s.Attributes["gateways"].IsComputed() {
		t.Error("gateways must be computed")
	}
}
