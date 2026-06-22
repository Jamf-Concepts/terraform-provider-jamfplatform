// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_external_source

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// buildPatchExternalSourceInput converts the Terraform plan model into an SDK
// PatchExternalSource payload. Name is required by the schema so we always send
// it as a non-nil pointer. ID is omitted on write — Create uses path id="0" and
// Update derives it from state. Optional/Optional+Computed fields that are
// null or unknown become nil pointers so the SDK's omitempty tag drops them from
// the wire (leaving the server to keep / default the value).
func buildPatchExternalSourceInput(plan PatchExternalSourceResourceModel) *proclassic.PatchExternalSource {
	return &proclassic.PatchExternalSource{
		Name:                         helpers.OptionalStringPointer(plan.Name),
		Enabled:                      helpers.OptionalBoolPointer(plan.Enabled),
		HostName:                     helpers.OptionalStringPointer(plan.HostName),
		Port:                         helpers.OptionalInt64Pointer(plan.Port),
		SslEnabled:                   helpers.OptionalBoolPointer(plan.SslEnabled),
		CertificateValidationEnabled: helpers.OptionalBoolPointer(plan.CertificateValidationEnabled),
	}
}
