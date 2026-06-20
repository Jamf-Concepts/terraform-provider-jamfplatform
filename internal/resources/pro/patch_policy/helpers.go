// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package patch_policy

import (
	"strconv"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// extractPatchPolicyID returns the assigned ID as a string from a Create/GET
// response. Create returns the ID at the top level (<patch_policy><id>); GET
// echoes the same. Prefer the top-level reading, fall back to general.id.
func extractPatchPolicyID(p *proclassic.PatchPolicy) string {
	if p == nil {
		return ""
	}
	if p.ID != nil {
		return strconv.Itoa(*p.ID)
	}
	if p.General != nil && p.General.ID != nil {
		return strconv.Itoa(*p.General.ID)
	}
	return ""
}

// preferCurrentInt64Pointer is the int64 sibling of preferCurrentStringPointer.
func preferCurrentInt64Pointer(api *int, current types.Int64) types.Int64 {
	if helpers.IsConfiguredValue(current) {
		return current
	}
	if api == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*api))
}

// int64ValueOrNull maps an SDK *int onto a Terraform Int64, null for nil. Used
// for the server-derived release_date (Computed-only).
func int64ValueOrNull(p *int) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*p))
}
