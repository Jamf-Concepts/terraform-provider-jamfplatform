// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package blueprint

import "github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/blueprints"

// Constants for blueprint deployment states. The SDK also generates
// OUT_OF_DATE, which nothing here compares against.
const (
	blueprintDeploymentStateDeployed    = blueprints.DeploymentStateStateDeployed
	blueprintDeploymentStateNotDeployed = blueprints.DeploymentStateStateNotDeployed
)

// stronglyTypedComponentIdentifiers lists all component identifiers that have strongly-typed representations.
var stronglyTypedComponentIdentifiers = map[string]struct{}{
	"com.jamf.ai-governance":                   {},
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
	"com.jamf.ddm-strict":                      {},
}

// blockOnlyComponentIdentifiers lists the strongly-typed components the provider offers only inside
// component_blocks. The flat top-level component style is deprecated and removed on or after
// 2026-10-22, so no component added after it was deprecated gains a flat attribute — which leaves a
// flat-mode read with nowhere to put one. Such a component is kept in raw_component rather than
// dropped, because a flat-mode apply rewrites every step: a component missing from state is deleted
// from the blueprint with no plan diff showing the deletion. Every entry must also appear in
// stronglyTypedComponentIdentifiers, and state_builders_test.go pins that a typed component with no
// BlueprintResourceModel field is registered here.
var blockOnlyComponentIdentifiers = map[string]struct{}{
	"com.jamf.ai-governance": {},
	"com.jamf.ddm-strict":    {},
}

// legacyPayloadSettingsBehaviour documents how Jamf treats the settings written for a legacy payload,
// appended to every legacy-payload schema description. It deliberately covers only what an author
// cannot learn from a diagnostic: what the provider absorbs silently, and the one case it cannot fix
// (an import cannot recover a redacted value). The rules the plan-time schema check reports — an
// unrecognised or miscased key, a wrong value type, a missing required key, an out-of-range integer —
// are named there, with the offending path and Apple's spelling, so they are summarised here rather
// than enumerated. See internal/common/appleprofiles.
const legacyPayloadSettingsBehaviour = "The platform validates each payload against Apple's payload keys for its `payload_type`, " +
	"and the provider checks the same rules during `plan`, so an unrecognised or miscased key, a wrong value type, " +
	"or a missing required key is reported before an apply rather than failing one. " +
	"Each of those is an **error**: Jamf drops a key it does not recognise while reporting success, so a payload " +
	"carrying one never applies. To skip the checks, move **every** legacy payload in the same block to a " +
	"single `raw_component` with identifier `com.jamf.ddm-configuration-profile` — the platform stores a " +
	"block's legacy payloads as one component, so a partial move would write that component twice and stop " +
	"the payloads left behind from being reconciled against the platform. " +
	"Two behaviours are absorbed for you instead: a key set to `null` is discarded by Jamf and tolerated here, so nulls " +
	"can stay in configuration; and Apple's common payload metadata (`payloadDisplayName`, `payloadOrganization`, " +
	"`payloadUUID`, `payloadVersion`) is stamped onto every payload and hidden unless you set it yourself. " +
	"Values the platform treats as credentials (a Wi-Fi `Password`, and `EAPClientConfiguration`'s `UserName`, `UserPassword` " +
	"and `OuterIdentity`) are returned redacted, and the provider keeps what you wrote so the plan still settles. " +
	"An imported blueprint carries the redaction, because the real value cannot be read back."
