// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package disk_encryption_configuration

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// keyTypeToWire returns the wire `key_type` value to send on Create /
// Update. The schema validator (stringvalidator.OneOf) enforces the
// wire-canonical spelling at plan time, so no normalisation is needed
// here. Null / unknown stays nil so the SDK omits the field —
// preserving the server's stored value under Classic's partial-merge
// semantics.
func keyTypeToWire(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	out := v.ValueString()
	return &out
}

// buildDiskEncryptionConfigurationInput converts the Terraform plan model
// into the SDK payload used for both Create and Update.
//
// Wire-quirk handling (audit reference §2.5–§2.10):
//
//   - `PasswordSha256` is never emitted on writes. The server returns a
//     redaction sentinel (`********************`) when a password is set,
//     not a real hash — sending it back would only confuse the server.
//   - `Key` is never emitted on writes. The server derives Subject DN from
//     the uploaded `Data`; the schema marks `key` Computed-only.
//   - `Password` is sent only when the user supplied a plaintext. A null
//     `password` keeps the field nil so the SDK omits it; under Classic's
//     partial-merge semantics the omitted field preserves the server's
//     stored value (audit §2.10 confirms password is independently
//     writable).
//
// `ID` is omitted on write — Create uses path id="0" and Update derives ID
// from state.
func buildDiskEncryptionConfigurationInput(plan DiskEncryptionConfigurationResourceModel) *proclassic.DiskEncryptionConfiguration {
	return &proclassic.DiskEncryptionConfiguration{
		Name:                     helpers.OptionalStringPointer(plan.Name),
		KeyType:                  keyTypeToWire(plan.KeyType),
		FileVaultEnabledUsers:    helpers.OptionalStringPointer(plan.FileVaultEnabledUsers),
		InstitutionalRecoveryKey: buildIRKInput(plan.InstitutionalRecoveryKey),
	}
}

// buildIRKInput converts the TF institutional_recovery_key block into the
// SDK shape. Returns nil when the user omits the block — the SDK's
// omitempty drops `<institutional_recovery_key>` from the wire so the
// server preserves stored values under Classic's partial-merge semantics
// (audit §2.6, §2.7).
//
// `Key` and `CertificateType` are never emitted on writes — both are
// server-derived. `PasswordSha256` is never emitted (the server's read
// value is a redaction sentinel, not a real hash; echoing it back would
// just confuse the server).
func buildIRKInput(m *diskEncryptionConfigurationIRKModel) *proclassic.DiskEncryptionConfigurationInstitutionalRecoveryKey {
	if m == nil {
		return nil
	}
	return &proclassic.DiskEncryptionConfigurationInstitutionalRecoveryKey{
		Password: helpers.OptionalStringPointer(m.Password),
		Data:     helpers.OptionalStringPointer(m.Data),
	}
}
