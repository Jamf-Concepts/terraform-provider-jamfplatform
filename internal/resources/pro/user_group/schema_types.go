// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package user_group

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// userGroupTimeoutAttributeTypes defines the timeout attribute types for the
// user group resource operations.
var userGroupTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// noSiteID is the Jamf Pro sentinel for "no site assignment" — the classic
// API always emits <site><id>-1</id><name>NONE</name></site> when the user
// did not specify one. We pass this back on POST when the user omitted
// site_id so the wire payload is explicit and reads remain stable.
const noSiteID = "-1"
