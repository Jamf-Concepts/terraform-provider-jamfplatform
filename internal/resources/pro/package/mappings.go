// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pkg

import "strings"

// CategoryIDDefault is the sentinel Jamf Pro returns for "no category".
// Matches the `pro_script` precedent and surface across the categories API.
const CategoryIDDefault = "-1"

// PriorityDefault is the server-side default the Jamf Pro UI presents for
// a fresh package record. Confirmed in §12.2 wire probe.
const PriorityDefault = 10

// PackageFilterSelectors enumerates the RSQL selectors the
// /v1/packages list endpoint accepts. Sourced from the server error
// response when an unsupported selector is sent. Note: alphabetised here
// for stable schema descriptions; semantics unaffected.
var PackageFilterSelectors = []string{
	"categoryId",
	"cloudTransferStatus",
	"fileName",
	"id",
	"info",
	"manifestFileName",
	"notes",
	"packageName",
}

// AllowedHashTypeValues are the hash algorithms Jamf Pro accepts on the
// `hash_type` attribute. Wire-probed against a live tenant: every listed value
// is settable on a write and the server rejects anything else. A cloud
// distribution point upload sets the value to `SHA3_512`; a record that has
// never had a file uploaded reads back `SHA_512`.
var AllowedHashTypeValues = []string{"MD5", "SHA_256", "SHA_512", "SHA3_512"}

// markdownValueList renders a slice of enum values as a backticked,
// comma-separated list for MarkdownDescription strings. Deriving the documented
// values from the same slice the OneOf validator uses keeps the docs and the
// validator from drifting apart.
func markdownValueList(vals []string) string {
	quoted := make([]string, len(vals))
	for i, v := range vals {
		quoted[i] = "`" + v + "`"
	}
	return strings.Join(quoted, ", ")
}
