// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pkg

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// buildPackageInput converts the Terraform plan into a SDK Package payload
// suitable for POST or PUT. The function picks the correct mode based on
// whether `package_file_source` is set:
//
//   - JCDS mode (PackageFileSource non-empty) emits only metadata fields and
//     leaves the hash/size pointers nil. The server populates them after the
//     subsequent upload + verification poll. ConflictsWith ensures the user
//     cannot supply hash attrs in this mode.
//   - FSDP-with-hashes mode emits user-supplied hash pointers via
//     OptionalStringPointer; the server stores whatever the user provides.
//   - Pure metadata-only mode emits nil hash pointers; the server returns
//     empty strings on read.
//
// Value-type bools (`fillUserTemplate`, `rebootRequired`, ...) are always
// emitted with their current value — the wire probe (§12.2) confirmed the
// server accepts zero-value bools cleanly. The PUT semantics are
// full-replace (§13.8.A.7), so the input builder MUST always emit the full
// intended state.
func buildPackageInput(plan PackageResourceModel) *pro.Package {
	input := &pro.Package{
		PackageName: plan.DisplayName.ValueString(),
		FileName:    plan.FileName.ValueString(),
	}

	// CategoryID: server default `"-1"`. Use the user value when set;
	// otherwise emit the sentinel to mirror the UI's "None" default.
	if helpers.IsConfiguredValue(plan.CategoryID) && plan.CategoryID.ValueString() != "" {
		input.CategoryID = plan.CategoryID.ValueString()
	} else {
		input.CategoryID = CategoryIDDefault
	}

	// Priority: server default 10. Value-type int — emit the chosen value.
	if helpers.IsConfiguredValue(plan.Priority) {
		input.Priority = int(plan.Priority.ValueInt64())
	} else {
		input.Priority = PriorityDefault
	}

	// Value-type bools: server accepts zero-values cleanly. Read the plan
	// directly; an unset (null) bool falls through to false, which matches
	// the UI defaults.
	input.FillUserTemplate = plan.FillUserTemplate.ValueBool()
	input.RebootRequired = plan.RebootRequired.ValueBool()

	// Pointer bools: emit only when the user set them. `Swu`,
	// `FillExistingUsers`, and `OsInstall` (deferred) are pointer bools in
	// the SDK; nil leaves the field unset on POST so the server applies
	// its default.
	input.FillExistingUsers = helpers.OptionalBoolPointer(plan.FillExistingUsers)
	input.Swu = helpers.OptionalBoolPointer(plan.AvailableInSoftwareUpdate)

	// Optional strings: omit on the wire (nil pointer) when unset.
	input.Info = helpers.OptionalStringPointer(plan.Info)
	input.Notes = helpers.OptionalStringPointer(plan.Notes)
	input.OsRequirements = helpers.OptionalStringPointer(plan.OSRequirements)

	// Hash attributes — emitted in three modes:
	//   - JCDS Create: plan modifier leaves Unknown → nil → server populates after upload.
	//   - JCDS Update with re-upload: caller overrides HashType/HashValue/Sha3512 with the
	//     locally-computed digest (see crud.go) BEFORE this builder runs against the input
	//     it returns; that path bypasses the values here.
	//   - JCDS Update without re-upload: plan modifier carries state values forward; we
	//     emit them so the full-replace PUT does not clear server-side hashes.
	//   - FSDP modes: user-supplied values come through plan.
	input.HashType = helpers.OptionalStringPointer(plan.HashType)
	input.HashValue = helpers.OptionalStringPointer(plan.HashValue)
	input.Sha3512 = helpers.OptionalStringPointer(plan.Sha3512)
	input.Sha256 = helpers.OptionalStringPointer(plan.Sha256)
	input.Md5 = helpers.OptionalStringPointer(plan.Md5)

	// Size: emit from plan when known so the full-replace PUT preserves the
	// server-derived value. Audit §13.8 A.7 confirmed the server ignores
	// user-supplied size on never-uploaded records — but on records with an
	// uploaded binary, omitting size from the PUT body clears the field
	// (see ManifestLifecycle Step 2 regression). Echoing state's value back
	// keeps the field stable.
	input.Size = helpers.OptionalStringPointer(plan.Size)

	return input
}

// mergePlanIntoServerState takes a freshly-fetched server record and
// overlays the user-managed metadata fields from plan, preserving every
// server-derived attribute (cloud_transfer_status, install_language,
// parent_package_id, self_healing_action, indexed, format, ...) that the
// resource does not own. The spike's S5 probe matched this shape: PUT body
// echoes the GET response with user changes layered on top, so a
// full-replace PUT does not silently clear fields the provider never
// chose to manage.
//
// Caller may still override HashType/HashValue/Sha3512 when re-uploading
// (see crud.go willReupload branch).
func mergePlanIntoServerState(plan PackageResourceModel, server *pro.Package) *pro.Package {
	// Start with a copy of the server's current Package so server-derived
	// fields the provider does not touch (CloudTransferStatus, Format,
	// Indexed, InstallLanguage, ParentPackageID, SelfHeal*, ...) survive
	// the PUT.
	merged := *server

	// User-managed metadata: take from plan.
	merged.PackageName = plan.DisplayName.ValueString()
	merged.FileName = plan.FileName.ValueString()

	if helpers.IsConfiguredValue(plan.CategoryID) && plan.CategoryID.ValueString() != "" {
		merged.CategoryID = plan.CategoryID.ValueString()
	} else {
		merged.CategoryID = CategoryIDDefault
	}

	if helpers.IsConfiguredValue(plan.Priority) {
		merged.Priority = int(plan.Priority.ValueInt64())
	} else {
		merged.Priority = PriorityDefault
	}

	merged.FillUserTemplate = plan.FillUserTemplate.ValueBool()
	merged.RebootRequired = plan.RebootRequired.ValueBool()
	merged.FillExistingUsers = helpers.OptionalBoolPointer(plan.FillExistingUsers)
	merged.Swu = helpers.OptionalBoolPointer(plan.AvailableInSoftwareUpdate)

	merged.Info = helpers.OptionalStringPointer(plan.Info)
	merged.Notes = helpers.OptionalStringPointer(plan.Notes)
	merged.OsRequirements = helpers.OptionalStringPointer(plan.OSRequirements)

	// Hash attributes: prefer user-supplied (FSDP mode), else echo server
	// state. JCDS-mode caller overrides these to localSha3 after merge.
	if helpers.IsConfiguredValue(plan.HashType) {
		merged.HashType = helpers.OptionalStringPointer(plan.HashType)
	}
	if helpers.IsConfiguredValue(plan.HashValue) {
		merged.HashValue = helpers.OptionalStringPointer(plan.HashValue)
	}
	if helpers.IsConfiguredValue(plan.Sha3512) {
		merged.Sha3512 = helpers.OptionalStringPointer(plan.Sha3512)
	}
	if helpers.IsConfiguredValue(plan.Sha256) {
		merged.Sha256 = helpers.OptionalStringPointer(plan.Sha256)
	}
	if helpers.IsConfiguredValue(plan.Md5) {
		merged.Md5 = helpers.OptionalStringPointer(plan.Md5)
	}

	return &merged
}
