// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package payloadhelpers

import (
	"fmt"
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
