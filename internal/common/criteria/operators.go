// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package criteria holds shared building blocks for Jamf "smart" group and
// advanced-search criteria. It owns the canonical operator vocabulary (the
// classic <search_type> value) shared by every criteria-bearing construct:
// jamfplatform_device_group (Platform Services), jamfplatform_pro_user_group
// (ProClassic), and the forthcoming Pro advanced computer/device/user search
// resources.
//
// The vocabulary is parameterisable per consumer: some operators only make sense
// for some object types. Device/computer groups and the computer / mobile-device
// advanced searches use the full set (Operators). Two consumers use a subset via
// Without:
//   - jamfplatform_pro_user_group drops the two date-window operators
//     ("in less than x days" / "in more than x days") — the classic /usergroups
//     endpoint returns 409 "Problem with criteria" for them on user
//     extension-attribute criteria (wire-probed during the user_group build).
//   - jamfplatform_pro_advanced_volume_purchasing_content_search drops the 13
//     operators absent from the Volume-Purchasing-Content criteria UI (membership,
//     all date operators, and >=/<=), leaving the 8-operator string+numeric set
//     the content attributes actually accept (UI-confirmed during the build).
//
// Field/operator validity beyond the vocabulary is enforced server-side (the Pro
// /v1 advanced-search endpoints reject operator/field mismatches per-criterion).
package criteria

import (
	"slices"
	"strings"
)

// Operators is the full canonical set of classic <search_type> operator strings.
// Use it directly for device/computer groups and advanced searches; derive a
// subset with Without for constructs that cannot use the whole vocabulary.
var Operators = []string{
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

// Without returns a copy of Operators with the named operators removed, in the
// canonical order. Used by consumers that accept only a subset of the vocabulary.
func Without(exclude ...string) []string {
	out := make([]string, 0, len(Operators))
	for _, op := range Operators {
		if !slices.Contains(exclude, op) {
			out = append(out, op)
		}
	}
	return out
}

// Description returns the schema MarkdownDescription listing the given operator
// set as inline code. Pass the consumer's actual accepted set so the docs match
// the validator.
func Description(ops []string) string {
	quoted := make([]string, len(ops))
	for i, v := range ops {
		quoted[i] = "`" + v + "`"
	}
	return "Operator to apply. Valid values are " + strings.Join(quoted, ", ") + "."
}
