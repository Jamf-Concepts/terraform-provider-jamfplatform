// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ibeacon

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ibeaconTimeoutAttributeTypes defines the timeout attribute types for the
// iBeacon resource operations.
var ibeaconTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// anyMajorMinorSentinel is the classic-wire encoding of "match any value" for
// either the major or minor axis. Jamf Pro accepts it on POST/PUT and emits
// it on GET when the iBeacon was created with `include_any_major_value` or
// `include_any_minor_value` set to true via the UI or API. The two axes are
// independent — Jamf allows major=42 minor=-1 (specific major, any minor) and
// vice versa. The string form is required — wire is XML and both fields are
// modelled as *string in the SDK.
const anyMajorMinorSentinel = "-1"
