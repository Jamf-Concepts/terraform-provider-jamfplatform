// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package user_group

import (
	"strings"
)

// ValidOperators contains the Jamf classic <search_type> values accepted on
// user-group criteria. The classic OpenAPI spec types <search_type> as a
// free-form string with only an example, so this list is provider-side
// guardrail rather than a server-enforced enum.
//
// NOTE: this duplicates internal/resources/device_group/mappings.go
// ValidOperators. The two consumers cover Platform Services + ProClassic
// inventory criteria — both are inventory-style criteria with the same
// operator vocabulary in practice. Per STYLE_GUIDE §Shared schemas
// (deferred abstraction), the abstraction trigger is 3 verified-identical
// shapes across shipped resources; we will revisit when a third consumer
// lands (e.g. jamfplatform_pro_computer_group or an advanced search).
var ValidOperators = []string{
	"is",
	"is not",
	"has",
	"does not have",
	"member of",
	"not member of",
	"before (yyyy-mm-dd)",
	"after (yyyy-mm-dd)",
	// "in less than x days" and "in more than x days" are intentionally
	// omitted: Jamf classic only honours these for certificate-expiry
	// criteria, not for user attributes / user extension attributes. They
	// surface a 409 "Problem with criteria" on /usergroups.
	"more than x days ago",
	"less than x days ago",
	"like",
	"not like",
	"greater than",
	"more than",
	"less than",
	"greater than or equal",
	"less than or equal",
	"matches regex",
	"does not match regex",
}

// operatorDescription returns a formatted MarkdownDescription listing all
// valid operator strings as inline code.
func operatorDescription() string {
	quoted := make([]string, len(ValidOperators))
	for i, v := range ValidOperators {
		quoted[i] = "`" + v + "`"
	}
	return "Operator to apply. Valid values are " + strings.Join(quoted, ", ") + "."
}
