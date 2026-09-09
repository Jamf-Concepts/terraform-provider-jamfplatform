// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package payloadhelpers

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

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
			if got := structuralEqual(tc.a, tc.b, true); got != tc.want {
				t.Errorf("structuralEqual(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

// TestPayloadsStructurallyEqual_ReorderedPayloadContent covers the strict
// comparator's share of the reorder problem. structuralEqual walks arrays
// positionally too, and both of its operands are usually server-canonical — so
// both carry Jamf Pro's ordering and agree. The exception is the plan modifier's
// fallback when payload_server_now is absent from private state: the drift arm
// then compares the last-applied server canonical against state.General.Payloads,
// which lenient self-healing keeps in the *user-authored* order. On a profile
// mixing storage categories that is a guaranteed false DecisionDrift and a plan
// that never converges.
//
// MaskPayload canonicalises PayloadContent order for both comparators, so this
// holds without structuralEqual needing a special case of its own.
func TestPayloadsStructurallyEqual_ReorderedPayloadContent(t *testing.T) {
	equal, err := PayloadsStructurallyEqual([]byte(mcxThenCert), []byte(certThenMCXSameContent))
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if !equal {
		t.Fatal("entry order must not make two otherwise identical payloads structurally unequal")
	}
}

// TestPayloadsStructurallyEqual_ReorderedWithRealDrift keeps the strict
// comparator strict: order is ignored, values are not.
func TestPayloadsStructurallyEqual_ReorderedWithRealDrift(t *testing.T) {
	equal, err := PayloadsStructurallyEqual([]byte(mcxThenCert), []byte(certThenMCXChangedInnerValue))
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if equal {
		t.Fatal("a changed value inside a relocated entry must still compare unequal")
	}
}

// twWebClipPayload wraps one web clip icon blob in the minimal profile shape
// MaskPayload accepts: a top-level Configuration dict carrying a single
// com.apple.webClip.managed entry whose Icon is the given PNG.
func twWebClipPayload(t *testing.T, iconPNG []byte) []byte {
	t.Helper()
	return fmt.Appendf(nil, `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadIdentifier</key><string>top-id</string>
<key>PayloadUUID</key><string>top-uuid</string>
<key>PayloadContent</key><array><dict>
<key>PayloadType</key><string>com.apple.webClip.managed</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadIdentifier</key><string>clip-id</string>
<key>PayloadUUID</key><string>clip-uuid</string>
<key>Label</key><string>Example</string>
<key>URL</key><string>https://example.com</string>
<key>Icon</key><data>%s</data>
</dict></array>
</dict></plist>`, base64.StdEncoding.EncodeToString(iconPNG))
}

// twIconFixture reads one of the four PNG pairs the icon tolerance was measured
// against. A local reader rather than webclipicon_test.go's helper keeps the two
// test files independent.
func twIconFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("reading fixture %s: %v", name, err)
	}
	return b
}

// twGradientIcon is a 180x180 diagonal gradient — the "before" side of the
// authored-edit pair below.
func twGradientIcon(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 180, 180))
	for y := range 180 {
		for x := range 180 {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 255 / 180), G: uint8(y * 255 / 180), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encoding gradient: %v", err)
	}
	return buf.Bytes()
}

// twBadgedGradientIcon is twGradientIcon with a full-contrast 22x22 badge
// painted into one corner — 1.5% of the area, which is a change an operator
// authored deliberately and yet normalises to a mean delta well inside
// iconMeanDeltaTolerance.
func twBadgedGradientIcon(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 180, 180))
	for y := range 180 {
		for x := range 180 {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 255 / 180), G: uint8(y * 255 / 180), B: 128, A: 255})
		}
	}
	draw.Draw(img, image.Rect(158, 158, 180, 180), image.NewUniform(color.NRGBA{R: 31, G: 31, B: 127, A: 255}), image.Point{}, draw.Src)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encoding badged gradient: %v", err)
	}
	return buf.Bytes()
}

// TestThreeWayCompare_AuthoredIconEditWithinToleranceApplies pins the arm the
// icon tolerance must never reach. planInput and lastInput are both user HCL,
// so a difference between them is an authored edit no matter how small the
// two icons' normalised delta is. The pair is deliberately chosen to be inside
// iconMeanDeltaTolerance — asserted below, so the test cannot pass vacuously —
// which is exactly the case that decided NoOp before the tolerance was scoped
// to the server-facing arm, leaving the edit unappliable and unreported.
func TestThreeWayCompare_AuthoredIconEditWithinToleranceApplies(t *testing.T) {
	before, after := twGradientIcon(t), twBadgedGradientIcon(t)
	if bytes.Equal(before, after) {
		t.Fatal("fixture icons must differ in bytes")
	}
	if !iconsEquivalent(before, after) {
		t.Fatal("fixture icons must be within iconMeanDeltaTolerance, or this test proves nothing")
	}

	authoredBefore := twWebClipPayload(t, before)
	authoredAfter := twWebClipPayload(t, after)

	got, err := ThreeWayCompare(authoredAfter, authoredBefore, authoredBefore, authoredBefore)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if got != DecisionApply {
		t.Fatalf("an authored icon edit must decide Apply, got %d", got)
	}
}

// TestThreeWayCompare_ServerIconRerenderIsNoOp is the other half of the same
// rule: the drift arm compares two server-facing values, so Jamf Pro's
// re-render of an unchanged icon must still decide NoOp.
func TestThreeWayCompare_ServerIconRerenderIsNoOp(t *testing.T) {
	authored := twWebClipPayload(t, twIconFixture(t, "webclip_icon_authored_a.png"))
	rendered := twWebClipPayload(t, twIconFixture(t, "webclip_icon_jamf_rendered_a.png"))

	got, err := ThreeWayCompare(authored, authored, authored, rendered)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if got != DecisionNoOp {
		t.Fatalf("a server-side icon re-render must decide NoOp, got %d", got)
	}
}

// TestPayloadsStructurallyEqual_ServerIconRerender covers the Read-side drift
// detector, whose operands are both server responses and which therefore keeps
// the tolerance.
func TestPayloadsStructurallyEqual_ServerIconRerender(t *testing.T) {
	authored := twWebClipPayload(t, twIconFixture(t, "webclip_icon_authored_b.png"))
	rendered := twWebClipPayload(t, twIconFixture(t, "webclip_icon_jamf_rendered_b.png"))

	equal, err := PayloadsStructurallyEqual(authored, rendered)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if !equal {
		t.Fatal("Jamf Pro's re-render of an unchanged icon must not read as drift")
	}
}
