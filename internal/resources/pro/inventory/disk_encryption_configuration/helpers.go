// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package disk_encryption_configuration

// Wire enum values for the top-level `<key_type>` element. Audit reference:
// local-testing/diskencryption/AUDIT_FINDINGS.md §1.
//
// The server normalises any case variant to `Individual and Institutional`
// (lowercase `and`) on read. The schema uses stringvalidator.OneOf against
// the wire-canonical spellings — users must supply the exact wire form so
// TF state stays stable across refreshes (matches the directory_binding
// `type` precedent in PR #143). Plan-modifier rewriting on a Required
// attribute violates the Terraform plugin framework's plan==config
// invariant, so case-insensitive acceptance is intentionally not offered.
const (
	keyTypeIndividual              = "Individual"
	keyTypeInstitutional           = "Institutional"
	keyTypeIndividualInstitutional = "Individual and Institutional"
)

// allKeyTypeWireValues lists the canonical wire-form `key_type` values.
var allKeyTypeWireValues = []string{
	keyTypeIndividual,
	keyTypeInstitutional,
	keyTypeIndividualInstitutional,
}

// Wire enum values for the top-level `<file_vault_enabled_users>` element.
const (
	fileVaultEnabledUsersCurrentOrNext   = "Current or Next User"
	fileVaultEnabledUsersManagementAccnt = "Management Account"
)

// allFileVaultEnabledUsersValues lists the accepted file_vault_enabled_users
// wire enum values.
var allFileVaultEnabledUsersValues = []string{
	fileVaultEnabledUsersCurrentOrNext,
	fileVaultEnabledUsersManagementAccnt,
}
