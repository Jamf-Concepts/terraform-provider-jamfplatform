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

// TestBuildInput_KeyTypePassthrough verifies the input builder forwards
// the wire-canonical `key_type` value verbatim. The schema validator
// (stringvalidator.OneOf) ensures only canonical spellings reach the
// builder; case folding is intentionally not offered (a plan-modifier
// rewrite on a Required attribute violates the framework's plan==config
// invariant, see STYLE_GUIDE §Plan-modifier rewrites on Required
// attributes).
func TestBuildInput_KeyTypePassthrough(t *testing.T) {
	plan := DiskEncryptionConfigurationResourceModel{
		Name:                  types.StringValue("test"),
		KeyType:               types.StringValue(keyTypeIndividualInstitutional),
		FileVaultEnabledUsers: types.StringValue(fileVaultEnabledUsersCurrentOrNext),
	}
	got := buildDiskEncryptionConfigurationInput(plan)
	if got.KeyType == nil || *got.KeyType != keyTypeIndividualInstitutional {
		t.Errorf("KeyType must pass through to wire form %q, got %v", keyTypeIndividualInstitutional, got.KeyType)
	}
}

// TestBuildInput_IRKBlock_PKCS12 covers the PKCS12 path. `certificate_type`,
// `data`, and `password` must reach the wire. `key` and `password_sha256`
// must NOT — they are server-side and writing them back would confuse
// the server.
func TestBuildInput_IRKBlock_PKCS12(t *testing.T) {
	plan := DiskEncryptionConfigurationResourceModel{
		Name:                  types.StringValue("test-pkcs12"),
		KeyType:               types.StringValue(keyTypeInstitutional),
		FileVaultEnabledUsers: types.StringValue(fileVaultEnabledUsersCurrentOrNext),
		InstitutionalRecoveryKey: &diskEncryptionConfigurationIRKModel{
			Key:             types.StringValue("CN=server-derived"), // should not reach wire
			CertificateType: types.StringValue("PKCS12"),
			PasswordSha256:  types.StringValue("********************"), // should not reach wire
			Password:        types.StringValue("hunter2"),
			Data:            types.StringValue("base64-cert-bytes"),
		},
	}
	got := buildDiskEncryptionConfigurationInput(plan)
	if got.InstitutionalRecoveryKey == nil {
		t.Fatalf("IRK must be non-nil")
	}
	if got.InstitutionalRecoveryKey.CertificateType == nil || *got.InstitutionalRecoveryKey.CertificateType != "PKCS12" {
		t.Errorf("CertificateType must reach wire (server requires it on POST), got %v", got.InstitutionalRecoveryKey.CertificateType)
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
