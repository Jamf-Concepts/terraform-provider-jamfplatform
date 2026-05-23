// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package pkg

// CategoryIDDefault is the sentinel Jamf Pro returns for "no category".
// Matches the `pro_script` precedent and surface across the categories API.
const CategoryIDDefault = "-1"

// PriorityDefault is the server-side default the Jamf Pro UI presents for
// a fresh package record. Confirmed in §12.2 wire probe.
const PriorityDefault = 10

// PackageFilterSelectors enumerates the RSQL selectors the
// /v1/packages list endpoint accepts. Lifted verbatim from PACKAGE_SPIKE
// §13.1 (server error response when an unsupported selector is sent).
// Note: alphabetised here for stable schema descriptions; semantics
// unaffected.
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

// AllowedHashTypeValues are the values accepted on the `hash_type` schema
// attribute. The server may return additional historical values (notably
// `"SHA_512"`, the unconfigured-record default) — those are accepted on
// reads but rejected at plan time on writes per PACKAGE_SPIKE §3.Q5.
var AllowedHashTypeValues = []string{"MD5", "SHA_256", "SHA3_512"}
