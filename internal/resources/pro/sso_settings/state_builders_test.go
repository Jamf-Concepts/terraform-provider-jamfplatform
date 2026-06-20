// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package sso_settings

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestSerialNumberToState_BigInt ensures the *json.Number SDK field surfaces
// full-precision 157-bit X.509 serials losslessly via .String().
func TestSerialNumberToState_BigInt(t *testing.T) {
	bigSerial := json.Number("139740726707269723692607826204984509091849452814")
	got := serialNumberToState(&bigSerial)
	if got.IsNull() {
		t.Fatal("expected non-null serial number")
	}
	if got.ValueString() != bigSerial.String() {
		t.Errorf("expected %q, got %q", bigSerial.String(), got.ValueString())
	}
}

// TestSerialNumberToState_Nil ensures nil maps to null state.
func TestSerialNumberToState_Nil(t *testing.T) {
	if !serialNumberToState((*json.Number)(nil)).IsNull() {
		t.Error("expected null state for nil serial")
	}
}

// TestAssignSamlSettingsModel_PreservesUserBase64 confirms the user's
// authored federation_metadata_file string flows through verbatim from
// prev (plan) to state, regardless of how the wire bytes encode. This
// matches the Optional-only schema shape: state mirrors the user's input
// directly so canonical re-encoding cannot trip Terraform Core's plan-vs-
// apply consistency guarantee on a Sensitive nested attribute.
func TestAssignSamlSettingsModel_PreservesUserBase64(t *testing.T) {
	userBase64 := "PEVudGl0eURlc2NyaXB0b3IgLz4="
	prev := &samlSettingsModel{
		FederationMetadataFile: types.StringValue(userBase64),
	}
	rawXML := []byte("<EntityDescriptor />")
	wire := &pro.SamlSettings{
		FederationMetadataFile: &rawXML,
	}
	out := assignSamlSettingsModel(prev, wire)
	if out == nil {
		t.Fatal("expected non-nil model")
	}
	if got := out.FederationMetadataFile.ValueString(); got != userBase64 {
		t.Errorf("federation_metadata_file = %q, want %q (verbatim user value)", got, userBase64)
	}
}

// TestAssignSsoSettingsResourceModel_OmitsInactiveBranch verifies that in
// pure SAML mode the OIDC stub injection does not surface as a populated
// state block when the user did not author oidc_settings.
func TestAssignSsoSettingsResourceModel_OmitsInactiveBranch(t *testing.T) {
	wire := &pro.SsoSettingsV3{
		ConfigurationType: configurationTypeSAML,
		OidcSettings: pro.OidcSettings{
			UserMapping: "EMAIL", // the stub the input builder injects
		},
		SamlSettings: pro.SamlSettings{},
	}
	state := &SsoSettingsResourceModel{}
	if d := assignSsoSettingsResourceModel(context.Background(), state, wire); d.HasError() {
		t.Fatalf("assigner returned errors: %v", d)
	}
	if state.OidcSettings != nil {
		t.Error("OIDC sub-block must remain nil when user did not author it in pure SAML mode")
	}
	if state.SamlSettings == nil {
		t.Error("SAML sub-block should be populated in SAML mode")
	}
}

// TestAssignSsoSettingsResourceModel_PopulatesAuthoredBranch verifies that
// when the user authored oidc_settings, server-echoed fields land in state.
func TestAssignSsoSettingsResourceModel_PopulatesAuthoredBranch(t *testing.T) {
	wire := &pro.SsoSettingsV3{
		ConfigurationType: configurationTypeOIDC,
		OidcSettings: pro.OidcSettings{
			UserMapping: "EMAIL",
		},
	}
	state := &SsoSettingsResourceModel{
		OidcSettings: &oidcSettingsModel{UserMapping: types.StringValue("EMAIL")},
	}
	if d := assignSsoSettingsResourceModel(context.Background(), state, wire); d.HasError() {
		t.Fatalf("assigner returned errors: %v", d)
	}
	if state.OidcSettings == nil {
		t.Fatal("OIDC sub-block must be populated when user authored it")
	}
	if state.OidcSettings.UserMapping.ValueString() != "EMAIL" {
		t.Errorf("user_mapping = %q, want EMAIL", state.OidcSettings.UserMapping.ValueString())
	}
}
