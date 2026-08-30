// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_app

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestZtnaAppsDataSource_Metadata(t *testing.T) {
	d := NewZtnaAppsDataSource()
	var resp datasource.MetadataResponse
	d.(*ZtnaAppsDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)

	if resp.TypeName != "jamfplatform_security_cloud_ztna_apps" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_security_cloud_ztna_apps", resp.TypeName)
	}
}

// TestZtnaAppsDataSource_Schema pins that the plural result carries the same fields
// as the singular data source, so a caller can filter the list in Terraform and get
// everything a single lookup would have given them.
func TestZtnaAppsDataSource_Schema(t *testing.T) {
	d := NewZtnaAppsDataSource()
	var resp datasource.SchemaResponse
	d.(*ZtnaAppsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "ztna_apps", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	apps, ok := s.Attributes["ztna_apps"].(dsschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("ztna_apps must be a ListNestedAttribute, got %T", s.Attributes["ztna_apps"])
	}
	if !s.Attributes["ztna_apps"].IsComputed() {
		t.Error("ztna_apps must be computed")
	}

	want := []string{
		"id", "name", "predefined_app_id", "app_type", "category", "hostnames",
		"direct_ips_and_subnets", "all_device_groups", "device_group_ids", "routing",
		"routing_overrides", "security",
	}
	for _, name := range want {
		if _, present := apps.NestedObject.Attributes[name]; !present {
			t.Errorf("ztna_apps missing nested attribute %q", name)
		}
	}
}
