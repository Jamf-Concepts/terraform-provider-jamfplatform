// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_grouped_gateway

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestGroupedGatewaysDataSource_Metadata(t *testing.T) {
	d := NewGroupedGatewaysDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*GroupedGatewaysDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_security_cloud_ztna_grouped_gateways" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_security_cloud_ztna_grouped_gateways", resp.TypeName)
	}
}

func TestGroupedGatewaysDataSource_Schema(t *testing.T) {
	d := NewGroupedGatewaysDataSource()
	var resp datasource.SchemaResponse
	d.(*GroupedGatewaysDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "grouped_gateways", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	groups, ok := s.Attributes["grouped_gateways"].(dsschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("grouped_gateways must be a ListNestedAttribute, got %T", s.Attributes["grouped_gateways"])
	}
	for _, name := range []string{"id", "name", "gateway_ids", "routing_strategy", "required_gateway_stability", "tenant_ids", "created_at"} {
		if _, present := groups.NestedObject.Attributes[name]; !present {
			t.Errorf("grouped_gateways missing nested attribute %q", name)
		}
	}
	if !s.Attributes["grouped_gateways"].IsComputed() {
		t.Error("grouped_gateways must be computed")
	}
}
