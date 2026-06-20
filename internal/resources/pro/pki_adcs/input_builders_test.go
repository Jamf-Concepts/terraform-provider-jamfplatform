// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pki_adcs

import (
	"encoding/base64"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

func TestConnectorModeMapping(t *testing.T) {
	if connectorModeToOutbound(connectorModeInbound) {
		t.Error("INBOUND must map to outbound=false")
	}
	if !connectorModeToOutbound(connectorModeOutbound) {
		t.Error("OUTBOUND must map to outbound=true")
	}
	if outboundToConnectorMode(false) != connectorModeInbound {
		t.Error("outbound=false must map to INBOUND")
	}
	if outboundToConnectorMode(true) != connectorModeOutbound {
		t.Error("outbound=true must map to OUTBOUND")
	}
}

func TestBuildCreateInput_Inbound(t *testing.T) {
	plan := AdcsResourceModel{
		ConnectorMode:     types.StringValue(connectorModeInbound),
		DisplayName:       types.StringValue("tf-adcs-in"),
		CaName:            types.StringValue("ca"),
		Fqdn:              types.StringValue("adcs.example.com"),
		RevocationEnabled: types.BoolValue(false),
		AdcsURL:           types.StringValue("connector.example.com"),
		APIClientID:       types.StringNull(),
		ServerCertificate: &adcsCertInputModel{Filename: types.StringValue("server.pem"), WoVersion: types.Int64Value(1)},
		ClientCertificate: &adcsClientCertInput{Filename: types.StringValue("client.p12"), WoVersion: types.Int64Value(1)},
	}
	cfg := AdcsResourceModel{
		ServerCertificate: &adcsCertInputModel{DataWo: types.StringValue(b64("PEM-BYTES")), WoVersion: types.Int64Value(1)},
		ClientCertificate: &adcsClientCertInput{DataWo: types.StringValue(b64("P12-BYTES")), PasswordWo: types.StringValue("pw"), WoVersion: types.Int64Value(1)},
	}

	out, err := buildAdcsCreateInput(plan, cfg)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.Outbound == nil || *out.Outbound {
		t.Error("INBOUND must send outbound=false")
	}
	if out.AdcsURL == nil || *out.AdcsURL != "connector.example.com" {
		t.Error("adcs_url must be sent for INBOUND")
	}
	if out.ApiClientID != nil {
		t.Error("api_client_id must NOT be sent for INBOUND")
	}
	if out.ServerCert == nil || string(out.ServerCert.Data) != "PEM-BYTES" || out.ServerCert.Filename != "server.pem" {
		t.Errorf("serverCert not built correctly: %+v", out.ServerCert)
	}
	if out.ServerCert.Password != nil {
		t.Error("server certificate must be password-less")
	}
	if out.ClientCert == nil || string(out.ClientCert.Data) != "P12-BYTES" || out.ClientCert.Password == nil || *out.ClientCert.Password != "pw" {
		t.Errorf("clientCert not built correctly: %+v", out.ClientCert)
	}
}

func TestBuildCreateInput_Outbound(t *testing.T) {
	plan := AdcsResourceModel{
		ConnectorMode: types.StringValue(connectorModeOutbound),
		DisplayName:   types.StringValue("tf-adcs-out"),
		APIClientID:   types.StringValue("11111111-2222-3333-4444-555555555555"),
		AdcsURL:       types.StringNull(),
	}
	out, err := buildAdcsCreateInput(plan, AdcsResourceModel{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.Outbound == nil || !*out.Outbound {
		t.Error("OUTBOUND must send outbound=true")
	}
	if out.ApiClientID == nil || *out.ApiClientID != "11111111-2222-3333-4444-555555555555" {
		t.Error("api_client_id must be sent for OUTBOUND")
	}
	if out.AdcsURL != nil || out.ServerCert != nil || out.ClientCert != nil {
		t.Error("OUTBOUND must not send adcs_url / certificates")
	}
}

func TestBuildCreateInput_InvalidBase64(t *testing.T) {
	plan := AdcsResourceModel{
		ConnectorMode:     types.StringValue(connectorModeInbound),
		ServerCertificate: &adcsCertInputModel{WoVersion: types.Int64Value(1)},
	}
	cfg := AdcsResourceModel{
		ServerCertificate: &adcsCertInputModel{DataWo: types.StringValue("not!base64!"), WoVersion: types.Int64Value(1)},
	}
	if _, err := buildAdcsCreateInput(plan, cfg); err == nil {
		t.Fatal("expected base64 decode error")
	}
}

func TestBuildUpdateInput_OmitsOutbound(t *testing.T) {
	plan := AdcsResourceModel{
		ConnectorMode: types.StringValue(connectorModeInbound),
		DisplayName:   types.StringValue("renamed"),
	}
	out, err := buildAdcsUpdateInput(plan, AdcsResourceModel{}, AdcsResourceModel{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.Outbound != nil {
		t.Error("Update must never send outbound (immutable)")
	}
	if out.DisplayName == nil || *out.DisplayName != "renamed" {
		t.Error("declared scalar must be sent on update")
	}
}

func TestBuildUpdateInput_CertRotationGate(t *testing.T) {
	// Same wo_version state↔plan ⇒ certificate omitted (preserved).
	plan := AdcsResourceModel{
		ConnectorMode:     types.StringValue(connectorModeInbound),
		ServerCertificate: &adcsCertInputModel{Filename: types.StringValue("s.pem"), WoVersion: types.Int64Value(1)},
		ClientCertificate: &adcsClientCertInput{Filename: types.StringValue("c.p12"), WoVersion: types.Int64Value(1)},
	}
	state := AdcsResourceModel{
		ServerCertificate: &adcsCertInputModel{WoVersion: types.Int64Value(1)},
		ClientCertificate: &adcsClientCertInput{WoVersion: types.Int64Value(1)},
	}
	cfg := AdcsResourceModel{
		ServerCertificate: &adcsCertInputModel{DataWo: types.StringValue(b64("X")), WoVersion: types.Int64Value(1)},
		ClientCertificate: &adcsClientCertInput{DataWo: types.StringValue(b64("Y")), WoVersion: types.Int64Value(1)},
	}
	out, err := buildAdcsUpdateInput(plan, state, cfg)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.ServerCert != nil || out.ClientCert != nil {
		t.Error("unchanged wo_version must omit certificate (preserve)")
	}

	// Bump server wo_version ⇒ server cert re-sent, client still preserved.
	plan.ServerCertificate.WoVersion = types.Int64Value(2)
	out, err = buildAdcsUpdateInput(plan, state, cfg)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.ServerCert == nil || string(out.ServerCert.Data) != "X" {
		t.Error("bumped server wo_version must re-send server cert")
	}
	if out.ClientCert != nil {
		t.Error("unchanged client wo_version must stay omitted")
	}
}
