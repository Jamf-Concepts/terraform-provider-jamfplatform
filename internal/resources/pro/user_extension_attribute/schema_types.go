// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package user_extension_attribute

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// userExtensionAttributeTimeoutAttributeTypes defines the timeout attribute
// types for the resource operations.
var userExtensionAttributeTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// Probed enum value sets (live tenant, 2026-06-02). The Classic API is
// permissive (it accepted out-of-set values), but the admin UI offers only
// these, so OneOf validators constrain to the UI-valid set. Classic strings are
// human-cased, not SCREAMING_SNAKE.
var (
	// validDataTypes is the "Data Type" dropdown: String / Integer / Date.
	validDataTypes = []string{"String", "Integer", "Date"}

	// validInputTypes is the "Input Type" dropdown. The user-EA UI offers only
	// Text Field / Pop-up Menu — there is no LDAP/Directory Service option (the
	// Classic user-EA payload carries no mapping field; the server silently
	// coerces "LDAP Mapping" to "Text Field").
	validInputTypes = []string{"Text Field", "Pop-up Menu"}
)

// Input-type discriminator constants.
const (
	inputTypeTextField = "Text Field"
	inputTypePopupMenu = "Pop-up Menu"
)
