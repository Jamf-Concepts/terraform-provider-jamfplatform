// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_predefined_apps

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestPredefinedAppsDataSource_Metadata(t *testing.T) {
	d := NewPredefinedAppsDataSource()
	req := datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}
	var resp datasource.MetadataResponse
	d.(*PredefinedAppsDataSource).Metadata(context.Background(), req, &resp)

	if resp.TypeName != "jamfplatform_security_cloud_ztna_predefined_apps" {
		t.Errorf("expected type name %q, got %q", "jamfplatform_security_cloud_ztna_predefined_apps", resp.TypeName)
	}
}

func TestPredefinedAppsDataSource_Schema(t *testing.T) {
	d := NewPredefinedAppsDataSource()
	var resp datasource.SchemaResponse
	d.(*PredefinedAppsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	s := resp.Schema
	for _, name := range []string{"id", "predefined_apps", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}

	apps, ok := s.Attributes["predefined_apps"].(dsschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("predefined_apps must be a ListNestedAttribute, got %T", s.Attributes["predefined_apps"])
	}
	for _, name := range []string{"id", "name", "hostnames"} {
		if _, present := apps.NestedObject.Attributes[name]; !present {
			t.Errorf("predefined_apps missing nested attribute %q", name)
		}
	}

	// STYLE_GUIDE §Sets vs Lists: a Computed nested collection is a types.List. A
	// SetAttribute here fails only under an acceptance apply, which is the worst
	// place to find out.
	if _, ok := apps.NestedObject.Attributes["hostnames"].(dsschema.ListAttribute); !ok {
		t.Errorf("hostnames must be a ListAttribute, got %T", apps.NestedObject.Attributes["hostnames"])
	}
}

// TestPredefinedAppsDataSource_IsEntirelyReadOnly pins the shape of a catalogue
// nobody manages: every attribute is Computed, and there is no argument to filter
// or select by because the endpoint accepts none.
func TestPredefinedAppsDataSource_IsEntirelyReadOnly(t *testing.T) {
	d := NewPredefinedAppsDataSource()
	var resp datasource.SchemaResponse
	d.(*PredefinedAppsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	for name, attr := range resp.Schema.Attributes {
		if name == "timeouts" {
			continue
		}
		if attr.IsRequired() || attr.IsOptional() {
			t.Errorf("%s must be computed-only; the catalogue takes no arguments", name)
		}
	}
}

// TestPredefinedAppsDataSource_NamesItsConsumer pins the honesty of the schema
// description in the other direction. It used to disclaim that the construct
// consuming a template was unbuilt; jamfplatform_security_cloud_ztna_app now ships,
// so the description has to name it rather than send an operator looking for
// nothing — and must not carry the retired disclaimer.
//
// The one-per-definition claim is pinned here too because it is a wire fact nothing
// else guards: a second application built from a definition the tenant already uses
// is refused with `409 CONFLICT "Resource already exists."`, and an operator who
// reads this catalogue and picks a template freely meets that refusal at apply.
func TestPredefinedAppsDataSource_NamesItsConsumer(t *testing.T) {
	d := NewPredefinedAppsDataSource()
	var resp datasource.SchemaResponse
	d.(*PredefinedAppsDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	description := resp.Schema.GetMarkdownDescription()

	if !strings.Contains(description, "jamfplatform_security_cloud_ztna_app") {
		t.Errorf("schema description must name the app resource that consumes a template, got: %s", description)
	}

	if strings.Contains(description, "not yet managed by this provider") {
		t.Errorf("schema description still carries the retired unbuilt-consumer disclaimer, got: %s", description)
	}

	if !strings.Contains(description, "one application per definition") {
		t.Errorf("schema description must say only one application per definition is allowed on a tenant, got: %s", description)
	}
}
