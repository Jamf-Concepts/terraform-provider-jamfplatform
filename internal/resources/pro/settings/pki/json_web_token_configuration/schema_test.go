// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package json_web_token_configuration

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

const wantTypeName = "jamfplatform_pro_pki_json_web_token_configuration"

func TestJSONWebTokenConfigurationResource_Metadata(t *testing.T) {
	r := NewJSONWebTokenConfigurationResource()
	var resp resource.MetadataResponse
	r.(*JSONWebTokenConfigurationResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != wantTypeName {
		t.Errorf("expected type name %q, got %q", wantTypeName, resp.TypeName)
	}
}

func TestJSONWebTokenConfigurationResource_Schema(t *testing.T) {
	r := NewJSONWebTokenConfigurationResource()
	var resp resource.SchemaResponse
	r.(*JSONWebTokenConfigurationResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	// Flat envelope — no <general> wrapper.
	if _, ok := s.Attributes["general"]; ok {
		t.Errorf("schema must be flat: must NOT expose a 'general' block")
	}

	// name + encryption_key_wo are Required.
	for _, name := range []string{"name", "encryption_key_wo"} {
		a, ok := s.Attributes[name]
		if !ok {
			t.Errorf("missing required attribute %q", name)
			continue
		}
		if !a.IsRequired() {
			t.Errorf("%s must be required", name)
		}
	}

	// token_expiry + enabled are Optional+Computed (omit = server value).
	for _, name := range []string{"token_expiry", "enabled"} {
		a, ok := s.Attributes[name]
		if !ok {
			t.Errorf("missing attribute %q", name)
			continue
		}
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%s must be optional+computed", name)
		}
	}

	// encryption_key_wo_version is Optional-only (the rotation trigger).
	if v, ok := s.Attributes["encryption_key_wo_version"]; !ok {
		t.Errorf("missing encryption_key_wo_version")
	} else if !v.IsOptional() || v.IsComputed() {
		t.Errorf("encryption_key_wo_version must be optional-only (not computed)")
	}

	// id computed-only.
	if id := s.Attributes["id"]; id.IsRequired() || id.IsOptional() || !id.IsComputed() {
		t.Errorf("id must be computed-only")
	}

	// encryption_key_wo is WriteOnly + Sensitive — the plaintext must never
	// reach state.
	key, ok := s.Attributes["encryption_key_wo"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("encryption_key_wo must be a StringAttribute")
	}
	if !key.WriteOnly || !key.Sensitive {
		t.Errorf("encryption_key_wo must be WriteOnly + Sensitive, got writeonly=%v sensitive=%v", key.WriteOnly, key.Sensitive)
	}
}

func TestJSONWebTokenConfigurationDataSource_Metadata(t *testing.T) {
	d := NewJSONWebTokenConfigurationDataSource()
	var resp datasource.MetadataResponse
	d.(*JSONWebTokenConfigurationDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != wantTypeName {
		t.Errorf("expected type name %q, got %q", wantTypeName, resp.TypeName)
	}
}

func TestJSONWebTokenConfigurationDataSource_Schema(t *testing.T) {
	d := NewJSONWebTokenConfigurationDataSource()
	var resp datasource.SchemaResponse
	d.(*JSONWebTokenConfigurationDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	// The encryption key must never be surfaced by the data source.
	for _, name := range []string{"encryption_key_wo", "encryption_key", "encryption_key_wo_version"} {
		if _, ok := resp.Schema.Attributes[name]; ok {
			t.Errorf("data source must not expose %q", name)
		}
	}
}

func TestJSONWebTokenConfigurationDataSource_ConfigValidators(t *testing.T) {
	d := NewJSONWebTokenConfigurationDataSource().(*JSONWebTokenConfigurationDataSource)
	if got := d.ConfigValidators(context.Background()); len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestJSONWebTokenConfigurationListResource_Schema(t *testing.T) {
	r := NewJSONWebTokenConfigurationListResource()
	var resp list.ListResourceSchemaResponse
	r.(*JSONWebTokenConfigurationListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
