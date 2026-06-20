// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package user_group

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/criteria"
)

// ValidOperators is the criteria operator vocabulary accepted on user-group
// criteria: the shared canonical set minus the two date-window operators. The
// classic /usergroups endpoint returns 409 "Problem with criteria" for "in less
// than x days" / "in more than x days" on user extension-attribute criteria
// (wire-probed during this resource's build) — they apply only to device /
// certificate-expiry date criteria. Device/computer groups and advanced searches
// use the full criteria.Operators set.
var ValidOperators = criteria.Without("in less than x days", "in more than x days")
