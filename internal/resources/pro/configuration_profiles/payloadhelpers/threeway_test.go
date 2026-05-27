// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package payloadhelpers

import "testing"

// Three-way fixtures cover the four wire scenarios the design hinges on:
//
//   - admin add via UI (decision: drift)
//   - admin remove via UI (decision: drift)
//   - Jamf-side strip on write (decision: no-op)
//   - user edits HCL (decision: apply)
//
// Each fixture mirrors the PayloadContent[i] shape of a PPPC profile —
// the same wire pattern that originally exposed the universal-fix bug
// with the DEVONthink Location strip.

const tw_two_services = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadIdentifier</key><string>top-id</string>
<key>PayloadUUID</key><string>top-uuid</string>
<key>PayloadContent</key><array><dict>
<key>PayloadType</key><string>com.apple.TCC.configuration-profile-policy</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadIdentifier</key><string>tcc-id</string>
<key>PayloadUUID</key><string>tcc-uuid</string>
<key>Services</key><dict>
<key>ScreenCapture</key><array><dict>
<key>Authorization</key><string>Allow</string>
<key>Identifier</key><string>com.example.app</string>
<key>CodeRequirement</key><string>req</string>
<key>IdentifierType</key><string>bundleID</string>
</dict></array>
<key>Accessibility</key><array><dict>
<key>Allowed</key><true/>
<key>Identifier</key><string>com.example.app</string>
<key>CodeRequirement</key><string>req</string>
<key>IdentifierType</key><string>bundleID</string>
</dict></array>
</dict>
</dict></array>
</dict></plist>`

const tw_one_service = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadIdentifier</key><string>top-id</string>
<key>PayloadUUID</key><string>top-uuid</string>
<key>PayloadContent</key><array><dict>
<key>PayloadType</key><string>com.apple.TCC.configuration-profile-policy</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadIdentifier</key><string>tcc-id</string>
<key>PayloadUUID</key><string>tcc-uuid</string>
<key>Services</key><dict>
<key>ScreenCapture</key><array><dict>
<key>Authorization</key><string>Allow</string>
<key>Identifier</key><string>com.example.app</string>
<key>CodeRequirement</key><string>req</string>
<key>IdentifierType</key><string>bundleID</string>
</dict></array>
</dict>
</dict></array>
</dict></plist>`

// tw_one_plus_location is the user's HCL input variant that authors an
// invalid TCC service (Location). The DEVONthink fixture proves Jamf
// silently strips Location on write — so on the first Apply, lastInput
// will hold this value while lastCanonical will hold tw_one_service.
const tw_one_plus_location = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadIdentifier</key><string>top-id</string>
<key>PayloadUUID</key><string>top-uuid</string>
<key>PayloadContent</key><array><dict>
<key>PayloadType</key><string>com.apple.TCC.configuration-profile-policy</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadIdentifier</key><string>tcc-id</string>
<key>PayloadUUID</key><string>tcc-uuid</string>
<key>Services</key><dict>
<key>ScreenCapture</key><array><dict>
<key>Authorization</key><string>Allow</string>
<key>Identifier</key><string>com.example.app</string>
<key>CodeRequirement</key><string>req</string>
<key>IdentifierType</key><string>bundleID</string>
</dict></array>
<key>Location</key><array><dict>
<key>Authorization</key><string>Allow</string>
<key>Identifier</key><string>com.example.app</string>
<key>CodeRequirement</key><string>req</string>
<key>IdentifierType</key><string>bundleID</string>
</dict></array>
</dict>
</dict></array>
</dict></plist>`

// TestThreeWayCompare_AdminAddedService asserts admin UI edit surfaces as
// drift: user HCL is unchanged but the server now has an Accessibility
// service entry that wasn't there at last Apply.
func TestThreeWayCompare_AdminAddedService(t *testing.T) {
	got, err := ThreeWayCompare(
		[]byte(tw_one_service),  // planInput     — user HCL, unchanged
		[]byte(tw_one_service),  // lastInput     — user HCL at last Apply
		[]byte(tw_one_service),  // lastCanonical — server immediately post-Apply
		[]byte(tw_two_services), // serverNow    — admin added Accessibility
	)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if got != DecisionDrift {
		t.Fatalf("admin-added service must decide Drift, got %d", got)
	}
}

// TestThreeWayCompare_AdminRemovedService asserts admin UI edit surfaces
// as drift: user HCL is unchanged but the server now has fewer service
// entries than at last Apply.
func TestThreeWayCompare_AdminRemovedService(t *testing.T) {
	got, err := ThreeWayCompare(
		[]byte(tw_two_services),
		[]byte(tw_two_services),
		[]byte(tw_two_services),
		[]byte(tw_one_service),
	)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if got != DecisionDrift {
		t.Fatalf("admin-removed service must decide Drift, got %d", got)
	}
}

// TestThreeWayCompare_JamfStripIgnored asserts the DEVONthink class of
// noise is suppressed: user authors Location, Jamf strips it on write, so
// lastInput holds the unstripped HCL and lastCanonical holds the stripped
// server response. On a subsequent plan-refresh where nothing else
// changed, the decision must be NoOp — not perpetual drift, not Apply.
func TestThreeWayCompare_JamfStripIgnored(t *testing.T) {
	got, err := ThreeWayCompare(
		[]byte(tw_one_plus_location), // planInput     — HCL still has Location
		[]byte(tw_one_plus_location), // lastInput     — same HCL last apply
		[]byte(tw_one_service),       // lastCanonical — Jamf stripped Location
		[]byte(tw_one_service),       // serverNow     — server unchanged
	)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if got != DecisionNoOp {
		t.Fatalf("Jamf-stripped key must decide NoOp, got %d", got)
	}
}

// TestThreeWayCompare_UserHCLChange asserts a genuine HCL edit propagates
// as Apply even when the stripped-key reference shape is present.
func TestThreeWayCompare_UserHCLChange(t *testing.T) {
	got, err := ThreeWayCompare(
		[]byte(tw_two_services),      // planInput     — user added Accessibility
		[]byte(tw_one_plus_location), // lastInput    — previous HCL had Location only
		[]byte(tw_one_service),       // lastCanonical
		[]byte(tw_one_service),       // serverNow
	)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if got != DecisionApply {
		t.Fatalf("user HCL change must decide Apply, got %d", got)
	}
}

// TestThreeWayCompare_AllAligned asserts the steady-state path: nothing
// has changed on either side since the last Apply.
func TestThreeWayCompare_AllAligned(t *testing.T) {
	got, err := ThreeWayCompare(
		[]byte(tw_one_plus_location),
		[]byte(tw_one_plus_location),
		[]byte(tw_one_service),
		[]byte(tw_one_service),
	)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if got != DecisionNoOp {
		t.Fatalf("steady state must decide NoOp, got %d", got)
	}
}

// TestStructuralEqual_NumericCrossType guards that howett.net/plist's
// int64/uint64/int trio compares equal across types for the same numeric
// value. Drift detection would false-positive without this.
func TestStructuralEqual_NumericCrossType(t *testing.T) {
	cases := []struct {
		name string
		a, b any
		want bool
	}{
		{"int64==uint64", int64(1), uint64(1), true},
		{"int==int64", 1, int64(1), true},
		{"int64!=int64", int64(1), int64(2), false},
		{"negative int64 vs uint64", int64(-1), uint64(1), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := structuralEqual(tc.a, tc.b); got != tc.want {
				t.Errorf("structuralEqual(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
