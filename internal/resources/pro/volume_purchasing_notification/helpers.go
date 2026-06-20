// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package volume_purchasing_notification

// Wire enum values, wire-probed against the live endpoint (a junk value returns
// 400 INVALID_FIELD listing the accepted set). These back static OneOf validators.
const (
	// Triggers — UI "Notifications" trigger checkboxes.
	triggerRemovedFromAppStore = "REMOVED_FROM_APP_STORE" // UI "Item removed from the App Store"
	triggerNoMoreLicenses      = "NO_MORE_LICENSES"       // UI "No more licenses available"

	// frequencyDaily is the only accepted internal-recipient frequency. The UI
	// exposes no frequency control (the Scope tab is "Users to distribute a daily
	// summary email to"), so the provider synthesises this on every write and
	// drops it on read.
	frequencyDaily = "DAILY"

	// siteNone is the Jamf Pro "no site" sentinel for site_id.
	siteNone = "-1"
)

// triggerEnumValues is the set accepted by the triggers field.
var triggerEnumValues = []string{triggerRemovedFromAppStore, triggerNoMoreLicenses}
