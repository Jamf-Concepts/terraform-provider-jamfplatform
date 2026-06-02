// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package advanced_user_search

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/criteria"
)

// ValidOperators is the criteria operator vocabulary accepted on advanced user
// search criteria: the shared canonical set minus the two date-window operators,
// matching jamfplatform_pro_user_group. User-attribute criteria are not date
// values, so the date-window operators do not apply.
//
// Note: a live wire-probe found the classic /advancedusersearches endpoint
// *accepts* (stores without a 409) a date-window operator on a user criterion —
// unlike /usergroups, which rejects it. That acceptance is weak evidence of
// validity (the classic API stores unvalidated operator strings), so this
// resource keeps the conservative subset for consistency with user groups. If a
// concrete user-search use case for the date-window operators emerges, switch to
// criteria.Operators.
var ValidOperators = criteria.Without("in less than x days", "in more than x days")
