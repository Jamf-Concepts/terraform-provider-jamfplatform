// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// registrationProviderTypeName mirrors the value the provider's own Metadata sets,
// so every type name this test derives is the one Terraform actually sees.
const registrationProviderTypeName = "jamfplatform"

// registeredResourceTypeNames returns the type name of every constructor in
// Resources(), in registration order.
func registeredResourceTypeNames(t *testing.T) []string {
	t.Helper()
	ctx := context.Background()
	p := New("test")()

	names := make([]string, 0)
	for _, newResource := range p.Resources(ctx) {
		var resp resource.MetadataResponse
		newResource().Metadata(ctx, resource.MetadataRequest{ProviderTypeName: registrationProviderTypeName}, &resp)
		names = append(names, resp.TypeName)
	}
	return names
}

// registeredDataSourceTypeNames returns the type name of every constructor in
// DataSources(), in registration order.
func registeredDataSourceTypeNames(t *testing.T) []string {
	t.Helper()
	ctx := context.Background()
	p := New("test")()

	names := make([]string, 0)
	for _, newDataSource := range p.DataSources(ctx) {
		var resp datasource.MetadataResponse
		newDataSource().Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: registrationProviderTypeName}, &resp)
		names = append(names, resp.TypeName)
	}
	return names
}

// TestRegisteredTypeNamesAreNonEmptyAndUnique walks the provider's registration
// slices rather than a construct's Metadata in isolation.
//
// A per-construct Metadata test builds the resource directly, so it says nothing
// about whether the provider registers it. An empty type name would make the
// construct unaddressable, and a duplicate silently shadows one of the two — the
// framework resolves by name, so the loser is simply unreachable. Neither is
// visible to any per-package test.
//
// No total count is asserted: a hardcoded number would break on every legitimate
// addition and teach reviewers to bump it without reading.
func TestRegisteredTypeNamesAreNonEmptyAndUnique(t *testing.T) {
	for _, kind := range []struct {
		label string
		names []string
	}{
		{label: "resource", names: registeredResourceTypeNames(t)},
		{label: "data source", names: registeredDataSourceTypeNames(t)},
	} {
		if len(kind.names) == 0 {
			t.Fatalf("the provider registers no %ss at all", kind.label)
		}
		seen := make(map[string]int, len(kind.names))
		for i, name := range kind.names {
			if name == "" {
				t.Errorf("%s at registration index %d reports an empty type name", kind.label, i)
				continue
			}
			if name == registrationProviderTypeName {
				t.Errorf("%s at registration index %d reports the bare provider type name %q, so its Metadata never appended a suffix", kind.label, i, name)
				continue
			}
			if first, dup := seen[name]; dup {
				t.Errorf("%s type name %q is registered twice, at indexes %d and %d; the framework resolves by name, so one of them is unreachable", kind.label, name, first, i)
				continue
			}
			seen[name] = i
		}
	}
}

// TestSecurityCloudConstructsAreRegistered pins the registration of the four Jamf
// Security Cloud constructs added alongside this test.
//
// Dropping a constructor from Resources() or DataSources() still COMPILES whenever
// the package import is kept alive by a sibling line — both DNS singletons register
// a resource and a data source from the same package, so removing either one leaves
// the import used and every existing test green. This is the only assertion that
// catches it.
func TestSecurityCloudConstructsAreRegistered(t *testing.T) {
	resources := registeredResourceTypeNames(t)
	dataSources := registeredDataSourceTypeNames(t)

	tests := []struct {
		typeName     string
		isResource   bool
		isDataSource bool
	}{
		{typeName: "jamfplatform_security_cloud_content_categories", isDataSource: true},
		{typeName: "jamfplatform_security_cloud_ztna_predefined_apps", isDataSource: true},
		{typeName: "jamfplatform_security_cloud_dns_search_domain", isResource: true, isDataSource: true},
		{typeName: "jamfplatform_security_cloud_dns_hostname_mappings", isResource: true, isDataSource: true},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			if got := slices.Contains(resources, tt.typeName); got != tt.isResource {
				t.Errorf("registered as a resource = %v, want %v", got, tt.isResource)
			}
			if got := slices.Contains(dataSources, tt.typeName); got != tt.isDataSource {
				t.Errorf("registered as a data source = %v, want %v", got, tt.isDataSource)
			}
		})
	}
}
