// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package disk_encryption_configuration

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestBuildInput_Individual_NoIRK covers the minimal Individual path:
// the IRK block is omitted in the plan and must not reach the wire.
func TestBuildInput_Individual_NoIRK(t *testing.T) {
	plan := DiskEncryptionConfigurationResourceModel{
		Name:                  types.StringValue("test-individual"),
		KeyType:               types.StringValue(keyTypeIndividual),
		FileVaultEnabledUsers: types.StringValue(fileVaultEnabledUsersCurrentOrNext),
	}
	got := buildDiskEncryptionConfigurationInput(plan)
	if got.Name == nil || *got.Name != "test-individual" {
		t.Errorf("Name did not round-trip, got %v", got.Name)
	}
	if got.KeyType == nil || *got.KeyType != keyTypeIndividual {
		t.Errorf("KeyType did not round-trip, got %v", got.KeyType)
	}
	if got.InstitutionalRecoveryKey != nil {
		t.Errorf("IRK must be nil when plan.InstitutionalRecoveryKey is nil")
	}
}

// TestBuildInput_KeyTypeCanonicalisation verifies the input builder
// rewrites Title-Case `Individual And Institutional` (uppercase `And`) to
// the wire-canonical `Individual and Institutional` (lowercase `and`).
func TestBuildInput_KeyTypeCanonicalisation(t *testing.T) {
	plan := DiskEncryptionConfigurationResourceModel{
		Name:                  types.StringValue("test"),
		KeyType:               types.StringValue("Individual And Institutional"),
		FileVaultEnabledUsers: types.StringValue(fileVaultEnabledUsersCurrentOrNext),
	}
	got := buildDiskEncryptionConfigurationInput(plan)
	if got.KeyType == nil || *got.KeyType != keyTypeIndividualInstitutional {
		t.Errorf("KeyType must canonicalise to wire form %q, got %v", keyTypeIndividualInstitutional, got.KeyType)
	}
}

// TestBuildInput_IRKBlock_PKCS12 covers the PKCS12 path. Both `data` and
// `password` must reach the wire. `key`, `certificate_type`,
// `password_sha256` must NOT — they are server-side and writing them
// back would confuse the server.
func TestBuildInput_IRKBlock_PKCS12(t *testing.T) {
	plan := DiskEncryptionConfigurationResourceModel{
		Name:                  types.StringValue("test-pkcs12"),
		KeyType:               types.StringValue(keyTypeInstitutional),
		FileVaultEnabledUsers: types.StringValue(fileVaultEnabledUsersCurrentOrNext),
		InstitutionalRecoveryKey: &diskEncryptionConfigurationIRKModel{
			Key:             types.StringValue("CN=server-derived"),    // should not reach wire
			CertificateType: types.StringValue("PKCS12"),               // should not reach wire
			PasswordSha256:  types.StringValue("********************"), // should not reach wire
			Password:        types.StringValue("hunter2"),
			Data:            types.StringValue("base64-cert-bytes"),
		},
	}
	got := buildDiskEncryptionConfigurationInput(plan)
	if got.InstitutionalRecoveryKey == nil {
		t.Fatalf("IRK must be non-nil")
	}
	if got.InstitutionalRecoveryKey.Password == nil || *got.InstitutionalRecoveryKey.Password != "hunter2" {
		t.Errorf("Password did not round-trip, got %v", got.InstitutionalRecoveryKey.Password)
	}
	if got.InstitutionalRecoveryKey.Data == nil || *got.InstitutionalRecoveryKey.Data != "base64-cert-bytes" {
		t.Errorf("Data did not round-trip, got %v", got.InstitutionalRecoveryKey.Data)
	}
	if got.InstitutionalRecoveryKey.Key != nil {
		t.Errorf("Key (server-derived) must NOT appear on wire, got %v", got.InstitutionalRecoveryKey.Key)
	}
	if got.InstitutionalRecoveryKey.CertificateType != nil {
		t.Errorf("CertificateType (server-determined) must NOT appear on wire, got %v", got.InstitutionalRecoveryKey.CertificateType)
	}
	if got.InstitutionalRecoveryKey.PasswordSha256 != nil {
		t.Errorf("PasswordSha256 (server sentinel) must NOT appear on wire, got %v", got.InstitutionalRecoveryKey.PasswordSha256)
	}
}

// TestBuildInput_PasswordOmittedWhenNull verifies that a null `password`
// produces a nil *string so the wire field is omitted. Audit §2.6:
// Classic's partial-merge semantics treat the omitted field as
// "preserve" — the right behaviour for write-only credentials.
func TestBuildInput_PasswordOmittedWhenNull(t *testing.T) {
	plan := DiskEncryptionConfigurationResourceModel{
		Name:                  types.StringValue("test"),
		KeyType:               types.StringValue(keyTypeInstitutional),
		FileVaultEnabledUsers: types.StringValue(fileVaultEnabledUsersCurrentOrNext),
		InstitutionalRecoveryKey: &diskEncryptionConfigurationIRKModel{
			Data:     types.StringValue("base64-bytes"),
			Password: types.StringNull(),
		},
	}
	got := buildDiskEncryptionConfigurationInput(plan)
	if got.InstitutionalRecoveryKey.Password != nil {
		t.Errorf("null Password must serialise to nil, got %v", *got.InstitutionalRecoveryKey.Password)
	}
}

// TestCanonicalKeyType_CaseInsensitive pins the case-mapping table used
// by both the plan modifier and the input builder.
func TestCanonicalKeyType_CaseInsensitive(t *testing.T) {
	cases := map[string]string{
		"Individual":                   keyTypeIndividual,
		"INDIVIDUAL":                   keyTypeIndividual,
		"individual":                   keyTypeIndividual,
		"Institutional":                keyTypeInstitutional,
		"institutional":                keyTypeInstitutional,
		"Individual and Institutional": keyTypeIndividualInstitutional,
		"Individual And Institutional": keyTypeIndividualInstitutional,
		"INDIVIDUAL AND INSTITUTIONAL": keyTypeIndividualInstitutional,
		"individual and institutional": keyTypeIndividualInstitutional,
	}
	for in, want := range cases {
		if got := canonicalKeyType(in); got != want {
			t.Errorf("canonicalKeyType(%q) = %q, want %q", in, got, want)
		}
	}

	// Unknown input passes through unchanged — schema validator handles
	// the rejection.
	if got := canonicalKeyType("Bogus"); got != "Bogus" {
		t.Errorf("canonicalKeyType passes unknown values through, got %q", got)
	}
}
