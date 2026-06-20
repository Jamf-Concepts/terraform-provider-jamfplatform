// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package class

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// classTimeoutAttributeTypes defines the timeout attribute types for the class
// resource operations.
var classTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// noSiteID is the Jamf Pro sentinel for "no site assignment" — the classic API
// emits <site><id>-1</id><name>NONE</name></site> when no site is set.
const noSiteID = "-1"
