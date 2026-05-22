// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"net/http"
	"strings"
	"testing"
)

func TestPrettyPrint_JSONObject(t *testing.T) {
	in := []byte(`{"name":"foo","id":1,"tags":["a","b"]}`)
	got := prettyPrint(in)
	if !strings.Contains(got, "\n  \"name\": \"foo\"") {
		t.Errorf("JSON not indented:\n%s", got)
	}
	if !strings.Contains(got, "\n  \"tags\": [") {
		t.Errorf("JSON nested array not indented:\n%s", got)
	}
}

func TestPrettyPrint_JSONArray(t *testing.T) {
	in := []byte(`[{"a":1},{"a":2}]`)
	got := prettyPrint(in)
	if !strings.HasPrefix(got, "[\n  {") {
		t.Errorf("JSON array root not indented:\n%s", got)
	}
}

func TestPrettyPrint_XMLBasic(t *testing.T) {
	in := []byte(`<user_group><id>3</id><name>Excluded Users</name><is_smart>false</is_smart></user_group>`)
	got := prettyPrint(in)
	// Each child element must land on its own indented line.
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

func TestPrettyPrint_XMLNested(t *testing.T) {
	in := []byte(`<user_group><criteria><size>1</size><criterion><name>User Group</name><value>x</value></criterion></criteria></user_group>`)
	got := prettyPrint(in)
	// Nested criterion gets two-step indent.
	if !strings.Contains(got, "    <criterion>\n      <name>User Group</name>") {
		t.Errorf("XML nesting not indented to 4 spaces:\n%s", got)
	}
}

func TestPrettyPrint_LeadingWhitespace(t *testing.T) {
	in := []byte("\n\t  <foo><bar/></foo>")
	got := prettyPrint(in)
	if !strings.Contains(got, "<foo>\n  <bar></bar>") {
		t.Errorf("leading whitespace tripped XML sniff:\n%s", got)
	}
}

func TestPrettyPrint_Opaque_PassesThrough(t *testing.T) {
	in := []byte("plain text, not JSON, not XML")
	got := prettyPrint(in)
	if got != string(in) {
		t.Errorf("opaque body mutated: %q", got)
	}
}

func TestPrettyPrint_InvalidJSON_PassesThrough(t *testing.T) {
	in := []byte(`{not valid json`)
	got := prettyPrint(in)
	if got != string(in) {
		t.Errorf("invalid JSON should fall through to raw: %q", got)
	}
}

func TestPrettyPrint_InvalidXML_PassesThrough(t *testing.T) {
	in := []byte(`<unclosed`)
	got := prettyPrint(in)
	if got != string(in) {
		t.Errorf("invalid XML should fall through to raw: %q", got)
	}
}

func TestPrettyPrint_Empty(t *testing.T) {
	if got := prettyPrint(nil); got != "" {
		t.Errorf("nil body: expected empty, got %q", got)
	}
	if got := prettyPrint([]byte("   ")); got != "   " {
		t.Errorf("whitespace-only: expected unchanged, got %q", got)
	}
}

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

func TestFormatBody_TruncatesAfterPretty(t *testing.T) {
	// Build a JSON object large enough to exceed maxLoggedBodyBytes after indent.
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
