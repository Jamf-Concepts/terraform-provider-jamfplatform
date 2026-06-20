// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pki_digicert

import (
	"encoding/base64"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildDigicertInput_ScalarsOmitNullPreserve(t *testing.T) {
	plan := DigicertResourceModel{
		DisplayName:       types.StringValue("My DigiCert"),
		HostName:          types.StringNull(),
		RevocationEnabled: types.BoolValue(true),
	}
	out, err := buildDigicertInput(plan, plan, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.CaName == nil || *out.CaName != "My DigiCert" {
		t.Errorf("caName: got %v, want pointer to %q", out.CaName, "My DigiCert")
	}
	if out.Fqdn != nil {
		t.Errorf("fqdn must be omitted (nil) when null, got %v", *out.Fqdn)
	}
	if out.RevocationEnabled == nil || *out.RevocationEnabled != true {
		t.Errorf("revocationEnabled: got %v, want pointer to true", out.RevocationEnabled)
	}
	if out.ClientCert != nil {
		t.Errorf("clientCert must be omitted when includeCert is false")
	}
}

func TestBuildDigicertInput_CertIncludedDecodesBase64(t *testing.T) {
	raw := []byte("fake-p12-bytes")
	b64 := base64.StdEncoding.EncodeToString(raw)

	cfg := DigicertResourceModel{
		ClientCertificate: &DigicertClientCertModel{
			DataWo:     types.StringValue(b64),
			PasswordWo: types.StringValue("secret"),
		},
	}
	plan := DigicertResourceModel{
		DisplayName: types.StringValue("x"),
		ClientCertificate: &DigicertClientCertModel{
			Filename:  types.StringValue("client.p12"),
			WoVersion: types.Int64Value(1),
		},
	}

	out, err := buildDigicertInput(plan, cfg, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ClientCert == nil {
		t.Fatalf("clientCert must be present when includeCert is true")
	}
	if string(out.ClientCert.Data) != string(raw) {
		t.Errorf("cert data: got %q, want decoded %q", out.ClientCert.Data, raw)
	}
	if out.ClientCert.Filename != "client.p12" {
		t.Errorf("filename: got %q, want %q", out.ClientCert.Filename, "client.p12")
	}
	if out.ClientCert.Password == nil || *out.ClientCert.Password != "secret" {
		t.Errorf("password: got %v, want pointer to %q", out.ClientCert.Password, "secret")
	}
}

func TestBuildDigicertInput_InvalidBase64Errors(t *testing.T) {
	cfg := DigicertResourceModel{
		ClientCertificate: &DigicertClientCertModel{DataWo: types.StringValue("!!!not-base64!!!")},
	}
	plan := DigicertResourceModel{ClientCertificate: &DigicertClientCertModel{}}
	if _, err := buildDigicertInput(plan, cfg, true); err == nil {
		t.Errorf("expected error for invalid base64, got nil")
	}
}

func TestShouldRotateCert(t *testing.T) {
	cases := []struct {
		name  string
		plan  *DigicertClientCertModel
		state *DigicertClientCertModel
		want  bool
	}{
		{"nil plan", nil, &DigicertClientCertModel{WoVersion: types.Int64Value(1)}, false},
		{"nil state (block added)", &DigicertClientCertModel{WoVersion: types.Int64Value(1)}, nil, true},
		{"same version", &DigicertClientCertModel{WoVersion: types.Int64Value(2)}, &DigicertClientCertModel{WoVersion: types.Int64Value(2)}, false},
		{"bumped version", &DigicertClientCertModel{WoVersion: types.Int64Value(3)}, &DigicertClientCertModel{WoVersion: types.Int64Value(2)}, true},
		{"null both", &DigicertClientCertModel{WoVersion: types.Int64Null()}, &DigicertClientCertModel{WoVersion: types.Int64Null()}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldRotateCert(tc.plan, tc.state); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
