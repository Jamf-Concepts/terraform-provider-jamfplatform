// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package advanced_volume_purchasing_content_search

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/criteria"
)

// ValidOperators is the criteria operator vocabulary accepted on advanced volume
// purchasing content search criteria. The Volume-Purchasing-Content criteria are
// content attributes (Content Name, Content Type, Price, …) — none are date or
// membership types — so the UI offers only an 8-operator string + numeric set.
// We derive that subset from the shared canonical vocabulary with criteria.Without,
// dropping the 13 operators that never apply to content attributes (membership,
// every date operator, and the >=/<= numeric variants). The server additionally
// validates operator/criterion pairings per-criterion; this subset only gives
// nicer plan-time feedback for operators that can never be valid here.
var ValidOperators = criteria.Without(
	"has",
	"does not have",
	"member of",
	"not member of",
	"before (yyyy-mm-dd)",
	"after (yyyy-mm-dd)",
	"in less than x days",
	"in more than x days",
	"more than x days ago",
	"less than x days ago",
	"greater than",
	"greater than or equal",
	"less than or equal",
) // → is, is not, like, not like, matches regex, does not match regex, more than, less than
