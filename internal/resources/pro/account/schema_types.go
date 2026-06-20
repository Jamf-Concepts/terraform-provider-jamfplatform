// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package account

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// accountTimeoutAttributeTypes defines the timeout attribute types.
var accountTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// UI-facing enum values (config). Translated to/from the Pro wire spellings
// (wire-probed 2026-06-12) by the maps below.
var (
	accessLevelValues  = []string{"Full Access", "Site Access", "Group Access"}
	privilegeSetValues = []string{"Administrator", "Auditor", "Enrollment Only", "Custom"}
	accessStatusValues = []string{"Enabled", "Disabled"}
	accountTypeValues  = []string{"DEFAULT", "FEDERATED"}
)

// access_level UI <-> Pro wire. Pro uses "GroupBasedAccess" (not "GroupAccess").
var accessLevelToWire = map[string]string{
	"Full Access":  "FullAccess",
	"Site Access":  "SiteAccess",
	"Group Access": "GroupBasedAccess",
}
var accessLevelFromWire = invert(accessLevelToWire)

// privilege_set UI <-> Pro wire. Pro uses "ENROLLMENT" (not "ENROLLMENT_ONLY").
var privilegeSetToWire = map[string]string{
	"Administrator":   "ADMINISTRATOR",
	"Auditor":         "AUDITOR",
	"Enrollment Only": "ENROLLMENT",
	"Custom":          "CUSTOM",
}
var privilegeSetFromWire = invert(privilegeSetToWire)

func invert(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[v] = k
	}
	return out
}

// translate returns m[v]; if v is absent it returns v unchanged so an
// unexpected server value surfaces verbatim rather than being silently blanked.
func translate(m map[string]string, v string) string {
	if out, ok := m[v]; ok {
		return out
	}
	return v
}
