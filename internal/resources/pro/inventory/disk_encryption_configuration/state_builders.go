// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package disk_encryption_configuration

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// assignDiskEncryptionConfigurationResourceModel populates the resource
// model from a DiskEncryptionConfiguration response. state.ID is only
// overwritten when the API ID is non-nil so a transient GET that drops
// the ID does not clobber the value persisted from Create.
//
// `state.InstitutionalRecoveryKey.Password` is deliberately NOT touched.
// The classic GET response never echoes the plaintext password — only
// the `password_sha256` redaction sentinel. Overwriting state.Password
// from the response would null it on every refresh, surfacing as drift
// on the next plan and asking the user to re-supply the password.
// Mirror of directory_binding's pattern (see audit §2.10).
func assignDiskEncryptionConfigurationResourceModel(state *DiskEncryptionConfigurationResourceModel, c *proclassic.DiskEncryptionConfiguration) diag.Diagnostics {
	var diags diag.Diagnostics
	if c == nil {
		return diags
	}
	if c.ID != nil {
		state.ID = helpers.StringValueFromIntPtr(c.ID)
	}
	state.Name = helpers.StringPointerValueOrNull(c.Name)
	state.KeyType = helpers.StringPointerValueOrNull(c.KeyType)
	state.FileVaultEnabledUsers = helpers.StringPointerValueOrNull(c.FileVaultEnabledUsers)

	// Preserve any user-supplied plaintext password across reads — the
	// wire never returns it. Capture before assignIRKResourceModel rebuilds
	// the block so the value is not lost when assignIRKResourceModel
	// allocates a fresh nested struct.
	var preservedPassword *string
	if state.InstitutionalRecoveryKey != nil && !state.InstitutionalRecoveryKey.Password.IsNull() && !state.InstitutionalRecoveryKey.Password.IsUnknown() {
		v := state.InstitutionalRecoveryKey.Password.ValueString()
		preservedPassword = &v
	}

	state.InstitutionalRecoveryKey = assignIRKResourceModel(c.InstitutionalRecoveryKey)
	if state.InstitutionalRecoveryKey != nil && preservedPassword != nil {
		state.InstitutionalRecoveryKey.Password = helpers.StringPointerValueOrNull(preservedPassword)
	}

	return diags
}

// assignDiskEncryptionConfigurationDataSourceModel populates the data
// source model from a response. Symmetric with the resource builder,
// minus the write-only `password` field (data sources are read-only —
// there is no user-supplied plaintext to preserve).
func assignDiskEncryptionConfigurationDataSourceModel(state *DiskEncryptionConfigurationDataSourceModel, c *proclassic.DiskEncryptionConfiguration) diag.Diagnostics {
	var diags diag.Diagnostics
	if c == nil {
		return diags
	}
	if c.ID != nil {
		state.ID = helpers.StringValueFromIntPtr(c.ID)
	}
	state.Name = helpers.StringPointerValueOrNull(c.Name)
	state.KeyType = helpers.StringPointerValueOrNull(c.KeyType)
	state.FileVaultEnabledUsers = helpers.StringPointerValueOrNull(c.FileVaultEnabledUsers)
	state.InstitutionalRecoveryKey = assignIRKDataSourceModel(c.InstitutionalRecoveryKey)
	return diags
}

// assignIRKResourceModel decodes the nested SDK block into the TF model,
// or returns nil when the API did not include the block — OR when the
// block exists on the wire but every field is empty.
//
// Critical wire quirk (audit §1, §2.2): the Jamf Pro server **always**
// emits `<institutional_recovery_key>` on read, even for `key_type =
// "Individual"` where the user never uploaded a cert. In that case the
// element contains only empty self-closing children
// (`<key/>`, `<certificate_type/>`, `<password_sha256/>`, `<data/>`).
// The SDK decodes that as a non-nil
// `DiskEncryptionConfigurationInstitutionalRecoveryKey` struct with all
// fields nil-or-empty-string.
//
// If we surfaced that empty struct as a populated TF model, every plan
// against an `Individual` key_type resource would show a permanent diff:
// the user's config has the block absent (nil pointer), the state has
// the block present (all null fields). We collapse that by returning nil
// when the wire content is empty.
//
// A non-empty wire block — meaning the server actually has IRK material
// — surfaces as a populated TF model so out-of-band cert uploads are
// observable.
func assignIRKResourceModel(k *proclassic.DiskEncryptionConfigurationInstitutionalRecoveryKey) *diskEncryptionConfigurationIRKModel {
	if k == nil {
		return nil
	}
	if irkIsWireEmpty(k) {
		return nil
	}
	return &diskEncryptionConfigurationIRKModel{
		Key:             helpers.StringPointerValueOrNull(k.Key),
		CertificateType: helpers.StringPointerValueOrNull(k.CertificateType),
		PasswordSha256:  helpers.StringPointerValueOrNull(k.PasswordSha256),
		Data:            helpers.StringPointerValueOrNull(k.Data),
		// Password stays whatever the caller set on the state model;
		// the caller (assignDiskEncryptionConfigurationResourceModel)
		// preserves it across reads.
	}
}

// assignIRKDataSourceModel mirrors assignIRKResourceModel for the data
// source — same empty-block collapse, minus the write-only Password
// field (data sources do not surface it).
func assignIRKDataSourceModel(k *proclassic.DiskEncryptionConfigurationInstitutionalRecoveryKey) *diskEncryptionConfigurationIRKDataSourceModel {
	if k == nil {
		return nil
	}
	if irkIsWireEmpty(k) {
		return nil
	}
	return &diskEncryptionConfigurationIRKDataSourceModel{
		Key:             helpers.StringPointerValueOrNull(k.Key),
		CertificateType: helpers.StringPointerValueOrNull(k.CertificateType),
		PasswordSha256:  helpers.StringPointerValueOrNull(k.PasswordSha256),
		Data:            helpers.StringPointerValueOrNull(k.Data),
	}
}

// irkIsWireEmpty reports whether every child element in the IRK block is
// nil or empty-string. The server emits this all-empty shape whenever no
// recovery cert is stored (audit §2.2). See assignIRKResourceModel for
// why we collapse it to a nil TF model.
func irkIsWireEmpty(k *proclassic.DiskEncryptionConfigurationInstitutionalRecoveryKey) bool {
	return ptrStringIsEmpty(k.Key) &&
		ptrStringIsEmpty(k.CertificateType) &&
		ptrStringIsEmpty(k.Data) &&
		ptrStringIsEmpty(k.Password) &&
		ptrStringIsEmpty(k.PasswordSha256)
}

// ptrStringIsEmpty returns true when the pointer is nil or points at the
// empty string.
func ptrStringIsEmpty(p *string) bool {
	return p == nil || *p == ""
}
