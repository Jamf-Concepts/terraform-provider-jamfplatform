// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package advanced_mobile_device_search

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// advancedMobileDeviceSearchTimeoutAttributeTypes defines the timeout attribute
// types for the advanced mobile device search resource operations.
var advancedMobileDeviceSearchTimeoutAttributeTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// noSiteID is the Jamf Pro sentinel for "no site assignment". The Pro /v1
// advanced-search payload returns `siteId: "-1"` when no site is set, and we
// always send a site_id so the wire payload is explicit and reads stay stable.
const noSiteID = "-1"
