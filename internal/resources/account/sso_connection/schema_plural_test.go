// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_connection

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

// pluralResultAttributeNames is every attribute one entry of the plural data
// source declares.
//
// The set is Jamf's rather than this provider's choice: the collection read
// returns no per-provider settings at all, so anything more here would mean one
// extra read per connection in the organization.
var pluralResultAttributeNames = []string{
	"id",
	"name",
	"connection_type",
	"hosting_region",
	"auth_method",
	"sync_attributes_at_login",
	"domains",
	"enabled_product_names",
	"ticket_url",
	"easy_config",
}

// pluralDataSourceSchema returns the plural data source schema.
func pluralDataSourceSchema(t *testing.T) dsschema.Schema {
	t.Helper()
	var resp datasource.SchemaResponse
	(&ConnectionsDataSource{}).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func TestConnectionsDataSource_Metadata(t *testing.T) {
	var resp datasource.MetadataResponse
	(&ConnectionsDataSource{}).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)

	if resp.TypeName != "jamfplatform_account_sso_connections" {
		t.Errorf("type name = %q, want the plural form", resp.TypeName)
	}
}

func TestConnectionsDataSource_Schema(t *testing.T) {
	s := pluralDataSourceSchema(t)

	for _, name := range []string{"id", "sso_connections", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if len(s.Attributes) != 3 {
		t.Errorf("the schema declares %d attributes, want three", len(s.Attributes))
	}
}

// TestConnectionsDataSource_TakesNoArguments pins that the plural read has
// nothing to filter on. Jamf exposes no search arguments for connections, so
// offering one would be offering something this provider would have to implement
// by reading everything and discarding most of it — which the practitioner can do
// themselves, visibly.
func TestConnectionsDataSource_TakesNoArguments(t *testing.T) {
	s := pluralDataSourceSchema(t)

	for name, attribute := range s.Attributes {
		if name == "timeouts" {
			continue
		}
		if attribute.IsRequired() || attribute.IsOptional() {
			t.Errorf("%q is settable; the plural read takes no arguments", name)
		}
	}
}

// TestConnectionsDataSource_ResultAttributes pins the entry shape, so an
// attribute added or renamed shows up here rather than in someone's broken
// configuration.
func TestConnectionsDataSource_ResultAttributes(t *testing.T) {
	s := pluralDataSourceSchema(t)

	results, ok := s.Attributes["sso_connections"].(dsschema.ListNestedAttribute)
	if !ok {
		t.Fatal("sso_connections is not a list of nested objects")
	}

	for _, name := range pluralResultAttributeNames {
		if _, ok := results.NestedObject.Attributes[name]; !ok {
			t.Errorf("an entry is missing %q", name)
		}
	}
	if len(results.NestedObject.Attributes) != len(pluralResultAttributeNames) {
		t.Errorf("an entry declares %d attributes and this test knows about %d",
			len(results.NestedObject.Attributes), len(pluralResultAttributeNames))
	}
}

// TestConnectionsDataSource_CarriesNoPerProviderSettings pins the deliberate
// omission, and the reason: reporting them would cost one read per connection.
func TestConnectionsDataSource_CarriesNoPerProviderSettings(t *testing.T) {
	s := pluralDataSourceSchema(t)

	results := s.Attributes["sso_connections"].(dsschema.ListNestedAttribute)
	for _, name := range append(settingsBlocks, "group_name_filter", "client_id", "attribute_map") {
		if _, ok := results.NestedObject.Attributes[name]; ok {
			t.Errorf("an entry declares %q, which the collection read does not supply", name)
		}
	}
	if !strings.Contains(s.MarkdownDescription, "singular `jamfplatform_account_sso_connection` data source") {
		t.Errorf("the description does not point at the singular data source for the settings:\n%s", s.MarkdownDescription)
	}
}

// TestConnectionsDataSource_DescriptionExplainsTheStoredName pins the one thing
// this data source is uniquely useful for: finding the name Jamf actually holds,
// which may be a uniquified form of the one a connection was created with.
func TestConnectionsDataSource_DescriptionExplainsTheStoredName(t *testing.T) {
	s := pluralDataSourceSchema(t)

	if !strings.Contains(s.MarkdownDescription, "uniquified") {
		t.Errorf("the description does not explain the stored name:\n%s", s.MarkdownDescription)
	}
}
