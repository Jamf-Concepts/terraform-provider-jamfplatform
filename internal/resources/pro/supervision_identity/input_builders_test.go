// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package supervision_identity

import (
	"encoding/base64"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

// TestHasCertificateData covers the generate-vs-upload discriminator.
func TestHasCertificateData(t *testing.T) {
	cases := []struct {
		name string
		val  types.String
		want bool
	}{
		{"null", types.StringNull(), false},
		{"unknown", types.StringUnknown(), false},
		{"empty", types.StringValue(""), false},
		{"whitespace", types.StringValue("  \n"), false},
		{"set", types.StringValue(b64("p12")), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := SupervisionIdentityResourceModel{CertificateData: c.val}
			if got := hasCertificateData(cfg); got != c.want {
				t.Errorf("hasCertificateData(%q) = %v, want %v", c.val, got, c.want)
			}
		})
	}
}

// TestBuildGenerateInput verifies the generate payload takes display_name from
// the plan, the password from config, and trims whitespace from the password.
func TestBuildGenerateInput(t *testing.T) {
	plan := SupervisionIdentityResourceModel{DisplayName: types.StringValue("Configurator")}
	cfg := SupervisionIdentityResourceModel{Password: types.StringValue("  secret-pw\n")}

	out := buildGenerateInput(plan, cfg)
	if out.DisplayName != "Configurator" {
		t.Errorf("DisplayName = %q", out.DisplayName)
	}
	if out.Password != "secret-pw" {
		t.Errorf("Password = %q, want trimmed %q", out.Password, "secret-pw")
	}
}

// TestBuildUploadInput verifies the upload payload base64-decodes the certificate
// into raw bytes, takes the password from config (trimmed), and the display name
// from the plan.
func TestBuildUploadInput(t *testing.T) {
	plan := SupervisionIdentityResourceModel{DisplayName: types.StringValue("Imported")}
	cfg := SupervisionIdentityResourceModel{
		Password:        types.StringValue("pw-123 "),
		CertificateData: types.StringValue(b64("raw-p12-bytes")),
	}

	out, err := buildUploadInput(plan, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.DisplayName != "Imported" {
		t.Errorf("DisplayName = %q", out.DisplayName)
	}
	if out.Password != "pw-123" {
		t.Errorf("Password = %q, want trimmed", out.Password)
	}
	if out.CertificateData == nil || string(*out.CertificateData) != "raw-p12-bytes" {
		t.Errorf("CertificateData not base64-decoded correctly: %v", out.CertificateData)
	}
}

// TestBuildUploadInput_BadBase64 verifies invalid certificate_data errors.
func TestBuildUploadInput_BadBase64(t *testing.T) {
	cfg := SupervisionIdentityResourceModel{
		Password:        types.StringValue("p"),
		CertificateData: types.StringValue("not!valid!base64!"),
	}
	if _, err := buildUploadInput(SupervisionIdentityResourceModel{}, cfg); err == nil {
		t.Errorf("expected error for invalid base64, got nil")
	}
}

// TestBuildUpdateInput verifies the rename payload carries only the display name.
func TestBuildUpdateInput(t *testing.T) {
	plan := SupervisionIdentityResourceModel{DisplayName: types.StringValue("Renamed")}
	out := buildUpdateInput(plan)
	if out.DisplayName != "Renamed" {
		t.Errorf("DisplayName = %q", out.DisplayName)
	}
}
