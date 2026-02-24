// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import (
	"strings"
)

// ValidOperators contains all valid search type values for criteria validation.
var ValidOperators = []string{
	"is",
	"is not",
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

// operatorDescription returns a formatted description string listing all valid operators.
func operatorDescription() string {
	quotedOperators := make([]string, len(ValidOperators))
	for i, v := range ValidOperators {
		quotedOperators[i] = "`" + v + "`"
	}
	return "Operator to apply. Valid values are " + strings.Join(quotedOperators, ", ") + "."
}
