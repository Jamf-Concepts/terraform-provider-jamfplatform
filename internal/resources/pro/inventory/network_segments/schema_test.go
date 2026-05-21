// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package network_segments

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestNetworkSegmentsDataSource_Metadata(t *testing.T) {
	d := NewNetworkSegmentsDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*NetworkSegmentsDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_pro_network_segments" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_pro_network_segments", resp.TypeName)
	}
}

func TestNetworkSegmentsDataSource_Schema(t *testing.T) {
	d := NewNetworkSegmentsDataSource()
	var resp datasource.SchemaResponse
	d.(*NetworkSegmentsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "timeouts", "filter", "network_segments"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	nsAttr, ok := s.Attributes["network_segments"]
	if !ok {
		t.Fatal("missing network_segments attribute")
	}
	nested, ok := nsAttr.(datasourceschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("network_segments should be ListNestedAttribute, got %T", nsAttr)
	}
	for _, name := range []string{"id", "name", "starting_address", "ending_address"} {
		if _, ok := nested.NestedObject.Attributes[name]; !ok {
			t.Errorf("network_segments nested missing attribute %q", name)
		}
	}
}
