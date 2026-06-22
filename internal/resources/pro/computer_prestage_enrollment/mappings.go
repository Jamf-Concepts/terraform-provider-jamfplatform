// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package computer_prestage_enrollment

// skipSetupItems wire-key ⇔ snake_case mapping is hand-encoded inside
// buildSkipSetupItemsMap (input_builders.go) and flattenSkipSetupItems
// (state_builders.go). The 25-row table lives there rather than as a map
// indirection so the SDK's typed map signature is preserved without a
// runtime lookup, and so static analysis catches a missing key at compile
// time when a field is renamed.

// Enum value sets (per OpenAPI spec + live wire-probe). Kept in one place so
// schema validators and tests share the source of truth.
var (
	recoveryLockPasswordTypeValues       = []string{"MANUAL", "RANDOM"}
	prestageMinimumOsTargetVersionValues = []string{
		"NO_ENFORCEMENT",
		"MINIMUM_OS_LATEST_VERSION",
		"MINIMUM_OS_LATEST_MAJOR_VERSION",
		"MINIMUM_OS_LATEST_MINOR_VERSION",
		"MINIMUM_OS_SPECIFIC_VERSION",
	}
	userAccountTypeValues = []string{"ADMINISTRATOR", "STANDARD", "SKIP"}
	prefillTypeValues     = []string{"CUSTOM", "DEVICE_OWNER"}
)

// Sentinel constants for nested block IDs / "none" values.
const (
	// sentinelNestedIDForCreate is the literal "-1" Jamf Pro requires on
	// POST nested blocks (locationInformation.id, purchasingInformation.id)
	// to signal "create a new server-side record". Verified by live wire
	// probe — POST with "0" returns 400 INVALID_ID.
	sentinelNestedIDForCreate = "-1"

	// sentinelNoneIDDash1 is Jamf Pro's "none" sentinel for several
	// optional ID fields (siteId, enrollmentSiteId, pssoConfigProfileId,
	// location.buildingId, location.departmentId,
	// customPackageDistributionPointId when unset).
	sentinelNoneIDDash1 = "-1"

	// sentinelDateUnset is the wire-format Jamf returns for an unset
	// purchasing date field.
	sentinelDateUnset = "1970-01-01"
)
