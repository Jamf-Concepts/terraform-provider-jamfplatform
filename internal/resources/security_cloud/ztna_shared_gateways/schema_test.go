// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_shared_gateways

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestSharedGatewaysDataSource_Metadata(t *testing.T) {
	d := NewSharedGatewaysDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*SharedGatewaysDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_security_cloud_ztna_shared_gateways" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_security_cloud_ztna_shared_gateways", resp.TypeName)
	}
}

func TestSharedGatewaysDataSource_Schema(t *testing.T) {
	d := NewSharedGatewaysDataSource()
	var resp datasource.SchemaResponse
	d.(*SharedGatewaysDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "shared_gateways", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	gateways, ok := s.Attributes["shared_gateways"].(dsschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("shared_gateways must be a ListNestedAttribute, got %T", s.Attributes["shared_gateways"])
	}
	for _, name := range []string{"id", "name"} {
		if _, present := gateways.NestedObject.Attributes[name]; !present {
			t.Errorf("shared_gateways missing nested attribute %q", name)
		}
	}
}

// TestSharedGatewaysDataSource_IsEntirelyReadOnly pins the shape of a catalogue
// nobody manages: every attribute is Computed, and there is no argument to filter
// or select by because the endpoint accepts none.
func TestSharedGatewaysDataSource_IsEntirelyReadOnly(t *testing.T) {
	d := NewSharedGatewaysDataSource()
	var resp datasource.SchemaResponse
	d.(*SharedGatewaysDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	for name, attr := range resp.Schema.Attributes {
		if name == "timeouts" {
			continue
		}
		if attr.IsRequired() || attr.IsOptional() {
			t.Errorf("%s must be computed-only; the catalogue takes no arguments", name)
		}
	}
}
