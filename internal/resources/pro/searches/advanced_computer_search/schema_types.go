// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package advanced_computer_search

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// advancedComputerSearchTimeoutAttributeTypes defines the timeout attribute
// types for the advanced computer search resource operations.
var advancedComputerSearchTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// noSiteID is the Jamf Pro sentinel for "no site assignment" — the classic API
// always emits <site><id>-1</id><name>NONE</name></site> when no site is set.
// We always serialise the site so the wire payload is explicit and reads stay
// stable.
const noSiteID = "-1"
