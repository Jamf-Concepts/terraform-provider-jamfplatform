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

// legacyPayloadSettingsBehaviour documents how Jamf treats the settings written for a legacy
// payload, appended to every legacy-payload schema description. Each rule is observed wire
// behaviour, not provider behaviour, and each one shapes what a plan can show.
const legacyPayloadSettingsBehaviour = "Jamf validates each payload against Apple's payload keys for its `payload_type`: " +
	"a key Apple does not define for that payload type is **silently discarded**, so it will not reach any device and Terraform " +
	"reports a difference on every plan (the provider warns and names the discarded keys on apply); a key whose value has the wrong " +
	"type is **rejected**, failing the apply with a validation error; and a key set to `null` is discarded, which the provider " +
	"tolerates, so nulls can stay in configuration without producing a perpetual diff. Key names are matched case-insensitively " +
	"and stored under Apple's spelling, so match Apple's capitalisation exactly to avoid a permanent difference. " +
	"Jamf also stamps Apple's common payload metadata (`payloadDisplayName`, `payloadOrganization`, `payloadUUID`, `payloadVersion`) " +
	"onto every payload; the provider hides those unless you set them yourself."
