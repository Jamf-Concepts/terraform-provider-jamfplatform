// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package smtp_server

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSmtpServerResource_Metadata(t *testing.T) {
	r := NewSmtpServerResource()
	var resp resource.MetadataResponse
	r.(*SmtpServerResource).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "jamfplatform"}, &resp)
	if want := "jamfplatform_pro_smtp_server"; resp.TypeName != want {
		t.Errorf("type name = %q, want %q", resp.TypeName, want)
	}
}

func TestSmtpServerResource_Schema(t *testing.T) {
	r := NewSmtpServerResource()
	var resp resource.SchemaResponse
	r.(*SmtpServerResource).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema

	for _, name := range []string{"id", "enabled", "authentication_type", "sender_settings", "connection_settings", "basic_auth_credentials", "graph_api_credentials", "google_mail_credentials", "timeouts"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("missing top-level attribute %q", name)
		}
	}

	if a := s.Attributes["authentication_type"]; !a.IsRequired() {
		t.Error("authentication_type must be Required")
	}
	if a := s.Attributes["id"]; a.IsRequired() || a.IsOptional() || !a.IsComputed() {
		t.Error("id must be Computed-only")
	}
	if a := s.Attributes["enabled"]; !a.IsOptional() || !a.IsComputed() {
		t.Error("enabled must be Optional+Computed")
	}

	// sender_settings: Required nested block.
	sender, ok := s.Attributes["sender_settings"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("sender_settings must be SingleNestedAttribute")
	}
	if !sender.IsRequired() {
		t.Error("sender_settings must be Required")
	}
	if !sender.Attributes["email_address"].IsRequired() {
		t.Error("sender_settings.email_address must be Required")
	}
	assertEmptySenderValueAccepted(t, sender, "email_address")
	assertEmptySenderValueAccepted(t, sender, "display_name")

	// connection_settings: Optional-only typed-pointer.
	conn, ok := s.Attributes["connection_settings"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("connection_settings must be SingleNestedAttribute")
	}
	if !conn.IsOptional() || conn.IsComputed() {
		t.Error("connection_settings must be Optional-only (not Computed)")
	}
	for _, name := range []string{"host", "port", "encryption_type"} {
		if !conn.Attributes[name].IsRequired() {
			t.Errorf("connection_settings.%s must be Required", name)
		}
	}

	assertWriteOnlySecret(t, s, "basic_auth_credentials", "password", "password_wo_version")
	assertWriteOnlySecret(t, s, "graph_api_credentials", "client_secret", "client_secret_wo_version")
	assertWriteOnlySecret(t, s, "google_mail_credentials", "client_secret", "client_secret_wo_version")

	// google_mail_credentials.authentications: Computed-only list.
	goog := s.Attributes["google_mail_credentials"].(schema.SingleNestedAttribute)
	auths := goog.Attributes["authentications"]
	if auths.IsRequired() || auths.IsOptional() || !auths.IsComputed() {
		t.Error("google_mail_credentials.authentications must be Computed-only")
	}
}

func assertWriteOnlySecret(t *testing.T, s schema.Schema, blockName, secretName, woName string) {
	t.Helper()
	block, ok := s.Attributes[blockName].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("%s must be SingleNestedAttribute", blockName)
	}
	if !block.IsOptional() || block.IsComputed() {
		t.Errorf("%s must be Optional-only (typed-pointer §282)", blockName)
	}
	secret := block.Attributes[secretName]
	if !secret.IsOptional() || !secret.IsSensitive() || !secret.IsWriteOnly() || secret.IsComputed() {
		t.Errorf("%s.%s must be Optional+Sensitive+WriteOnly (never Computed); got optional=%v sensitive=%v writeOnly=%v computed=%v",
			blockName, secretName, secret.IsOptional(), secret.IsSensitive(), secret.IsWriteOnly(), secret.IsComputed())
	}
	wo := block.Attributes[woName]
	if !wo.IsOptional() {
		t.Errorf("%s.%s must be Optional", blockName, woName)
	}
}

func TestSmtpServerDataSource_Schema(t *testing.T) {
	d := NewSmtpServerDataSource()
	var resp datasource.SchemaResponse
	d.(*SmtpServerDataSource).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("data source schema diagnostics: %v", resp.Diagnostics)
	}
	s := resp.Schema
	for _, name := range []string{"id", "enabled", "authentication_type", "sender_settings", "connection_settings", "basic_auth_credentials", "graph_api_credentials", "google_mail_credentials"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("data source missing attribute %q", name)
		}
	}
	// Data source must not expose secrets.
	basic := s.Attributes["basic_auth_credentials"].(dsschema.SingleNestedAttribute)
	if _, leaked := basic.Attributes["password"]; leaked {
		t.Error("data source must not expose basic_auth_credentials.password")
	}
}

// assertEmptySenderValueAccepted proves no attribute-level validator on a
// sender_settings field refuses an empty string.
//
// Jamf Pro stores an empty sender email address and display name on any tenant
// that has never set up mail, and accepts writing them back while the connection
// is disabled, so a minimum-length validator here would make a real tenant state
// impossible to declare and would break configuration generated by
// `terraform plan -generate-config-out`. The prohibition on empty values belongs
// to an enabled connection only, and lives in
// validateSenderSettingsWhenEnabled, which this asserts is the sole enforcement
// point.
func assertEmptySenderValueAccepted(t *testing.T, sender schema.SingleNestedAttribute, name string) {
	t.Helper()

	attr, ok := sender.Attributes[name].(schema.StringAttribute)
	if !ok {
		t.Fatalf("sender_settings.%s must be StringAttribute", name)
	}

	req := validator.StringRequest{
		Path:           path.Root("sender_settings").AtName(name),
		ConfigValue:    types.StringValue(""),
		PathExpression: path.MatchRoot("sender_settings").AtName(name),
	}
	for _, v := range attr.Validators {
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), req, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("sender_settings.%s must accept an empty string, got: %v", name, resp.Diagnostics.Errors())
		}
	}
}
