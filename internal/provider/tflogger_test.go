// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"encoding/xml"
	"net/http"
	"strings"
	"testing"
)

// render is a test helper: renders a body and fails if it was withheld.
func render(t *testing.T, in []byte) string {
	t.Helper()
	got, ok := redactAndFormat(in)
	if !ok {
		t.Fatalf("body unexpectedly withheld: %q", in)
	}
	return got
}

// escapeXMLText escapes a string the way it would appear as XML character data,
// so tests can embed a plist inside an element the same way Jamf Pro does.
func escapeXMLText(t *testing.T, s string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := xml.EscapeText(&buf, []byte(s)); err != nil {
		t.Fatalf("escaping test fixture: %v", err)
	}
	return buf.String()
}

// --- formatting ------------------------------------------------------------

func TestRedactAndFormat_JSONObject(t *testing.T) {
	got := render(t, []byte(`{"name":"foo","id":1,"tags":["a","b"]}`))
	if !strings.Contains(got, "\n  \"name\": \"foo\"") {
		t.Errorf("JSON not indented:\n%s", got)
	}
	if !strings.Contains(got, "\n  \"tags\": [") {
		t.Errorf("JSON nested array not indented:\n%s", got)
	}
}

func TestRedactAndFormat_JSONArray(t *testing.T) {
	got := render(t, []byte(`[{"a":1},{"a":2}]`))
	if !strings.HasPrefix(got, "[\n  {") {
		t.Errorf("JSON array root not indented:\n%s", got)
	}
}

func TestRedactAndFormat_XMLBasic(t *testing.T) {
	in := []byte(`<user_group><id>3</id><name>Excluded Users</name><is_smart>false</is_smart></user_group>`)
	got := render(t, in)
	for _, want := range []string{
		"<user_group>\n  <id>3</id>",
		"  <name>Excluded Users</name>",
		"  <is_smart>false</is_smart>",
		"</user_group>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("XML missing fragment %q:\n%s", want, got)
		}
	}
}

func TestRedactAndFormat_XMLNested(t *testing.T) {
	in := []byte(`<user_group><criteria><size>1</size><criterion><name>User Group</name><value>x</value></criterion></criteria></user_group>`)
	got := render(t, in)
	if !strings.Contains(got, "    <criterion>\n      <name>User Group</name>") {
		t.Errorf("XML nesting not indented to 4 spaces:\n%s", got)
	}
}

func TestRedactAndFormat_LeadingWhitespace(t *testing.T) {
	got := render(t, []byte("\n\t  <foo><bar/></foo>"))
	if !strings.Contains(got, "<foo>\n  <bar></bar>") {
		t.Errorf("leading whitespace tripped XML sniff:\n%s", got)
	}
}

// TestRedactJSON_PreservesMemberOrder guards the reason redaction streams tokens
// instead of round-tripping through map[string]any: Jamf Pro rejects some
// classic payloads whose members arrive out of order, so a debug log that
// silently re-sorted them would misdirect that exact investigation.
func TestRedactJSON_PreservesMemberOrder(t *testing.T) {
	got := render(t, []byte(`{"zebra":1,"apple":2,"mango":3}`))
	zebra := strings.Index(got, "zebra")
	apple := strings.Index(got, "apple")
	mango := strings.Index(got, "mango")
	if zebra >= apple || apple >= mango {
		t.Errorf("member order not preserved (zebra=%d apple=%d mango=%d):\n%s", zebra, apple, mango, got)
	}
}

// TestRedactJSON_PreservesNumberLiterals guards dec.UseNumber(): the default
// float64 decode would rewrite a large ID and drop a trailing zero.
func TestRedactJSON_PreservesNumberLiterals(t *testing.T) {
	got := render(t, []byte(`{"big":12345678901234567890,"exact":1.10}`))
	for _, want := range []string{"12345678901234567890", "1.10"} {
		if !strings.Contains(got, want) {
			t.Errorf("number literal %q not preserved:\n%s", want, got)
		}
	}
}

// --- JSON redaction --------------------------------------------------------

func TestRedactJSON_RedactsSecretScalars(t *testing.T) {
	in := []byte(`{"username":"svc-bind","password":"hunter2","clientSecret":"cs-abc","token":"tok-xyz"}`)
	got := render(t, in)

	for _, leak := range []string{"hunter2", "cs-abc", "tok-xyz"} {
		if strings.Contains(got, leak) {
			t.Errorf("secret %q leaked:\n%s", leak, got)
		}
	}
	if !strings.Contains(got, `"username": "svc-bind"`) {
		t.Errorf("non-secret username should survive:\n%s", got)
	}
	if n := strings.Count(got, redactedPlaceholder); n != 3 {
		t.Errorf("expected 3 redactions, got %d:\n%s", n, got)
	}
}

// TestRedactJSON_KeepsNonSecretLookalikes is the point of exact-match matching:
// every field here contains "password", "key" or "token" but none is a secret,
// and several are exactly what DEBUG was enabled to inspect.
func TestRedactJSON_KeepsNonSecretLookalikes(t *testing.T) {
	in := []byte(`{
	  "passwordMinLength": 12,
	  "passwordMaxAge": 90,
	  "passwordHistoryDepth": 5,
	  "passcodePresent": true,
	  "passed": false,
	  "passPercentage": 80,
	  "keyUsage": "digitalSignature",
	  "groupRdnKey": "cn",
	  "triggerKey": "startup",
	  "tokenExpiration": "2026-01-01",
	  "bootstrapTokenEscrowed": true,
	  "vppTokenEnabled": false,
	  "keystoreFileName": "sso.jks"
	}`)
	got := render(t, in)
	if strings.Contains(got, redactedPlaceholder) {
		t.Errorf("non-secret lookalike field was redacted:\n%s", got)
	}
}

func TestRedactJSON_RedactsWholeSubtreeUnderSecretKey(t *testing.T) {
	in := []byte(`{"keystore":{"bytes":"MIIKk...","alias":"jamf"},"name":"sso"}`)
	got := render(t, in)

	for _, leak := range []string{"MIIKk", "alias", "jamf"} {
		if strings.Contains(got, leak) {
			t.Errorf("nested value %q survived under a secret key:\n%s", leak, got)
		}
	}
	if !strings.Contains(got, `"keystore": "REDACTED"`) {
		t.Errorf("expected whole keystore subtree redacted:\n%s", got)
	}
	if !strings.Contains(got, `"name": "sso"`) {
		t.Errorf("sibling should survive:\n%s", got)
	}
}

func TestRedactJSON_CaseInsensitiveFieldMatch(t *testing.T) {
	in := []byte(`{"Password":"a","PASSWORD":"b","password":"c"}`)
	got := render(t, in)
	for _, leak := range []string{`"a"`, `"b"`, `"c"`} {
		if strings.Contains(got, leak) {
			t.Errorf("value %s leaked despite case-insensitive match:\n%s", leak, got)
		}
	}
}

func TestRedactJSON_RedactsInsideArrays(t *testing.T) {
	in := []byte(`{"accounts":[{"username":"a","password":"p1"},{"username":"b","password":"p2"}]}`)
	got := render(t, in)
	for _, leak := range []string{"p1", "p2"} {
		if strings.Contains(got, leak) {
			t.Errorf("secret %q inside array leaked:\n%s", leak, got)
		}
	}
	if n := strings.Count(got, redactedPlaceholder); n != 2 {
		t.Errorf("expected 2 redactions, got %d:\n%s", n, got)
	}
}

// --- XML redaction ---------------------------------------------------------

func TestRedactXML_RedactsSensitiveElements(t *testing.T) {
	in := []byte(`<directory_binding><name>AD</name><username>svc</username><password>hunter2</password><password_sha256>deadbeef</password_sha256></directory_binding>`)
	got := render(t, in)

	for _, leak := range []string{"hunter2", "deadbeef"} {
		if strings.Contains(got, leak) {
			t.Errorf("secret %q leaked:\n%s", leak, got)
		}
	}
	if !strings.Contains(got, "<name>AD</name>") || !strings.Contains(got, "<username>svc</username>") {
		t.Errorf("non-secret elements should survive:\n%s", got)
	}
	if !strings.Contains(got, "<password>REDACTED</password>") {
		t.Errorf("expected redacted password element:\n%s", got)
	}
}

// --- plist-in-XML redaction ------------------------------------------------

func TestRedactXML_RedactsPlistPayloadSecrets(t *testing.T) {
	inner := `<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict>` +
		`<key>PayloadIdentifier</key><string>com.jamf.wifi</string>` +
		`<key>SSID_STR</key><string>CorpWiFi</string>` +
		`<key>Password</key><string>hunter2</string>` +
		`</dict></plist>`
	in := []byte(`<configuration_profile><general><name>WiFi</name><payloads>` +
		escapeXMLText(t, inner) + `</payloads></general></configuration_profile>`)

	got := render(t, in)

	if strings.Contains(got, "hunter2") {
		t.Errorf("plist payload password leaked:\n%s", got)
	}
	if !strings.Contains(got, redactedPlaceholder) {
		t.Errorf("expected a redaction inside the payload:\n%s", got)
	}
	for _, want := range []string{"PayloadIdentifier", "com.jamf.wifi", "SSID_STR", "CorpWiFi"} {
		if !strings.Contains(got, want) {
			t.Errorf("non-secret payload key %q was lost:\n%s", want, got)
		}
	}
}

// TestRedactPlist_PayloadContentArrayVsBlob covers the one key that means two
// different things: at the top level PayloadContent is the ARRAY of payloads
// (must be walked, not blanked); inside a certificate payload it is the raw
// PKCS#12 bytes (must be redacted).
func TestRedactPlist_PayloadContentArrayVsBlob(t *testing.T) {
	inner := `<plist version="1.0"><dict>` +
		`<key>PayloadContent</key><array><dict>` +
		`<key>PayloadType</key><string>com.apple.security.pkcs12</string>` +
		`<key>PayloadContent</key><data>TUlJS2t3SUJBekNDQ2s4</data>` +
		`</dict></array>` +
		`</dict></plist>`

	got := redactPlist([]byte(inner))

	if strings.Contains(got, "TUlJS2t3SUJBekNDQ2s4") {
		t.Errorf("certificate blob leaked:\n%s", got)
	}
	if !strings.Contains(got, "com.apple.security.pkcs12") {
		t.Errorf("top-level PayloadContent array was blanked, losing the profile:\n%s", got)
	}
	if !strings.Contains(got, redactedPlaceholder) {
		t.Errorf("expected the blob to be redacted:\n%s", got)
	}
}

func TestRedactPlist_UnparseableFailsClosed(t *testing.T) {
	got := redactPlist([]byte(`<plist version="1.0"><dict><key>oops`))
	if got != redactedPlaceholder {
		t.Errorf("unparseable plist must be withheld wholesale, got: %q", got)
	}
}

// --- fail-closed behaviour -------------------------------------------------

func TestFormatBody_WithholdsOpaqueBody(t *testing.T) {
	got := formatBody([]byte("user=admin&password=hunter2"))
	if strings.Contains(got, "hunter2") {
		t.Errorf("form-encoded secret leaked: %q", got)
	}
	if !strings.Contains(got, "body withheld") {
		t.Errorf("expected withheld notice, got: %q", got)
	}
}

func TestFormatBody_WithholdsInvalidJSON(t *testing.T) {
	got := formatBody([]byte(`{"password":"hunter2",`))
	if strings.Contains(got, "hunter2") {
		t.Errorf("secret in truncated JSON leaked: %q", got)
	}
	if !strings.Contains(got, "body withheld") {
		t.Errorf("expected withheld notice, got: %q", got)
	}
}

func TestFormatBody_WithholdsInvalidXML(t *testing.T) {
	got := formatBody([]byte(`<binding><password>hunter2</password`))
	if strings.Contains(got, "hunter2") {
		t.Errorf("secret in malformed XML leaked: %q", got)
	}
	if !strings.Contains(got, "body withheld") {
		t.Errorf("expected withheld notice, got: %q", got)
	}
}

func TestFormatBody_WithholdsTrailingGarbageAfterJSON(t *testing.T) {
	got := formatBody([]byte(`{"a":1} then some raw text with password=hunter2`))
	if strings.Contains(got, "hunter2") {
		t.Errorf("trailing raw content leaked: %q", got)
	}
}

func TestFormatBody_KeepsMultipartSentinel(t *testing.T) {
	if got := formatBody([]byte(multipartSentinel)); got != multipartSentinel {
		t.Errorf("multipart sentinel should pass through, got %q", got)
	}
}

func TestFormatBody_Empty(t *testing.T) {
	if got := formatBody([]byte("   ")); got != "   " {
		t.Errorf("whitespace-only: expected unchanged, got %q", got)
	}
}

// --- headers ---------------------------------------------------------------

func TestFormatHeaders_SortedAndJoined(t *testing.T) {
	h := http.Header{
		"X-Custom":      []string{"v1"},
		"Content-Type":  []string{"application/xml"},
		"Cache-Control": []string{"no-cache", "no-store"},
	}
	got := formatHeaders(h)
	want := "Cache-Control: no-cache, no-store\nContent-Type: application/xml\nX-Custom: v1"
	if got != want {
		t.Errorf("unexpected output:\nwant: %q\ngot:  %q", want, got)
	}
}

func TestFormatHeaders_RedactsSensitive(t *testing.T) {
	h := http.Header{
		"Authorization": []string{"Bearer secret-token-do-not-leak"},
		"Set-Cookie":    []string{"session=abc123; HttpOnly"},
		"Cookie":        []string{"session=abc123"},
		"X-Api-Key":     []string{"k-12345"},
		"Content-Type":  []string{"application/json"},
	}
	got := formatHeaders(h)
	for _, leak := range []string{"secret-token-do-not-leak", "abc123", "k-12345"} {
		if strings.Contains(got, leak) {
			t.Errorf("sensitive value leaked in output: %q\nfull:\n%s", leak, got)
		}
	}
	for _, want := range []string{"Authorization: REDACTED", "Set-Cookie: REDACTED", "Cookie: REDACTED", "X-Api-Key: REDACTED", "Content-Type: application/json"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing line %q in:\n%s", want, got)
		}
	}
}

func TestFormatHeaders_Empty(t *testing.T) {
	if got := formatHeaders(nil); got != "" {
		t.Errorf("nil headers: expected empty, got %q", got)
	}
	if got := formatHeaders(http.Header{}); got != "" {
		t.Errorf("empty headers: expected empty, got %q", got)
	}
}

// --- truncation ------------------------------------------------------------

// TestFormatBody_TruncatesAfterRedaction builds a JSON object large enough to
// exceed maxLoggedBodyBytes once indented, and asserts the truncation marker.
func TestFormatBody_TruncatesAfterRedaction(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`{"items":[`)
	for i := range 200 {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(`{"name":"item-with-a-fairly-long-name-to-pad-bytes","id":`)
		sb.WriteString("12345")
		sb.WriteString("}")
	}
	sb.WriteString("]}")

	got := formatBody([]byte(sb.String()))
	if !strings.HasSuffix(got, "... (truncated)") {
		t.Errorf("expected truncation marker, got tail: %q", got[len(got)-50:])
	}
	if len(got) > maxLoggedBodyBytes+len("... (truncated)") {
		t.Errorf("truncated string too long: %d", len(got))
	}
}

// TestFormatBody_RedactsBeforeTruncating proves ordering: a secret early in a
// long body must be gone even though truncation only removes the tail.
func TestFormatBody_RedactsBeforeTruncating(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`{"password":"hunter2","items":[`)
	for i := range 300 {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(`"padding-value-to-push-past-the-truncation-limit"`)
	}
	sb.WriteString("]}")

	got := formatBody([]byte(sb.String()))
	if strings.Contains(got, "hunter2") {
		t.Errorf("secret survived in a truncated body:\n%s", got[:200])
	}
	if !strings.HasSuffix(got, "... (truncated)") {
		t.Errorf("expected truncation to still apply")
	}
}
