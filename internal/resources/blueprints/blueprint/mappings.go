// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package blueprint

// Constants for blueprint deployment states.
const (
	blueprintDeploymentStateDeployed    = "DEPLOYED"
	blueprintDeploymentStateNotDeployed = "NOT_DEPLOYED"
)

// stronglyTypedComponentIdentifiers lists all component identifiers that have strongly-typed representations.
var stronglyTypedComponentIdentifiers = map[string]struct{}{
	"com.jamf.ddm.audio-accessory-settings":    {},
	"com.jamf.ddm.custom-declarations":         {},
	"com.jamf.ddm.disk-management":             {},
	"com.jamf.ddm.math-settings":               {},
	"com.jamf.ddm.passcode-settings":           {},
	"com.jamf.ddm.safari-bookmarks":            {},
	"com.jamf.ddm.safari-extensions":           {},
	"com.jamf.ddm.safari-settings":             {},
	"com.jamf.ddm.service-background-tasks":    {},
	"com.jamf.ddm.service-configuration-files": {},
	"com.jamf.ddm.sw-updates":                  {},
	"com.jamf.ddm.software-update-settings":    {},
	"com.jamf.ddm-configuration-profile":       {},
}
