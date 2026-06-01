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

// stringIDPtr parses a TF String holding a numeric ID into *int. Returns nil for
// null/unknown/empty/un-parseable.
func stringIDPtr(value types.String) *int {
	if !helpers.IsConfiguredValue(value) {
		return nil
	}
	s := value.ValueString()
	if s == "" {
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &v
}

// stringFromIntPtr renders an *int as an *string for the preferCurrent helpers.
func stringFromIntPtr(p *int) *string {
	if p == nil {
		return nil
	}
	s := strconv.Itoa(*p)
	return &s
}

// preferCurrentStringPointer returns the caller's configured value when set,
// otherwise adopts the API value (or null when both are absent). Protects
// Optional+Computed scalars in managed sections against classic-API echo
// quirks, at the cost of not detecting server-side drift — the standard
// ProClassic tradeoff (see restricted_software / policy).
func preferCurrentStringPointer(api *string, current types.String) types.String {
	if helpers.IsConfiguredValue(current) {
		return current
	}
	if api == nil {
		return types.StringNull()
	}
	return types.StringValue(*api)
}

// preferCurrentBoolPointer is the bool sibling of preferCurrentStringPointer.
func preferCurrentBoolPointer(api *bool, current types.Bool) types.Bool {
	if helpers.IsConfiguredValue(current) {
		return current
	}
	if api == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*api)
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
