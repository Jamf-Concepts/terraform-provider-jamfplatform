// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package webhook

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

const wantTypeName = "jamfplatform_pro_webhook"

func TestWebhookResource_Metadata(t *testing.T) {
	r := NewWebhookResource()
	var resp resource.MetadataResponse
	r.(*WebhookResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != wantTypeName {
		t.Errorf("expected type name %q, got %q", wantTypeName, resp.TypeName)
	}
}

func TestWebhookResource_Schema(t *testing.T) {
	r := NewWebhookResource()
	var resp resource.SchemaResponse
	r.(*WebhookResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	// Flat envelope — no <general> wrapper.
	if _, ok := s.Attributes["general"]; ok {
		t.Errorf("webhook schema must be flat: must NOT expose a 'general' block")
	}

	required := []string{"name", "url", "event"}
	for _, name := range required {
		a, ok := s.Attributes[name]
		if !ok {
			t.Errorf("missing required attribute %q", name)
			continue
		}
		if !a.IsRequired() {
			t.Errorf("%s must be required", name)
		}
	}

	optionalComputed := []string{"enabled", "authentication_type", "connection_timeout", "read_timeout", "content_type", "hash_algorithm", "enable_display_fields_for_group_object"}
	for _, name := range optionalComputed {
		a, ok := s.Attributes[name]
		if !ok {
			t.Errorf("missing attribute %q", name)
			continue
		}
		if !a.IsOptional() || !a.IsComputed() {
			t.Errorf("%s must be optional+computed", name)
		}
	}

	// username / header / smart_group_id are Optional-only (not Computed): they
	// must fall to null when auth/event changes.
	for _, name := range []string{"username", "header", "smart_group_id"} {
		a, ok := s.Attributes[name]
		if !ok {
			t.Errorf("missing attribute %q", name)
			continue
		}
		if !a.IsOptional() || a.IsComputed() {
			t.Errorf("%s must be optional-only (not computed)", name)
		}
	}

	// display_fields is computed-only (the API rejects writes).
	if df, ok := s.Attributes["display_fields"]; !ok {
		t.Errorf("missing display_fields")
	} else if df.IsRequired() || df.IsOptional() || !df.IsComputed() {
		t.Errorf("display_fields must be computed-only")
	}

	// id computed-only.
	if id := s.Attributes["id"]; id.IsRequired() || id.IsOptional() || !id.IsComputed() {
		t.Errorf("id must be computed-only")
	}

	// password is WriteOnly + Sensitive; header is Sensitive (state-tracked).
	pw, ok := s.Attributes["password"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("password must be a StringAttribute")
	}
	if !pw.WriteOnly || !pw.Sensitive {
		t.Errorf("password must be WriteOnly + Sensitive, got writeonly=%v sensitive=%v", pw.WriteOnly, pw.Sensitive)
	}
	hdr, ok := s.Attributes["header"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("header must be a StringAttribute")
	}
	if hdr.WriteOnly {
		t.Errorf("header must NOT be WriteOnly (server echoes it, state-tracked)")
	}
	if !hdr.Sensitive {
		t.Errorf("header must be Sensitive")
	}

	if _, ok := s.Attributes["password_wo_version"]; !ok {
		t.Errorf("missing password_wo_version")
	}
}

// TestWebhookResource_AuthEnumIncludesMTLS pins the wire-probed writable set.
// MTLS is included: the server accepts it on create and update, so a
// faithfully-imported MTLS webhook must validate.
func TestWebhookResource_AuthEnumIncludesMTLS(t *testing.T) {
	want := map[string]bool{"NONE": true, "BASIC": true, "HEADER": true, "HASH_SIGNATURE": true, "MTLS": true}
	if len(webhookAuthTypes) != len(want) {
		t.Fatalf("expected %d auth types, got %d (%v)", len(want), len(webhookAuthTypes), webhookAuthTypes)
	}
	seen := map[string]bool{}
	for _, v := range webhookAuthTypes {
		if !want[v] {
			t.Errorf("unexpected auth type %q", v)
		}
		seen[v] = true
	}
	if !seen["MTLS"] {
		t.Errorf("MTLS must be present in the writable authentication_type enum")
	}
}

func TestWebhookResource_ConfigValidators(t *testing.T) {
	r := NewWebhookResource().(*WebhookResource)
	if got := r.ConfigValidators(context.Background()); len(got) != 6 {
		t.Fatalf("expected 6 config validators, got %d", len(got))
	}
}

func TestWebhookResource_EventEnumCount(t *testing.T) {
	if len(webhookEvents) != 23 {
		t.Errorf("expected 23 events, got %d", len(webhookEvents))
	}
}

func TestWebhookDataSource_Metadata(t *testing.T) {
	d := NewWebhookDataSource()
	var resp datasource.MetadataResponse
	d.(*WebhookDataSource).Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if resp.TypeName != wantTypeName {
		t.Errorf("expected type name %q, got %q", wantTypeName, resp.TypeName)
	}
}

func TestWebhookDataSource_ConfigValidators(t *testing.T) {
	d := NewWebhookDataSource().(*WebhookDataSource)
	if got := d.ConfigValidators(context.Background()); len(got) != 1 {
		t.Fatalf("expected 1 config validator, got %d", len(got))
	}
}

func TestWebhookListResource_Schema(t *testing.T) {
	r := NewWebhookListResource()
	var resp list.ListResourceSchemaResponse
	r.(*WebhookListResource).ListResourceConfigSchema(context.Background(), list.ListResourceSchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	if _, ok := resp.Schema.Attributes["filter"]; !ok {
		t.Errorf("list schema missing filter attribute")
	}
}
