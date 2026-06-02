// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_extension_attribute

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// mobileDeviceExtensionAttributeTimeoutAttributeTypes defines the timeout
// attribute types for the resource operations.
var mobileDeviceExtensionAttributeTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// Probed enum value sets (live tenant, 2026-06-02). Mobile-device EAs cannot run
// scripts, so SCRIPT is absent; OPERATING_SYSTEM is not a valid inventory
// display for mobile devices.
var (
	// validDataTypes is the "Data type" dropdown: String / Integer / Date.
	validDataTypes = []string{"STRING", "INTEGER", "DATE"}

	// validInputTypes is the "Input type" set (no SCRIPT for mobile).
	validInputTypes = []string{"TEXT", "POPUP", "DIRECTORY_SERVICE_ATTRIBUTE_MAPPING"}

	// validInventoryDisplays is the "Inventory display" dropdown (5 — no
	// OPERATING_SYSTEM for mobile).
	validInventoryDisplays = []string{
		"GENERAL",
		"HARDWARE",
		"USER_AND_LOCATION",
		"PURCHASING",
		"EXTENSION_ATTRIBUTES",
	}
)

// Input-type discriminator constants.
const (
	inputTypeText  = "TEXT"
	inputTypePopup = "POPUP"
	inputTypeLDAP  = "DIRECTORY_SERVICE_ATTRIBUTE_MAPPING"
)
