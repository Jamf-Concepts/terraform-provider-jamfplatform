// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package plisthelpers

import "testing"

const minimalPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadContent</key><array/>
</dict></plist>`

func TestParsePlist_BasicRoundTrip(t *testing.T) {
	m, _, err := ParsePlist([]byte(minimalPlist))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if m["PayloadType"] != "Configuration" {
		t.Fatalf("PayloadType: got %v", m["PayloadType"])
	}
}

func TestParsePlist_InvalidInput(t *testing.T) {
	if _, _, err := ParsePlist([]byte("not a plist")); err == nil {
		t.Fatal("expected error on invalid input")
	}
}

func TestParsePlist_BareDictFragment(t *testing.T) {
	// Managed-app-config preferences arrive as a bare <dict> with no <plist>
	// wrapper — must parse.
	m, _, err := ParsePlist([]byte("<dict>\n  <key>ServerURL</key>\n  <string>https://example.com</string>\n</dict>\n"))
	if err != nil {
		t.Fatalf("parse bare dict: %v", err)
	}
	if m["ServerURL"] != "https://example.com" {
		t.Fatalf("ServerURL: got %v", m["ServerURL"])
	}
}

func TestMarshalPlist_RoundTrip(t *testing.T) {
	parsed, _, err := ParsePlist([]byte(minimalPlist))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, err := MarshalPlist(parsed)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	reparsed, _, err := ParsePlist(out)
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	if reparsed["PayloadType"] != parsed["PayloadType"] {
		t.Errorf("PayloadType: got=%v want=%v", reparsed["PayloadType"], parsed["PayloadType"])
	}
}

func TestCanonicalisePlistXML_UnparseableReturnedUnchanged(t *testing.T) {
	in := []byte("not a plist")
	if got := CanonicalisePlistXML(in); string(got) != string(in) {
		t.Fatalf("expected unparseable input returned unchanged, got %q", got)
	}
}

func TestEqual(t *testing.T) {
	cases := []struct {
		name string
		a, b any
		want bool
	}{
		{"equal maps", map[string]any{"k": "v"}, map[string]any{"k": "v"}, true},
		{"asymmetric keyset", map[string]any{"k": "v", "x": 1}, map[string]any{"k": "v"}, false},
		{"value mismatch", map[string]any{"k": "v"}, map[string]any{"k": "w"}, false},
		{"arrays positional", []any{"a", "b"}, []any{"a", "b"}, true},
		{"array length mismatch", []any{"a"}, []any{"a", "b"}, false},
		{"int trio uint64/int64", uint64(5), int64(5), true},
		{"nil both", nil, nil, true},
		{"nil one side", nil, "x", false},
	}
	for _, tc := range cases {
		if got := Equal(tc.a, tc.b); got != tc.want {
			t.Errorf("%s: Equal=%v want %v", tc.name, got, tc.want)
		}
	}
}

func TestNumericEqual(t *testing.T) {
	if !NumericEqual(5, uint64(5)) || !NumericEqual(5, int64(5)) || !NumericEqual(5, 5) {
		t.Error("5 should equal its uint64/int64/int forms")
	}
	if NumericEqual(-1, uint64(1)) {
		t.Error("negative int64 must not equal a uint64")
	}
	if NumericEqual(5, "5") {
		t.Error("numeric must not equal a string")
	}
}

func TestSemanticallyEqual(t *testing.T) {
	// Same dict, different formatting (indent, key order, trailing newline).
	a := "<dict>\n\t<key>A</key><string>1</string>\n\t<key>B</key><integer>2</integer>\n</dict>\n"
	b := "<dict><key>B</key><integer>2</integer><key>A</key><string>1</string></dict>"
	eq, ok := SemanticallyEqual([]byte(a), []byte(b))
	if !ok {
		t.Fatal("both should parse as plist")
	}
	if !eq {
		t.Error("same dict modulo formatting/key-order should be equal")
	}

	// Real value change.
	c := "<dict><key>A</key><string>2</string></dict>"
	d := "<dict><key>A</key><string>3</string></dict>"
	if eq, _ := SemanticallyEqual([]byte(c), []byte(d)); eq {
		t.Error("different values must not be equal")
	}

	// Non-plist → ok=false (caller falls back).
	if _, ok := SemanticallyEqual([]byte("plain text"), []byte("plain text")); ok {
		t.Error("non-plist input should report ok=false")
	}
}
