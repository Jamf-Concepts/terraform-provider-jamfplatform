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

func TestCompactStructuralWhitespace_PrettyPrintedArraysCollapsed(t *testing.T) {
	// com.apple.homescreenlayout shape: Pages is an array of arrays, plus a
	// Folder holding its own Pages array-of-arrays — nested arrays at two
	// depths, all pretty-printed. Every structural gap must vanish; the
	// double space inside the <string> value must survive.
	in := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Pages</key>
	<array>
		<array>
			<dict>
				<key>displayName</key>
				<string>App  Name</string>
				<key>Pages</key>
				<array>
					<array/>
					<array></array>
				</array>
			</dict>
		</array>
		<array/>
	</array>
</dict>
</plist>
`
	want := `<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><dict><key>Pages</key><array><array><dict><key>displayName</key><string>App  Name</string><key>Pages</key><array><array/><array></array></array></dict></array><array/></array></dict></plist>`
	got, err := CompactStructuralWhitespace([]byte(in))
	if err != nil {
		t.Fatalf("compact: %v", err)
	}
	if string(got) != want {
		t.Errorf("compact mismatch:\ngot:  %s\nwant: %s", got, want)
	}
	// Compaction must be semantics-preserving.
	if eq, ok := SemanticallyEqual([]byte(in), got); !ok || !eq {
		t.Errorf("compacted output not semantically equal to input (eq=%v ok=%v)", eq, ok)
	}
}

func TestCompactStructuralWhitespace_AlreadyCompactNoOp(t *testing.T) {
	in := `<plist version="1.0"><dict><key>K</key><array><array/></array></dict></plist>`
	got, err := CompactStructuralWhitespace([]byte(in))
	if err != nil {
		t.Fatalf("compact: %v", err)
	}
	if string(got) != in {
		t.Errorf("already-compact input changed:\ngot:  %s\nwant: %s", got, in)
	}
}

func TestCompactStructuralWhitespace_EmptyInputNoOp(t *testing.T) {
	got, err := CompactStructuralWhitespace(nil)
	if err != nil {
		t.Fatalf("compact: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("empty input: got %q", got)
	}
}

func TestCompactStructuralWhitespace_LeafContentUntouched(t *testing.T) {
	// Whitespace, newlines, base64 line-wraps, and unicode inside leaf value
	// elements are content, not formatting — byte-for-byte preservation.
	leaves := []string{
		"<string>   </string>",                                  // whitespace-only string value
		"<string>line one\n\tline two</string>",                 // embedded newline + tab
		"<key>a b</key>",                                        // key with a space
		"<string>TUlJQ2x6Q0NBWA==\nTUlJQ2x6Q0NBWA==\n</string>", // line-wrapped base64
		"<data>\n\tTUlJQ2x6Q0NBWA==\n</data>",                   // wrapped <data> blob
		"<string>émoji 🎉 — ünïcode</string>",                    // unicode
	}
	for _, leaf := range leaves {
		in := "<plist version=\"1.0\">\n<dict>\n\t<key>K</key>\n\t" + leaf + "\n</dict>\n</plist>"
		want := `<plist version="1.0"><dict><key>K</key>` + leaf + `</dict></plist>`
		// <key>a b</key> is the dict key itself, not a value — adjust shape.
		if leaf == "<key>a b</key>" {
			in = "<plist version=\"1.0\">\n<dict>\n\t" + leaf + "\n\t<string>v</string>\n</dict>\n</plist>"
			want = `<plist version="1.0"><dict>` + leaf + `<string>v</string></dict></plist>`
		}
		got, err := CompactStructuralWhitespace([]byte(in))
		if err != nil {
			t.Fatalf("%s: compact: %v", leaf, err)
		}
		if string(got) != want {
			t.Errorf("%s:\ngot:  %s\nwant: %s", leaf, got, want)
		}
	}
}

func TestCompactStructuralWhitespace_CommentsAndCDATAPreserved(t *testing.T) {
	in := "<plist version=\"1.0\">\n<dict>\n\t<!-- keep </array> me -->\n\t<key>K</key>\n\t<string><![CDATA[  raw <array> bytes  ]]></string>\n\t<key>W</key>\n\t<array><![CDATA[  ]]></array>\n</dict>\n</plist>"
	want := `<plist version="1.0"><dict><!-- keep </array> me --><key>K</key><string><![CDATA[  raw <array> bytes  ]]></string><key>W</key><array><![CDATA[  ]]></array></dict></plist>`
	got, err := CompactStructuralWhitespace([]byte(in))
	if err != nil {
		t.Fatalf("compact: %v", err)
	}
	if string(got) != want {
		t.Errorf("comment/CDATA handling:\ngot:  %s\nwant: %s", got, want)
	}
}

func TestCompactStructuralWhitespace_CharacterReferenceWhitespacePreserved(t *testing.T) {
	// &#x20; decodes to a space (whitespace CharData token) but the raw bytes
	// are not whitespace — must survive verbatim.
	in := `<plist version="1.0"><dict><key>K</key><array>&#x20;</array></dict></plist>`
	got, err := CompactStructuralWhitespace([]byte(in))
	if err != nil {
		t.Fatalf("compact: %v", err)
	}
	if string(got) != in {
		t.Errorf("character reference stripped:\ngot:  %s\nwant: %s", got, in)
	}
}

func TestCompactStructuralWhitespace_MalformedReturnsInputAndError(t *testing.T) {
	in := `<plist><dict><key>K</key></dict>` // unclosed <plist>
	got, err := CompactStructuralWhitespace([]byte(in))
	if err == nil {
		t.Fatal("expected error on malformed XML")
	}
	if string(got) != in {
		t.Errorf("malformed input must be returned unchanged, got %q", got)
	}
}
