// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package payloadhelpers

import (
	"strings"
	"testing"
)

// profileWith wraps a single PayloadContent entry of the given type around the
// supplied key/value plist source.
func profileWith(ptype, keysAndValues string) []byte {
	return []byte(`<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" ` +
		`"http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><dict>` +
		`<key>PayloadContent</key><array><dict>` +
		`<key>PayloadType</key><string>` + ptype + `</string>` +
		`<key>PayloadVersion</key><integer>1</integer>` +
		keysAndValues +
		`</dict></array>` +
		`<key>PayloadType</key><string>Configuration</string>` +
		`<key>PayloadVersion</key><integer>1</integer>` +
		`</dict></plist>`)
}

// mcxProfile wraps vendor preference source in the macOS "Application & Custom
// Settings" shape, whose inner subtree Jamf Pro stores faithfully.
func mcxProfile(keysAndValues string) []byte {
	return profileWith("com.apple.ManagedClient.preferences",
		`<key>PayloadContent</key><dict><key>com.zz.probe</key><dict><key>Forced</key><array>`+
			`<dict><key>mcx_preference_settings</key><dict>`+keysAndValues+`</dict></dict>`+
			`</array></dict></dict>`)
}

func TestPayloadImportRisk_FlagsVerbatimHazards(t *testing.T) {
	tests := []struct {
		name     string
		payload  []byte
		platform ProfilePlatform
		wantPath string
	}{
		{
			name:     "ampersand in a verbatim macOS payload",
			payload:  profileWith("com.apple.loginwindow", `<key>LoginwindowText</key><string>Acme &amp; Co</string>`),
			platform: PlatformMacOS,
			wantPath: "PayloadContent[0].LoginwindowText",
		},
		{
			name:     "less-than in a verbatim macOS payload",
			payload:  profileWith("com.apple.loginwindow", `<key>LoginwindowText</key><string>a &lt; b</string>`),
			platform: PlatformMacOS,
			wantPath: "PayloadContent[0].LoginwindowText",
		},
		{
			name:     "internal line feed in a verbatim macOS payload",
			payload:  profileWith("com.apple.loginwindow", "<key>LoginwindowText</key><string>one&#10;two</string>"),
			platform: PlatformMacOS,
			wantPath: "PayloadContent[0].LoginwindowText",
		},
		{
			name:     "internal tab in a verbatim macOS payload",
			payload:  profileWith("com.apple.loginwindow", "<key>LoginwindowText</key><string>one&#9;two</string>"),
			platform: PlatformMacOS,
			wantPath: "PayloadContent[0].LoginwindowText",
		},
		{
			name:     "already-corrupted value escalates further",
			payload:  profileWith("com.apple.loginwindow", `<key>LoginwindowText</key><string>Acme &amp;amp; Co</string>`),
			platform: PlatformMacOS,
			wantPath: "PayloadContent[0].LoginwindowText",
		},
		{
			name:     "URL query string in a mobile web clip",
			payload:  profileWith("com.apple.webClip.managed", `<key>URL</key><string>https://x.test/?a=1&amp;b=2</string>`),
			platform: PlatformMobileDevice,
			wantPath: "PayloadContent[0].URL",
		},
		{
			name:     "MCX is verbatim on mobile even though it is faithful on macOS",
			payload:  mcxProfile(`<key>note</key><string>Acme &amp; Co</string>`),
			platform: PlatformMobileDevice,
			wantPath: "note",
		},
		{
			name:     "applicationaccess is verbatim on macOS even though it is faithful on mobile",
			payload:  profileWith("com.apple.applicationaccess", `<key>note</key><string>Acme &amp; Co</string>`),
			platform: PlatformMacOS,
			wantPath: "PayloadContent[0].note",
		},
		{
			name: "top-level slot outside PayloadContent is verbatim",
			payload: []byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict>` +
				`<key>ConsentText</key><dict><key>default</key><string>Acme &amp; Co</string></dict>` +
				`<key>PayloadType</key><string>Configuration</string></dict></plist>`),
			platform: PlatformMacOS,
			wantPath: "ConsentText.default",
		},
		{
			name:     "unprobed payload type is treated as verbatim",
			payload:  profileWith("com.zz.never.probed", `<key>note</key><string>Acme &amp; Co</string>`),
			platform: PlatformMacOS,
			wantPath: "PayloadContent[0].note",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			detail, lossy, ok := PayloadImportRisk(tc.payload, tc.platform)
			if !ok {
				t.Fatalf("payload did not parse")
			}
			if !lossy {
				t.Fatalf("expected the gate to flag this payload, got clean")
			}
			if !strings.Contains(detail, tc.wantPath) {
				t.Errorf("detail did not name %q:\n%s", tc.wantPath, detail)
			}
		})
	}
}

// TestPayloadImportRisk_NoFalsePositives is the important half: every case here
// round-trips through Jamf Pro unchanged, so flagging any of them would block an
// import that would have worked.
func TestPayloadImportRisk_NoFalsePositives(t *testing.T) {
	tests := []struct {
		name     string
		payload  []byte
		platform ProfilePlatform
	}{
		{
			name:     "greater-than survives verbatim storage",
			payload:  profileWith("com.apple.loginwindow", `<key>LoginwindowText</key><string>a &gt; b</string>`),
			platform: PlatformMacOS,
		},
		{
			name:     "double and single quotes survive",
			payload:  profileWith("com.apple.loginwindow", `<key>LoginwindowText</key><string>say &quot;hi&quot; and &apos;bye&apos;</string>`),
			platform: PlatformMacOS,
		},
		{
			name:     "carriage return survives and is the documented line-break form",
			payload:  profileWith("com.apple.loginwindow", "<key>LoginwindowText</key><string>one&#13;two</string>"),
			platform: PlatformMacOS,
		},
		{
			name:     "unicode line and paragraph separators survive",
			payload:  profileWith("com.apple.loginwindow", "<key>LoginwindowText</key><string>a&#8232;b&#8233;c&#133;d</string>"),
			platform: PlatformMacOS,
		},
		{
			name:     "leading and trailing whitespace is trimmed on both sides",
			payload:  profileWith("com.apple.loginwindow", "<key>LoginwindowText</key><string>&#10;&#9;padded&#9;&#10;</string>"),
			platform: PlatformMacOS,
		},
		{
			name:     "ampersand inside macOS Application and Custom Settings",
			payload:  mcxProfile(`<key>note</key><string>Acme &amp; Co</string>`),
			platform: PlatformMacOS,
		},
		{
			name:     "less-than inside macOS Application and Custom Settings",
			payload:  mcxProfile(`<key>expr</key><string>a &lt; b</string>`),
			platform: PlatformMacOS,
		},
		{
			name:     "line feed inside macOS Application and Custom Settings",
			payload:  mcxProfile("<key>note</key><string>one&#10;two</string>"),
			platform: PlatformMacOS,
		},
		{
			name:     "ampersand in a macOS re-render type",
			payload:  profileWith("com.apple.webcontent-filter", `<key>UserDefinedName</key><string>Acme &amp; Co</string>`),
			platform: PlatformMacOS,
		},
		{
			name:     "ampersand in a mobile re-render type",
			payload:  profileWith("com.apple.applicationaccess", `<key>note</key><string>Acme &amp; Co</string>`),
			platform: PlatformMobileDevice,
		},
		{
			name:     "masked key holding an ampersand is never the reason to refuse",
			payload:  profileWith("com.apple.loginwindow", `<key>PayloadDisplayName</key><string>Acme &amp; Co</string>`),
			platform: PlatformMacOS,
		},
		{
			name: "masked top-level organisation holding an ampersand",
			payload: []byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict>` +
				`<key>PayloadOrganization</key><string>Acme &amp; Co</string>` +
				`<key>PayloadType</key><string>Configuration</string></dict></plist>`),
			platform: PlatformMacOS,
		},
		{
			name:     "base64 data blob is not a string leaf",
			payload:  profileWith("com.apple.security.pkcs1", `<key>PayloadContent</key><data>YSA8IGImIGM=</data>`),
			platform: PlatformMacOS,
		},
		{
			name:     "plain text in a verbatim type",
			payload:  profileWith("com.apple.loginwindow", `<key>LoginwindowText</key><string>Welcome to Acme</string>`),
			platform: PlatformMacOS,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			detail, lossy, ok := PayloadImportRisk(tc.payload, tc.platform)
			if !ok {
				t.Fatalf("payload did not parse")
			}
			if lossy {
				t.Errorf("false positive — gate refused a payload that round-trips:\n%s", detail)
			}
		})
	}
}

// TestPayloadImportRisk_UnparseablePayloadIsNotAGate keeps an unreadable payload
// out of the gate's remit: ok=false tells the caller to let the import proceed.
func TestPayloadImportRisk_UnparseablePayloadIsNotAGate(t *testing.T) {
	if _, _, ok := PayloadImportRisk([]byte("not a plist at all"), PlatformMacOS); ok {
		t.Fatal("expected ok=false for an unparseable payload")
	}
	if diags := ImportGateDiagnostics([]byte("not a plist at all"), PlatformMacOS, "n", "1"); diags.HasError() {
		t.Fatalf("unparseable payload must not block import: %v", diags)
	}
	if diags := ImportGateDiagnostics(nil, PlatformMacOS, "n", "1"); diags.HasError() {
		t.Fatalf("empty payload must not block import: %v", diags)
	}
}

// TestImportGateDiagnostics_WordingNamesTheConsequence guards the parts of the
// message an operator acts on: what happened, that nothing changed, and that
// repair is admin-UI-only.
func TestImportGateDiagnostics_WordingNamesTheConsequence(t *testing.T) {
	diags := ImportGateDiagnostics(
		profileWith("com.apple.loginwindow", `<key>LoginwindowText</key><string>Acme &amp; Co</string>`),
		PlatformMacOS, "Login Window", "16",
	)
	if !diags.HasError() {
		t.Fatal("expected an error diagnostic")
	}
	detail := diags.Errors()[0].Detail()
	for _, want := range []string{
		"Login Window",
		"id 16",
		"Nothing has been imported",
		"admin",
		"Application & Custom Settings",
		"no override",
	} {
		if !strings.Contains(detail, want) {
			t.Errorf("detail is missing %q:\n%s", want, detail)
		}
	}
	if summary := diags.Errors()[0].Summary(); summary != ImportGateSummary {
		t.Errorf("summary = %q, want %q", summary, ImportGateSummary)
	}
}

// TestImportGateSkip_OnlyDropsUnmanageableItems checks the list-resource
// predicate, which must drop exactly the profiles the error-path gate refuses.
func TestImportGateSkip_OnlyDropsUnmanageableItems(t *testing.T) {
	if !ImportGateSkip(
		profileWith("com.apple.loginwindow", `<key>LoginwindowText</key><string>Acme &amp; Co</string>`),
		PlatformMacOS) {
		t.Error("expected an unmanageable profile to be skipped")
	}
	if ImportGateSkip(
		profileWith("com.apple.loginwindow", `<key>LoginwindowText</key><string>Welcome</string>`),
		PlatformMacOS) {
		t.Error("a clean profile must not be skipped")
	}
	if ImportGateSkip(nil, PlatformMacOS) {
		t.Error("an empty payload must not be skipped")
	}
	if ImportGateSkip([]byte("not a plist"), PlatformMacOS) {
		t.Error("an unparseable payload must not be skipped")
	}
}

// TestImportGateSkipWarning_ConsolidatesAndBounds checks the list path reports one
// warning naming the dropped profiles, never an error (which would abandon config
// generation for the whole tenant), and caps how many it lists.
func TestImportGateSkipWarning_ConsolidatesAndBounds(t *testing.T) {
	if diags := ImportGateSkipWarning(nil, "jamfplatform_pro_macos_configuration_profile"); len(diags) != 0 {
		t.Fatalf("no skips must produce no diagnostics, got %v", diags)
	}

	diags := ImportGateSkipWarning([]string{"Login Window", "Setup Manager"},
		"jamfplatform_pro_macos_configuration_profile")
	if diags.HasError() {
		t.Fatalf("list path must warn, not error: %v", diags)
	}
	if len(diags.Warnings()) != 1 {
		t.Fatalf("expected one consolidated warning, got %d", len(diags.Warnings()))
	}
	detail := diags.Warnings()[0].Detail()
	for _, want := range []string{"Login Window", "Setup Manager", "terraform import"} {
		if !strings.Contains(detail, want) {
			t.Errorf("detail is missing %q:\n%s", want, detail)
		}
	}

	many := make([]string, maxListedSkips+5)
	for i := range many {
		many[i] = "profile"
	}
	overflow := ImportGateSkipWarning(many, "jamfplatform_pro_macos_configuration_profile")
	if !strings.Contains(overflow.Warnings()[0].Detail(), "and 5 more") {
		t.Errorf("expected the listing to be capped:\n%s", overflow.Warnings()[0].Detail())
	}
}

// TestApplyVerbatimStorage pins the transform itself against the wire law.
func TestApplyVerbatimStorage(t *testing.T) {
	tests := []struct{ in, want string }{
		{"plain", "plain"},
		{"a & b", "a &amp; b"},
		{"a < b", "a &lt; b"},
		{"a > b", "a > b"},
		{"a\nb", "ab"},
		{"a\tb", "ab"},
		{"a\rb", "a\rb"},
		{"&amp;", "&amp;amp;"},
		{"a\u2028b\u2029c\u0085d", "a\u2028b\u2029c\u0085d"},
	}
	for _, tc := range tests {
		if got := applyVerbatimStorage(tc.in); got != tc.want {
			t.Errorf("applyVerbatimStorage(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
