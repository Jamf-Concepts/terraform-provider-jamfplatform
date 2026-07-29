// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_extension_attribute

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// computerExtensionAttributeTimeoutAttributeTypes defines the timeout attribute
// types for the resource operations.
var computerExtensionAttributeTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// Probed enum value sets (live tenant, 2026-06-02). Build OneOf validators from
// these; do not assume — Jamf rejects out-of-set values with 400.
var (
	// validDataTypes is the "Data type" dropdown: String / Integer / Date.
	validDataTypes = []string{"STRING", "INTEGER", "DATE"}

	// validInputTypes is the "Input type" set. The modern admin UI offers only
	// TEXT / POPUP / SCRIPT for new computer EAs, but the API accepts and
	// round-trips DIRECTORY_SERVICE_ATTRIBUTE_MAPPING and existing LDAP-mapped
	// EAs use it, so it is retained here so such EAs can be imported and managed.
	validInputTypes = []string{"TEXT", "POPUP", "SCRIPT", "DIRECTORY_SERVICE_ATTRIBUTE_MAPPING"}

	// validManageExistingData is the accepted set for manage_existing_data
	// (wire-probed: the Pro PUT for a SCRIPT EA requires DELETE or RETAIN).
	validManageExistingData = []string{"DELETE", "RETAIN"}

	// manageExistingDataDefault is sent when a SCRIPT EA update disables the EA
	// and the user omitted manage_existing_data — RETAIN preserves
	// already-collected inventory data, the safe default. See
	// manageExistingDataFor for when the field is sent at all.
	manageExistingDataDefault = "RETAIN"

	// validInventoryDisplays is the "Inventory display" dropdown (6 tabs).
	validInventoryDisplays = []string{
		"GENERAL",
		"HARDWARE",
		"OPERATING_SYSTEM",
		"USER_AND_LOCATION",
		"PURCHASING",
		"EXTENSION_ATTRIBUTES",
	}
)

// Input-type discriminator constants.
const (
	inputTypeText   = "TEXT"
	inputTypePopup  = "POPUP"
	inputTypeScript = "SCRIPT"
	inputTypeLDAP   = "DIRECTORY_SERVICE_ATTRIBUTE_MAPPING"
)
