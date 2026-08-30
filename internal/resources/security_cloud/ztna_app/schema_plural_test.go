// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_app

import (
	"context"
	"strings"
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

// TestZtnaAppDataSources_EnumDescriptions pins that every read-only enum-valued
// attribute on both data sources documents its accepted spellings, generated from the
// same slice the resource's OneOf validators use. The spellings are the provider's
// own admin-UI labels, so an operator comparing against a guessed value in a `for`
// expression gets a silently empty result rather than an error.
func TestZtnaAppDataSources_EnumDescriptions(t *testing.T) {
	singular := ztnaAppDataSourceSchema(t)
	plural := ztnaAppsResultAttributes(t)

	routing := map[string][]string{
		"traffic_routing": routingModeValues(),
		"routing_mode":    dnsResolutionValues(),
	}
	for name, values := range routing {
		attr, ok := dsRoutingSchemaAttributes()[name]
		if !ok {
			t.Fatalf("routing block missing %q", name)
		}
		assertDocumentsValues(t, "routing."+name, attr.GetMarkdownDescription(), values)
	}

	for _, surface := range []struct {
		label string
		attrs map[string]dsschema.Attribute
	}{
		{"data source", singular},
		{"plural result", plural},
	} {
		assertDocumentsValues(t, surface.label+" app_type", surface.attrs["app_type"].GetMarkdownDescription(), appTypeValues())

		security, ok := surface.attrs["security"].(dsschema.SingleNestedAttribute)
		if !ok {
			t.Fatalf("%s security must be a SingleNestedAttribute, got %T", surface.label, surface.attrs["security"])
		}
		risk, ok := security.Attributes["device_risk"].(dsschema.SingleNestedAttribute)
		if !ok {
			t.Fatalf("%s security.device_risk must be a SingleNestedAttribute, got %T", surface.label, security.Attributes["device_risk"])
		}
		assertDocumentsValues(t, surface.label+" deny_at_risk_level", risk.Attributes["deny_at_risk_level"].GetMarkdownDescription(), riskLevelValues())
	}
}

// TestZtnaAppDataSources_HostnamesDescriptionMatches pins that the plural result
// keeps the singular data source's caveat about a predefined application's host
// names, which is the surprising half and the one operators iterate over.
func TestZtnaAppDataSources_HostnamesDescriptionMatches(t *testing.T) {
	singular := ztnaAppDataSourceSchema(t)["hostnames"].GetMarkdownDescription()
	plural := ztnaAppsResultAttributes(t)["hostnames"].GetMarkdownDescription()
	if singular != plural {
		t.Errorf("hostnames description differs\nsingular: %s\nplural:   %s", singular, plural)
	}
}

// assertDocumentsValues fails when a description omits any accepted spelling.
func assertDocumentsValues(t *testing.T, name, description string, values []string) {
	t.Helper()
	if len(values) == 0 {
		t.Fatalf("%s has no accepted values to document", name)
	}
	if !strings.Contains(description, markdownList(values)) {
		t.Errorf("%s description does not document its accepted values %v: %s", name, values, description)
	}
}

// ztnaAppDataSourceSchema returns the singular data source's top-level attributes.
func ztnaAppDataSourceSchema(t *testing.T) map[string]dsschema.Attribute {
	t.Helper()
	d := NewZtnaAppDataSource()
	var resp datasource.SchemaResponse
	d.(*ZtnaAppDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema.Attributes
}

// ztnaAppsResultAttributes returns the attributes of one plural result element.
func ztnaAppsResultAttributes(t *testing.T) map[string]dsschema.Attribute {
	t.Helper()
	d := NewZtnaAppsDataSource()
	var resp datasource.SchemaResponse
	d.(*ZtnaAppsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	apps, ok := resp.Schema.Attributes["ztna_apps"].(dsschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("ztna_apps must be a ListNestedAttribute, got %T", resp.Schema.Attributes["ztna_apps"])
	}
	return apps.NestedObject.Attributes
}
