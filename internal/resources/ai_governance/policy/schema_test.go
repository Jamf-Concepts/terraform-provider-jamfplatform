// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// resourceSchema builds and validates the resource schema.
func resourceSchema(t *testing.T) rschema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	NewPolicyResource().(*PolicyResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

// dataSourceSchema builds and validates the singular data source schema.
func dataSourceSchema(t *testing.T) dsschema.Schema {
	t.Helper()
	var resp datasource.SchemaResponse
	NewPolicyDataSource().(*PolicyDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

// pluralDataSourceSchema builds and validates the plural data source schema.
func pluralDataSourceSchema(t *testing.T) dsschema.Schema {
	t.Helper()
	var resp datasource.SchemaResponse
	NewPoliciesDataSource().(*PoliciesDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func TestMetadata(t *testing.T) {
	cases := map[string]func() string{
		"jamfplatform_ai_governance_policy": func() string {
			var resp resource.MetadataResponse
			NewPolicyResource().(*PolicyResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
			return resp.TypeName
		},
		"jamfplatform_ai_governance_policies": func() string {
			var resp datasource.MetadataResponse
			NewPoliciesDataSource().(*PoliciesDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
			return resp.TypeName
		},
	}
	for want, get := range cases {
		if got := get(); got != want {
			t.Errorf("type name = %q, want %q", got, want)
		}
	}

	var dsResp datasource.MetadataResponse
	NewPolicyDataSource().(*PolicyDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &dsResp)
	if dsResp.TypeName != "jamfplatform_ai_governance_policy" {
		t.Errorf("data source type name = %q", dsResp.TypeName)
	}
}

func TestResourceSchemaShape(t *testing.T) {
	s := resourceSchema(t)

	for _, name := range []string{
		"id", "name", "description", "tool_id", "schema_version", "settings_json",
		"publish", "published_version", "has_draft", "schema_drift", "created_at", "updated_at", "timeouts",
	} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	for _, name := range []string{"name", "tool_id", "schema_version", "settings_json"} {
		if !s.Attributes[name].IsRequired() {
			t.Errorf("%s must be required", name)
		}
	}
	for _, name := range []string{"id", "published_version", "has_draft", "schema_drift", "created_at", "updated_at"} {
		attr := s.Attributes[name]
		if attr.IsRequired() || attr.IsOptional() || !attr.IsComputed() {
			t.Errorf("%s must be computed-only", name)
		}
	}
	if !s.Attributes["description"].IsOptional() || s.Attributes["description"].IsComputed() {
		t.Error("description must be optional and not computed")
	}
	publish := s.Attributes["publish"]
	if !publish.IsOptional() || !publish.IsComputed() {
		t.Error("publish must be optional and computed so its default applies")
	}
}

// TestSettingsUsesTheJSONType pins that the settings attribute keeps its semantic-equality type. A
// plain StringAttribute would compile and pass every unit test, then diff on every plan for a
// policy whose stored key order differs from jsonencode's.
func TestSettingsUsesTheJSONType(t *testing.T) {
	attr, ok := resourceSchema(t).Attributes["settings_json"].(rschema.StringAttribute)
	if !ok {
		t.Fatalf("settings_json is %T, want a StringAttribute", resourceSchema(t).Attributes["settings_json"])
	}
	if _, ok := attr.CustomType.(jsonObjectType); !ok {
		t.Errorf("settings_json CustomType = %T, want jsonObjectType", attr.CustomType)
	}
}

// TestToolIDRequiresReplace pins immutability: a policy's tool is fixed once it exists, so changing
// it has to replace the resource rather than attempt an update the platform has no field for.
func TestToolIDRequiresReplace(t *testing.T) {
	attr, ok := resourceSchema(t).Attributes["tool_id"].(rschema.StringAttribute)
	if !ok {
		t.Fatal("tool_id is not a StringAttribute")
	}
	if len(attr.PlanModifiers) == 0 {
		t.Fatal("tool_id has no plan modifiers, so it would not force replacement")
	}
	found := false
	for _, modifier := range attr.PlanModifiers {
		if strings.Contains(modifier.Description(context.Background()), "destroy and recreate") {
			found = true
		}
	}
	if !found {
		t.Error("tool_id must carry RequiresReplace")
	}
}

// TestNoStatusOrVersionAttributes pins the two wire fields deliberately left unmapped: status is
// always ACTIVE for any readable policy, and version is a row-revision counter that changes on
// writes that change nothing.
func TestNoStatusOrVersionAttributes(t *testing.T) {
	for _, schemaUnderTest := range []map[string]rschema.Attribute{resourceSchema(t).Attributes} {
		for _, name := range []string{"status", "version", "created_by", "updated_by"} {
			if _, ok := schemaUnderTest[name]; ok {
				t.Errorf("resource must not expose %q", name)
			}
		}
	}
	for _, name := range []string{"status", "version", "created_by", "updated_by"} {
		if _, ok := dataSourceSchema(t).Attributes[name]; ok {
			t.Errorf("data source must not expose %q", name)
		}
	}
}

// TestDataSourceLooksUpByIDOnly pins that no name lookup exists. Policy names are not unique, so a
// name lookup would resolve arbitrarily.
func TestDataSourceLooksUpByIDOnly(t *testing.T) {
	s := dataSourceSchema(t)
	if !s.Attributes["id"].IsRequired() {
		t.Error("id must be required on the singular data source")
	}
	if s.Attributes["name"].IsOptional() || s.Attributes["name"].IsRequired() {
		t.Error("name must be computed-only — it is not a lookup key")
	}
}

func TestPluralDataSourceSchemaShape(t *testing.T) {
	s := pluralDataSourceSchema(t)
	for _, name := range []string{"sort", "schema_drift_only", "policies"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	if !s.Attributes["policies"].IsComputed() {
		t.Error("policies must be computed")
	}

	nested, ok := s.Attributes["policies"].(dsschema.ListNestedAttribute)
	if !ok {
		t.Fatal("policies is not a ListNestedAttribute")
	}
	if _, present := nested.NestedObject.Attributes["settings_json"]; present {
		t.Error("the listing must not carry settings — the platform's list items omit them")
	}
}

// TestDescriptionsAvoidWireVocabulary pins STYLE_GUIDE §User-facing descriptions are UI-aligned.
// Product framing such as "Jamf rejects" is fine; protocol framing is not.
func TestDescriptionsAvoidWireVocabulary(t *testing.T) {
	banned := []string{"endpoint", "HTTP", " PUT ", " GET ", " POST ", " DELETE ", "/v1/", "SDK"}
	collect := map[string]string{}
	for name, attr := range resourceSchema(t).Attributes {
		collect["resource."+name] = attr.GetMarkdownDescription()
	}
	for name, attr := range dataSourceSchema(t).Attributes {
		collect["data_source."+name] = attr.GetMarkdownDescription()
	}
	for name, attr := range pluralDataSourceSchema(t).Attributes {
		collect["plural."+name] = attr.GetMarkdownDescription()
	}

	for where, description := range collect {
		for _, word := range banned {
			if strings.Contains(description, word) {
				t.Errorf("%s description carries wire vocabulary %q: %s", where, word, description)
			}
		}
	}
}
