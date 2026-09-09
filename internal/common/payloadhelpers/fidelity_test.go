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
	"strings"
	"testing"
)

// The mSCP-style case that drove this rewrite: a decoratively line-wrapped
// ConsentText, which the server stores with every line feed and indent tab
// deleted so the words either side merge.
const (
	mscpAuthoredConsent = "THE SOFTWARE IS PROVIDED 'AS IS' WITHOUT ANY WARRANTY OF ANY KIND, EITHER\n\t\t\t\tEXPRESSED, IMPLIED, OR STATUTORY, INCLUDING, BUT NOT LIMITED TO, ANY WARRANTY THAT\n\t\t\t\tTHE SOFTWARE WILL CONFORM TO SPECIFICATIONS."
	mscpStoredConsent   = "THE SOFTWARE IS PROVIDED 'AS IS' WITHOUT ANY WARRANTY OF ANY KIND, EITHEREXPRESSED, IMPLIED, OR STATUTORY, INCLUDING, BUT NOT LIMITED TO, ANY WARRANTY THATTHE SOFTWARE WILL CONFORM TO SPECIFICATIONS."
)

func TestPayloadFidelityErrorDetail_LineBreakClass(t *testing.T) {
	got := PayloadFidelityErrorDetail(
		consentTextWith(t, mscpAuthoredConsent),
		consentTextWith(t, mscpStoredConsent),
		FidelityPhaseCreate,
	)
	mustContain(t, got, "ConsentText.default")
	mustContain(t, got, "line breaks removed")
	mustContain(t, got, "&#13;")
	mustContain(t, got, "&#8232;")
	// The excerpt must make the invisible characters visible.
	mustContain(t, got, `\n\t\t\t\t`)
	// This payload holds no "&" or "<" — the old blanket text blamed PI-827
	// here and told the reader to remove characters that were not present.
	mustNotContain(t, got, "PI-827")
}

func TestPayloadFidelityErrorDetail_EntityLayerClass(t *testing.T) {
	got := PayloadFidelityErrorDetail(
		consentTextWith(t, "Here is an &amp; ok"),
		consentTextWith(t, "Here is an &amp;amp; ok"),
		FidelityPhaseCreate,
	)
	mustContain(t, got, "ConsentText.default")
	mustContain(t, got, "PI-827")
	mustNotContain(t, got, "line breaks removed")
}

func TestPayloadFidelityErrorDetail_AstralClass(t *testing.T) {
	got := PayloadFidelityErrorDetail(
		consentTextWith(t, "release party 🎉"),
		consentTextWith(t, "release party \uFFFD\uFFFD"),
		FidelityPhaseCreate,
	)
	mustContain(t, got, "basic multilingual plane")
	mustNotContain(t, got, "PI-827")
}

func TestPayloadFidelityErrorDetail_DroppedValue(t *testing.T) {
	// Stored side has no ConsentText at all — the shape Jamf Pro produces when
	// it drops a whole dictionary rather than mangling a value in place.
	got := PayloadFidelityErrorDetail(
		consentTextWith(t, "some agreement text"),
		[]byte(minimalPlist),
		FidelityPhaseCreate,
	)
	mustContain(t, got, "ConsentText.default")
	mustContain(t, got, "not stored at all")
	mustContain(t, got, "(nothing — the value is absent)")
}

func TestPayloadFidelityErrorDetail_MaskedKeysAreNotBlamed(t *testing.T) {
	// Jamf Pro rewrites the identifiers and display name on every write. Those
	// are never why verification failed, so the detail must fall back to the
	// unattributed text rather than naming PayloadUUID as the culprit.
	const stored = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadUUID</key><string>99999999-9999-9999-9999-999999999999</string>
<key>PayloadIdentifier</key><string>99999999-9999-9999-9999-999999999999</string>
<key>PayloadDisplayName</key><string>Renamed By Jamf</string>
<key>PayloadContent</key><array/>
</dict></plist>`
	got := PayloadFidelityErrorDetail([]byte(minimalPlist), []byte(stored), FidelityPhaseCreate)
	mustContain(t, got, "could not attribute the difference to a single value")
	mustNotContain(t, got, "PayloadUUID")
	mustNotContain(t, got, "PayloadDisplayName")
}

func TestPayloadFidelityErrorDetail_PhaseTails(t *testing.T) {
	authored := consentTextWith(t, mscpAuthoredConsent)
	stored := consentTextWith(t, mscpStoredConsent)

	create := PayloadFidelityErrorDetail(authored, stored, FidelityPhaseCreate)
	mustContain(t, create, "has been rolled back")

	update := PayloadFidelityErrorDetail(authored, stored, FidelityPhaseUpdate)
	mustContain(t, update, "no longer matches this configuration")
	mustNotContain(t, update, "rolled back")
}

func TestPayloadFidelityErrorDetail_UnparseableFallsBack(t *testing.T) {
	got := PayloadFidelityErrorDetail([]byte("<dict><key>unclosed"), []byte(minimalPlist), FidelityPhaseCreate)
	mustContain(t, got, "could not attribute the difference to a single value")
	// The fallback still has to teach the three fixes.
	mustContain(t, got, "&#13;")
	mustContain(t, got, "PI-827")
	mustContain(t, got, "basic multilingual plane")
}

func TestPayloadFidelityErrorDetail_CapsTheReport(t *testing.T) {
	authored := manyValuePlist(t, "line one\nline two")
	stored := manyValuePlist(t, "line oneline two")
	got := PayloadFidelityErrorDetail(authored, stored, FidelityPhaseCreate)
	mustContain(t, got, "Jamf Pro stored 5 payload values differently")
	mustContain(t, got, "2 further value(s) also differ")
	if n := strings.Count(got, "line breaks removed"); n != maxReportedFindings {
		t.Errorf("named %d findings, want the %d cap", n, maxReportedFindings)
	}
}

func TestPayloadFidelityErrorDetail_SingularWording(t *testing.T) {
	got := PayloadFidelityErrorDetail(
		consentTextWith(t, mscpAuthoredConsent),
		consentTextWith(t, mscpStoredConsent),
		FidelityPhaseCreate,
	)
	mustContain(t, got, "stored a payload value differently")
	mustNotContain(t, got, "1 payload values")
}

func TestExcerpt_WindowsOnFirstDifference(t *testing.T) {
	a := strings.Repeat("A", 200) + "DIFFERENT" + strings.Repeat("B", 200)
	b := strings.Repeat("A", 200) + "same" + strings.Repeat("B", 200)
	got := excerpt(a, b)
	mustContain(t, got, "DIFFERENT")
	if !strings.HasPrefix(got, "…") || !strings.HasSuffix(got, "…") {
		t.Errorf("expected elision markers both ends: %s", got)
	}
	if len(got) > 140 {
		t.Errorf("excerpt too long (%d bytes): %s", len(got), got)
	}
}

func TestExcerpt_KeepsRuneBoundaries(t *testing.T) {
	// A multi-byte rune straddling the window edge must not be split into
	// invalid UTF-8 by the quoting.
	a := strings.Repeat("é", 120) + "X"
	b := strings.Repeat("é", 120) + "Y"
	got := excerpt(a, b)
	mustNotContain(t, got, `\x`)
}

func TestClassify_CarriageReturnIsNotALineBreakFailure(t *testing.T) {
	// A value whose only whitespace is CR round-trips (Jamf Pro keeps it), so a
	// divergence there is not the line-break class and must not recommend the
	// representation the payload already uses.
	if c := classify("line one\rline two", "line one\rline twoX", true); c == classLineBreak {
		t.Error("CR-only value classified as a line-break failure")
	}
}

func manyValuePlist(t *testing.T, value string) []byte {
	t.Helper()
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
`)
	for i := range 5 {
		fmt.Fprintf(&b, "<key>Banner%d</key><string>%s</string>\n", i, value)
	}
	b.WriteString("</dict></plist>")
	return []byte(b.String())
}

func mustContain(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Errorf("detail missing %q:\n%s", want, got)
	}
}

func mustNotContain(t *testing.T, got, unwanted string) {
	t.Helper()
	if strings.Contains(got, unwanted) {
		t.Errorf("detail unexpectedly contains %q:\n%s", unwanted, got)
	}
}

// Wire-observed 2026-08-11 against Jamf Pro 11.30.x: an authored
// [MCX, loginwindow] profile comes back as [loginwindow, MCX], because a
// loginwindow payload is stored verbatim while MCX is re-rendered and Jamf Pro
// puts the verbatim block first. The loginwindow value also loses its line feed,
// so this pair carries a reorder and exactly one real defect at the same time.
const (
	reorderedAuthored = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadIdentifier</key><string>top-id</string>
<key>PayloadUUID</key><string>top-uuid</string>
<key>PayloadContent</key><array>
<dict>
<key>PayloadType</key><string>com.apple.ManagedClient.preferences</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadContent</key><dict>
<key>com.example.browser</key><dict>
<key>Forced</key><array><dict>
<key>mcx_preference_settings</key><dict>
<key>HomepageLocation</key><string>https://example.com/start</string>
</dict>
</dict></array>
</dict>
</dict>
</dict>
<dict>
<key>PayloadType</key><string>com.apple.loginwindow</string>
<key>PayloadVersion</key><integer>1</integer>
<key>LoginwindowText</key><string>FIRST LINE
SECOND LINE</string>
</dict>
</array>
</dict></plist>`

	reorderedStored = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadIdentifier</key><string>top-id</string>
<key>PayloadUUID</key><string>top-uuid</string>
<key>PayloadContent</key><array>
<dict>
<key>PayloadType</key><string>com.apple.loginwindow</string>
<key>PayloadVersion</key><integer>1</integer>
<key>LoginwindowText</key><string>FIRST LINESECOND LINE</string>
</dict>
<dict>
<key>PayloadType</key><string>com.apple.ManagedClient.preferences</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadContent</key><dict>
<key>com.example.browser</key><dict>
<key>Forced</key><array><dict>
<key>mcx_preference_settings</key><dict>
<key>HomepageLocation</key><string>https://example.com/start</string>
</dict>
</dict></array>
</dict>
</dict>
</dict>
</array>
</dict></plist>`
)

// TestPayloadFidelityErrorDetail_ReorderedEntriesBlameOnlyTheRealValue is the
// regression guard for the diagnostic half of the reorder bug. The paths this
// message quotes are indexed, so when Jamf Pro moves an entry the authored and
// stored sides disagree about which entry index 1 is. Without
// alignPayloadContentOrder this pair produced four findings — a value that stored
// perfectly reported as "not stored at all", a phantom PayloadType mismatch, the
// real defect misclassified as absent, and the actual line-break finding pushed
// past maxReportedFindings and never shown to the operator.
func TestPayloadFidelityErrorDetail_ReorderedEntriesBlameOnlyTheRealValue(t *testing.T) {
	got := PayloadFidelityErrorDetail([]byte(reorderedAuthored), []byte(reorderedStored), FidelityPhaseCreate)

	// Exactly one value differs, and it is named at the index the operator wrote it at.
	mustContain(t, got, "Jamf Pro stored a payload value differently")
	mustContain(t, got, "PayloadContent[1].LoginwindowText")
	mustContain(t, got, "line breaks removed")
	mustContain(t, got, `"FIRST LINE\nSECOND LINE"`)

	// None of the reorder artefacts may appear.
	mustNotContain(t, got, "further value(s) also differ")
	mustNotContain(t, got, "not stored at all")
	mustNotContain(t, got, "HomepageLocation")
	mustNotContain(t, got, "com.apple.ManagedClient.preferences")
}

// TestPayloadFidelityErrorDetail_ReorderAloneIsNotReported covers the case that
// actually reached a user: the array was reordered and nothing else changed, so
// the diff must be silent. The equality gate suppresses this before the
// diagnostic is reached, but the two must not be able to disagree.
func TestPayloadFidelityErrorDetail_ReorderAloneIsNotReported(t *testing.T) {
	authored, stored := []byte(reorderedAuthored), []byte(reorderedStored)
	// Same pair with the line feed left intact on the stored side.
	stored = []byte(strings.Replace(string(stored), "FIRST LINESECOND LINE", "FIRST LINE\nSECOND LINE", 1))

	findings, ok := diffPayloadStrings(authored, stored)
	if !ok {
		t.Fatal("pair did not parse")
	}
	if len(findings) != 0 {
		for _, f := range findings {
			t.Logf("unexpected finding: %s (present=%v)", f.path, f.present)
		}
		t.Errorf("a reorder with no value change must produce no findings, got %d", len(findings))
	}
}

// fidelitySolidPNG is a decodable single-colour PNG, the smallest icon this
// package's comparison can tell apart from another: two solid icons of
// different colours are far past iconMeanDeltaTolerance however Jamf Pro
// rescales them.
func fidelitySolidPNG(t *testing.T, c color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encoding icon: %v", err)
	}
	return buf.Bytes()
}

// fidelityWebClipProfile wraps one web clip entry carrying the supplied icon, plus any
// extra PayloadContent entries given verbatim.
func fidelityWebClipProfile(icon []byte, extraEntries string) []byte {
	return []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadContent</key><array>
<dict>
<key>PayloadType</key><string>com.apple.webClip.managed</string>
<key>PayloadVersion</key><integer>1</integer>
<key>Label</key><string>Handbook</string>
<key>URL</key><string>https://example.com/handbook</string>
<key>Icon</key><data>` + base64.StdEncoding.EncodeToString(icon) + `</data>
</dict>` + extraEntries + `
</array>
</dict></plist>`)
}

// removalPasswordEntry is the PayloadContent entry Jamf Pro materialises when
// the resource sets self_service.authorization_password — nothing the operator
// authored, so it is dropped from the stored side before the diff.
const removalPasswordEntry = `
<dict>
<key>PayloadType</key><string>com.apple.profileRemovalPassword</string>
<key>PayloadVersion</key><integer>1</integer>
<key>RemovalPassword</key><string>secret</string>
</dict>`

// TestDiffPayloadTrees_InjectedEntryDoesNotHideTheRealFinding pins the fix for
// the length-mismatch shortcut. A server-injected entry makes the stored
// PayloadContent one longer than the authored one, and the array branch of
// diffNonStringLeaves reports a length mismatch *without recursing* — so it
// replaced the attribution rather than adding to it, and a genuinely diverging
// icon on the same profile was never named.
func TestDiffPayloadTrees_InjectedEntryDoesNotHideTheRealFinding(t *testing.T) {
	authored := fidelityWebClipProfile(fidelitySolidPNG(t, color.RGBA{R: 0, G: 0, B: 0, A: 255}), "")
	stored := fidelityWebClipProfile(fidelitySolidPNG(t, color.RGBA{R: 255, G: 255, B: 255, A: 255}), removalPasswordEntry)

	equal, err := PayloadsSemanticallyEqual(authored, stored)
	if err != nil {
		t.Fatalf("comparing payloads: %v", err)
	}
	if equal {
		t.Fatal("fixture does not diverge under the equality check, so the differ has nothing to attribute")
	}

	got := PayloadFidelityErrorDetail(authored, stored, FidelityPhaseCreate)
	mustContain(t, got, "PayloadContent[0].Icon")
	mustContain(t, got, "stored as a different picture")
	mustNotContain(t, got, "a list of 1 item")
	mustNotContain(t, got, "a list of 2 items")
	mustNotContain(t, got, "RemovalPassword")
}

// TestDiffPayloadTrees_InjectedEntryAloneIsNotReported is the other half: the
// injected entry is the *only* difference, so the diff must be silent — the
// filter has to match MaskPayload's exactly or the two disagree.
func TestDiffPayloadTrees_InjectedEntryAloneIsNotReported(t *testing.T) {
	icon := fidelitySolidPNG(t, color.RGBA{R: 12, G: 34, B: 56, A: 255})
	authored := fidelityWebClipProfile(icon, "")
	stored := fidelityWebClipProfile(icon, removalPasswordEntry)

	findings, ok := diffPayloadStrings(authored, stored)
	if !ok {
		t.Fatal("pair did not parse")
	}
	if len(findings) != 0 {
		for _, f := range findings {
			t.Logf("unexpected finding: %s (present=%v)", f.path, f.present)
		}
		t.Errorf("an injected entry with no value change must produce no findings, got %d", len(findings))
	}
}

// dictVersusScalar authors a dictionary of non-string leaves where the stored
// side holds a scalar. Non-string deliberately: a string leaf inside the
// authored dict would be caught by the flatten pass, which is not the branch
// under test.
const (
	dictVersusScalarAuthored = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>Extras</key><dict><key>Count</key><integer>1</integer></dict>
</dict></plist>`

	dictVersusScalarStored = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>Extras</key><integer>1</integer>
</dict></plist>`
)

// TestDiffPayloadTrees_DictAgainstNonDictIsBlamed covers the branch that used
// to return nil for a whole subtree: LenientEqualPlist fails a dictionary
// facing a non-dictionary outright, and the array branch beside it already
// reported its own type mismatch, so the two disagreed with each other and the
// failure fell through to the unattributed text.
func TestDiffPayloadTrees_DictAgainstNonDictIsBlamed(t *testing.T) {
	authored, stored := []byte(dictVersusScalarAuthored), []byte(dictVersusScalarStored)

	equal, err := PayloadsSemanticallyEqual(authored, stored)
	if err != nil {
		t.Fatalf("comparing payloads: %v", err)
	}
	if equal {
		t.Fatal("fixture does not diverge under the equality check, so the differ has nothing to attribute")
	}

	got := PayloadFidelityErrorDetail(authored, stored, FidelityPhaseCreate)
	mustContain(t, got, "Extras")
	mustContain(t, got, "a dictionary of 1 key")
	mustNotContain(t, got, "could not attribute the difference")
}

// fidelityMCXProfile wraps one "Application & Custom Settings" entry whose inner
// preference dictionary holds only booleans — no string leaf, so the flatten
// pass cannot attribute anything here and the MCX rule is the only thing that
// can.
func fidelityMCXProfile(preferences string) []byte {
	return []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadContent</key><array>
<dict>
<key>PayloadType</key><string>com.apple.ManagedClient.preferences</string>
<key>PayloadVersion</key><integer>1</integer>` + preferences + `
</dict>
</array>
</dict></plist>`)
}

func fidelityMCXPreferences(settings string) string {
	return `
<key>PayloadContent</key><dict>
<key>com.example.browser</key><dict>
<key>Forced</key><array><dict>
<key>mcx_preference_settings</key><dict>` + settings + `</dict>
</dict></array>
</dict>
</dict>`
}

// TestDiffPayloadTrees_MCXPreferencesDroppedWholesaleIsBlamed is the second
// half of issue #418: LenientEqualPlist's MCX branch fails on the *presence*
// of the inner PayloadContent, precisely because a one-sided vendor preference
// key is real drift, while the differ's intersection walk skipped the key
// entirely and left the diagnostic listing line feeds and ampersands the
// payload does not contain.
func TestDiffPayloadTrees_MCXPreferencesDroppedWholesaleIsBlamed(t *testing.T) {
	authored := fidelityMCXProfile(fidelityMCXPreferences(`<key>SafeBrowsingEnabled</key><true/>`))
	stored := fidelityMCXProfile("")

	equal, err := PayloadsSemanticallyEqual(authored, stored)
	if err != nil {
		t.Fatalf("comparing payloads: %v", err)
	}
	if equal {
		t.Fatal("fixture does not diverge under the equality check, so the differ has nothing to attribute")
	}

	got := PayloadFidelityErrorDetail(authored, stored, FidelityPhaseCreate)
	mustContain(t, got, "PayloadContent[0].PayloadContent")
	mustContain(t, got, "not stored at all")
	mustNotContain(t, got, "could not attribute the difference")
}

// TestDiffPayloadTrees_MCXPreferenceKeyDroppedIsBlamed covers the same rule one
// level down. plisthelpers.Equal requires matching keysets at every depth of
// the preference subtree, so a single key Jamf Pro did not store fails the
// comparison — and an intersection walk steps straight over it.
func TestDiffPayloadTrees_MCXPreferenceKeyDroppedIsBlamed(t *testing.T) {
	authored := fidelityMCXProfile(fidelityMCXPreferences(`<key>SafeBrowsingEnabled</key><true/><key>PasswordManagerEnabled</key><false/>`))
	stored := fidelityMCXProfile(fidelityMCXPreferences(`<key>SafeBrowsingEnabled</key><true/>`))

	equal, err := PayloadsSemanticallyEqual(authored, stored)
	if err != nil {
		t.Fatalf("comparing payloads: %v", err)
	}
	if equal {
		t.Fatal("fixture does not diverge under the equality check, so the differ has nothing to attribute")
	}

	got := PayloadFidelityErrorDetail(authored, stored, FidelityPhaseCreate)
	mustContain(t, got, "mcx_preference_settings.PasswordManagerEnabled")
	mustContain(t, got, "not stored at all")
	mustNotContain(t, got, "could not attribute the difference")
}

// TestDiffPayloadTrees_MCXPreferencesUnchangedAreSilent guards the other
// direction: an identical preference subtree must produce nothing, so the new
// rule cannot invent drift on every MCX payload in the corpus.
func TestDiffPayloadTrees_MCXPreferencesUnchangedAreSilent(t *testing.T) {
	prefs := fidelityMCXPreferences(`<key>SafeBrowsingEnabled</key><true/>`)
	findings, ok := diffPayloadStrings(fidelityMCXProfile(prefs), fidelityMCXProfile(prefs))
	if !ok {
		t.Fatal("pair did not parse")
	}
	if len(findings) != 0 {
		for _, f := range findings {
			t.Logf("unexpected finding: %s (present=%v)", f.path, f.present)
		}
		t.Errorf("an unchanged MCX payload must produce no findings, got %d", len(findings))
	}
}
