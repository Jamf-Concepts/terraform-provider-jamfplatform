// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package return_to_service

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// returnToServiceTimeoutAttributeTypes defines the timeout attribute types for
// the Return to Service resource operations.
var returnToServiceTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// wholeNumberRegex matches a positive whole number with no leading zero. The
// Jamf Pro Return to Service endpoint rejects a wifi_profile_id that is not "a
// whole number greater than 0"; validating at plan time gives a clean error
// instead of a server round-trip.
var wholeNumberRegex = regexp.MustCompile(`^[1-9][0-9]*$`)
