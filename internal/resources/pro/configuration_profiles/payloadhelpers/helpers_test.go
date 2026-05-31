// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package payloadhelpers

import (
	"strings"
	"testing"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/plisthelpers"
)

const minimalPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadUUID</key><string>AAAABBBB-1111-2222-3333-444455556666</string>
<key>PayloadIdentifier</key><string>AAAABBBB-1111-2222-3333-444455556666</string>
<key>PayloadDisplayName</key><string>Test Profile</string>
<key>PayloadContent</key><array/>
</dict></plist>`

const minimalWireXML = `<?xml version="1.0" encoding="UTF-8"?>
<configuration_profile><general><id>1</id><payloads>&lt;?xml version=&quot;1.0&quot; encoding=&quot;UTF-8&quot;?&gt;
&lt;!DOCTYPE plist PUBLIC &quot;-//Apple//DTD PLIST 1.0//EN&quot; &quot;http://www.apple.com/DTDs/PropertyList-1.0.dtd&quot;&gt;
&lt;plist version=&quot;1.0&quot;&gt;&lt;dict&gt;
&lt;key&gt;PayloadType&lt;/key&gt;&lt;string&gt;Configuration&lt;/string&gt;
&lt;key&gt;PayloadVersion&lt;/key&gt;&lt;integer&gt;1&lt;/integer&gt;
&lt;key&gt;PayloadUUID&lt;/key&gt;&lt;string&gt;AAAABBBB-1111-2222-3333-444455556666&lt;/string&gt;
&lt;key&gt;PayloadIdentifier&lt;/key&gt;&lt;string&gt;AAAABBBB-1111-2222-3333-444455556666&lt;/string&gt;
&lt;key&gt;PayloadDisplayName&lt;/key&gt;&lt;string&gt;Test Profile&lt;/string&gt;
&lt;key&gt;PayloadContent&lt;/key&gt;&lt;array/&gt;
&lt;/dict&gt;&lt;/plist&gt;</payloads></general></configuration_profile>`

func TestPayloadsSemanticallyEqual_Identical(t *testing.T) {
	eq, err := PayloadsSemanticallyEqual([]byte(minimalPlist), []byte(minimalPlist))
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if !eq {
		t.Fatal("identical payloads must be equal")
	}
}

func TestPayloadsSemanticallyEqual_MaskedKeysOnly_Equal(t *testing.T) {
	const a = `<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict><key>PayloadType</key><string>Configuration</string><key>PayloadVersion</key><integer>1</integer><key>PayloadUUID</key><string>AAAA</string><key>PayloadDisplayName</key><string>A</string><key>PayloadContent</key><array/></dict></plist>`
	const b = `<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict><key>PayloadType</key><string>Configuration</string><key>PayloadVersion</key><integer>1</integer><key>PayloadUUID</key><string>BBBB</string><key>PayloadDisplayName</key><string>B</string><key>PayloadContent</key><array/></dict></plist>`
	eq, err := PayloadsSemanticallyEqual([]byte(a), []byte(b))
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if !eq {
		t.Fatal("diff only in masked keys must be equal")
	}
}

func TestPayloadsSemanticallyEqual_RealChange_NotEqual(t *testing.T) {
	const a = `<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict><key>PayloadType</key><string>Configuration</string><key>PayloadVersion</key><integer>1</integer><key>PayloadContent</key><array/></dict></plist>`
	const b = `<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict><key>PayloadType</key><string>Configuration</string><key>PayloadVersion</key><integer>2</integer><key>PayloadContent</key><array/></dict></plist>`
	eq, err := PayloadsSemanticallyEqual([]byte(a), []byte(b))
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if eq {
		t.Fatal("changed PayloadVersion must not be equal")
	}
}

func TestPayloadsSemanticallyEqual_WhitespaceDiffEqual(t *testing.T) {
	reformatted := strings.ReplaceAll(minimalPlist, "\t", "    ")
	eq, err := PayloadsSemanticallyEqual([]byte(minimalPlist), []byte(reformatted))
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if !eq {
		t.Fatal("whitespace-only diff must be equal")
	}
}

func TestInjectTopLevelIdentifierValues_ReplacesUUIDs(t *testing.T) {
	const existingUUID = "CCCCDDDD-5555-6666-7777-888899990000"
	out, err := InjectTopLevelIdentifierValues([]byte(minimalPlist), existingUUID, existingUUID)
	if err != nil {
		t.Fatalf("inject: %v", err)
	}
	check, _, err := plisthelpers.ParsePlist(out)
	if err != nil {
		t.Fatalf("parse output: %v", err)
	}
	if check["PayloadUUID"] != existingUUID {
		t.Errorf("PayloadUUID: got=%v want=%v", check["PayloadUUID"], existingUUID)
	}
	if check["PayloadIdentifier"] != existingUUID {
		t.Errorf("PayloadIdentifier: got=%v want=%v", check["PayloadIdentifier"], existingUUID)
	}
}

func TestInjectTopLevelIdentifierValues_EmptyUUID_NoOp(t *testing.T) {
	out, err := InjectTopLevelIdentifierValues([]byte(minimalPlist), "", "")
	if err != nil {
		t.Fatalf("inject: %v", err)
	}
	if string(out) != minimalPlist {
		t.Fatal("expected no-op when both uuid and identifier are empty")
	}
}

func TestInjectTopLevelIdentifiers_NilExisting_NoOp(t *testing.T) {
	in := []byte(minimalPlist)
	out, err := InjectTopLevelIdentifiers(in, nil)
	if err != nil {
		t.Fatalf("inject: %v", err)
	}
	if string(out) != string(in) {
		t.Fatal("expected no-op when existing is nil")
	}
}

func TestInjectTopLevelIdentifiers_UnparseableExisting_NoOp(t *testing.T) {
	in := []byte(minimalPlist)
	out, err := InjectTopLevelIdentifiers(in, []byte("not a plist"))
	if err != nil {
		t.Fatalf("inject: %v", err)
	}
	if string(out) != string(in) {
		t.Fatal("expected no-op when existing is unparseable")
	}
}

func TestExtractServerPayloadFromGeneral_DecodesEntities(t *testing.T) {
	payload, err := ExtractServerPayloadFromGeneral([]byte(minimalWireXML))
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if _, _, err := plisthelpers.ParsePlist(payload); err != nil {
		t.Fatalf("extracted bytes not valid plist: %v", err)
	}
}

func TestExtractServerPayloadFromGeneral_NoPayloadsBlock(t *testing.T) {
	_, err := ExtractServerPayloadFromGeneral([]byte(`<configuration_profile><general><id>1</id></general></configuration_profile>`))
	if err == nil {
		t.Fatal("expected error when no <payloads> block")
	}
}

func TestMaskPayload_DropsTopLevelMaskedKeys(t *testing.T) {
	m, err := MaskPayload([]byte(minimalPlist))
	if err != nil {
		t.Fatalf("mask: %v", err)
	}
	for _, k := range []string{"PayloadUUID", "PayloadIdentifier", "PayloadDisplayName"} {
		if _, ok := m[k]; ok {
			t.Errorf("masked key %q must be dropped", k)
		}
	}
}

func TestMaskPayload_DropsServerInjectedPayloadTypes(t *testing.T) {
	// Plan side has one PayloadContent entry (notifications). Server side
	// echoes back the same entry plus a server-injected
	// com.apple.profileRemovalPassword entry that materialised because the
	// user set the classic-API authorization_password field. After masking,
	// both sides must have the same single-entry PayloadContent array so
	// PayloadsSemanticallyEqual returns true.
	plan := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadType</key><string>Configuration</string>
  <key>PayloadIdentifier</key><string>plan-id</string>
  <key>PayloadUUID</key><string>plan-uuid</string>
  <key>PayloadVersion</key><integer>1</integer>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>PayloadType</key><string>com.apple.notificationsettings</string>
      <key>PayloadUUID</key><string>nf-uuid</string>
      <key>PayloadIdentifier</key><string>nf-id</string>
      <key>PayloadVersion</key><integer>1</integer>
    </dict>
  </array>
</dict>
</plist>`
	server := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadType</key><string>Configuration</string>
  <key>PayloadIdentifier</key><string>server-id</string>
  <key>PayloadUUID</key><string>server-uuid</string>
  <key>PayloadVersion</key><integer>1</integer>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>PayloadType</key><string>com.apple.profileRemovalPassword</string>
      <key>PayloadUUID</key><string>rp-uuid</string>
      <key>PayloadIdentifier</key><string>rp-id</string>
      <key>PayloadVersion</key><integer>1</integer>
      <key>RemovalPassword</key><string>secret</string>
    </dict>
    <dict>
      <key>PayloadType</key><string>com.apple.notificationsettings</string>
      <key>PayloadUUID</key><string>nf-uuid</string>
      <key>PayloadIdentifier</key><string>nf-id</string>
      <key>PayloadVersion</key><integer>1</integer>
    </dict>
  </array>
</dict>
</plist>`
	eq, err := PayloadsSemanticallyEqual([]byte(plan), []byte(server))
	if err != nil {
		t.Fatalf("PayloadsSemanticallyEqual: %v", err)
	}
	if !eq {
		t.Fatal("expected plan and server payloads to be semantically equal after server-injected RemovalPassword filtered out")
	}
}

func TestLenientEqualPlist_AsymmetricKey_Ignored(t *testing.T) {
	a := map[string]any{"shared": "value", "onlyInA": "x"}
	b := map[string]any{"shared": "value"}
	if !LenientEqualPlist(a, b) {
		t.Fatal("asymmetric key must be ignored (intersection semantics)")
	}
}

func TestLenientEqualPlist_SharedKeyDiffers_NotEqual(t *testing.T) {
	a := map[string]any{"k": "v1"}
	b := map[string]any{"k": "v2"}
	if LenientEqualPlist(a, b) {
		t.Fatal("differing shared key must not be equal")
	}
}

// MCX drift-detection fixtures: a minimal "Application & Custom Settings"
// payload shaped like local-testing/pro/support_files/minimal_notifications.mobileconfig.
// The inner mcx_preference_settings dict is opaque user-content; key add/remove
// inside it must surface as drift.
const mcxBaseline = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadIdentifier</key><string>top-id</string>
<key>PayloadUUID</key><string>top-uuid</string>
<key>PayloadContent</key><array><dict>
<key>PayloadType</key><string>com.apple.ManagedClient.preferences</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadIdentifier</key><string>mcx-id</string>
<key>PayloadUUID</key><string>mcx-uuid</string>
<key>PayloadContent</key><dict>
<key>com.example.app</key><dict>
<key>Forced</key><array><dict>
<key>mcx_preference_settings</key><dict>
<key>FeatureEnabled</key><true/>
<key>BannerMessage</key><string>hello</string>
</dict>
</dict></array>
</dict>
</dict>
</dict></array>
</dict></plist>`

const mcxAddedInnerKey = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadIdentifier</key><string>top-id</string>
<key>PayloadUUID</key><string>top-uuid</string>
<key>PayloadContent</key><array><dict>
<key>PayloadType</key><string>com.apple.ManagedClient.preferences</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadIdentifier</key><string>mcx-id</string>
<key>PayloadUUID</key><string>mcx-uuid</string>
<key>PayloadContent</key><dict>
<key>com.example.app</key><dict>
<key>Forced</key><array><dict>
<key>mcx_preference_settings</key><dict>
<key>FeatureEnabled</key><true/>
<key>BannerMessage</key><string>hello</string>
<key>EnablePageZeroProtection2</key><false/>
</dict>
</dict></array>
</dict>
</dict>
</dict></array>
</dict></plist>`

const mcxRemovedInnerKey = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadIdentifier</key><string>top-id</string>
<key>PayloadUUID</key><string>top-uuid</string>
<key>PayloadContent</key><array><dict>
<key>PayloadType</key><string>com.apple.ManagedClient.preferences</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadIdentifier</key><string>mcx-id</string>
<key>PayloadUUID</key><string>mcx-uuid</string>
<key>PayloadContent</key><dict>
<key>com.example.app</key><dict>
<key>Forced</key><array><dict>
<key>mcx_preference_settings</key><dict>
<key>FeatureEnabled</key><true/>
</dict>
</dict></array>
</dict>
</dict>
</dict></array>
</dict></plist>`

const mcxChangedInnerValue = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadIdentifier</key><string>top-id</string>
<key>PayloadUUID</key><string>top-uuid</string>
<key>PayloadContent</key><array><dict>
<key>PayloadType</key><string>com.apple.ManagedClient.preferences</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadIdentifier</key><string>mcx-id</string>
<key>PayloadUUID</key><string>mcx-uuid</string>
<key>PayloadContent</key><dict>
<key>com.example.app</key><dict>
<key>Forced</key><array><dict>
<key>mcx_preference_settings</key><dict>
<key>FeatureEnabled</key><false/>
<key>BannerMessage</key><string>hello</string>
</dict>
</dict></array>
</dict>
</dict>
</dict></array>
</dict></plist>`

func TestPayloadsSemanticallyEqual_MCXIdentical_Equal(t *testing.T) {
	eq, err := PayloadsSemanticallyEqual([]byte(mcxBaseline), []byte(mcxBaseline))
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if !eq {
		t.Fatal("identical MCX payloads must compare equal")
	}
}

func TestPayloadsSemanticallyEqual_MCXAddedInnerKey_NotEqual(t *testing.T) {
	eq, err := PayloadsSemanticallyEqual([]byte(mcxBaseline), []byte(mcxAddedInnerKey))
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if eq {
		t.Fatal("admin-injected key inside mcx_preference_settings must produce drift")
	}
}

func TestPayloadsSemanticallyEqual_MCXRemovedInnerKey_NotEqual(t *testing.T) {
	eq, err := PayloadsSemanticallyEqual([]byte(mcxBaseline), []byte(mcxRemovedInnerKey))
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if eq {
		t.Fatal("admin-removed key inside mcx_preference_settings must produce drift")
	}
}

func TestPayloadsSemanticallyEqual_MCXChangedInnerValue_NotEqual(t *testing.T) {
	eq, err := PayloadsSemanticallyEqual([]byte(mcxBaseline), []byte(mcxChangedInnerValue))
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if eq {
		t.Fatal("admin-changed value inside mcx_preference_settings must produce drift")
	}
}

// Base64-in-string normalization: when a vendor payload embeds a base64 blob
// inside a <string> value and Jamf Pro line-wraps it on read, the whitespace
// difference must not produce a spurious diff.
const base64Compact = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>EmbeddedCert</key><string>TUlJRENDQWZpZ0F3SUJBZ0lVTUJ4UkxEdGRCZW9PNkZ3MGZQMzZ5cFppL0VBd0RRWUpLb1pJaHZjTkFRRUw=</string>
<key>PayloadContent</key><array/>
</dict></plist>`

const base64Wrapped = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>EmbeddedCert</key><string>TUlJRENDQWZpZ0F3SUJBZ0lVTUJ4
UkxEdGRCZW9PNkZ3MGZQMzZ5cFpp
L0VBd0RRWUpLb1pJaHZjTkFRRUw=</string>
<key>PayloadContent</key><array/>
</dict></plist>`

func TestPayloadsSemanticallyEqual_Base64InString_WrappedAndCompactEqual(t *testing.T) {
	eq, err := PayloadsSemanticallyEqual([]byte(base64Compact), []byte(base64Wrapped))
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if !eq {
		t.Fatal("base64-in-string differing only by line-wrap whitespace must compare equal")
	}
}

func TestNormalizeBase64InString_NonBase64Unchanged(t *testing.T) {
	in := "Allow 1Password Launch Item"
	if got := normalizeBase64InString(in); got != in {
		t.Errorf("non-base64 text mutated: got %q want %q", got, in)
	}
}

func TestNormalizeBase64InString_NoWhitespaceUnchanged(t *testing.T) {
	in := "TUlJRENDQWZpZw=="
	if got := normalizeBase64InString(in); got != in {
		t.Errorf("base64 without whitespace mutated: got %q want %q", got, in)
	}
}

func TestNormalizeBase64InString_StripsInternalWhitespace(t *testing.T) {
	in := "TUlJRENDQWZpZ0F3SUJBZ0lVTUJ4\nUkxEdGRCZW9PNkZ3MGZQMzZ5cFppL0VBd0RRWUpLb1pJaHZjTkFRRUw="
	want := "TUlJRENDQWZpZ0F3SUJBZ0lVTUJ4UkxEdGRCZW9PNkZ3MGZQMzZ5cFppL0VBd0RRWUpLb1pJaHZjTkFRRUw="
	if got := normalizeBase64InString(in); got != want {
		t.Errorf("base64 with whitespace not collapsed: got %q want %q", got, want)
	}
}

func TestNormalizeBase64InString_EmptyAfterStripUnchanged(t *testing.T) {
	in := "   \n\t  "
	if got := normalizeBase64InString(in); got != in {
		t.Errorf("whitespace-only string mutated: got %q want %q", got, in)
	}
}

func TestNormalizeBase64InString_NaturalTextWithSpacesUnchanged(t *testing.T) {
	// Natural-language text containing only base64-alphabet characters and
	// spaces — guards against the heuristic over-firing on values like
	// "Allow 1Password Launch Item" whose whitespace-stripped form would
	// coincidentally decode as base64.
	in := "Allow 1Password Launch Item"
	if got := normalizeBase64InString(in); got != in {
		t.Errorf("natural text with spaces mutated: got %q want %q", got, in)
	}
}

func TestNormalizeBase64InString_NewlineButTooShortUnchanged(t *testing.T) {
	// Newline present but stripped form is below the 32-char floor — must
	// not be collapsed even if it happens to decode as base64.
	in := "TUlJ\nRENDQQ=="
	if got := normalizeBase64InString(in); got != in {
		t.Errorf("short newline-bearing string mutated: got %q want %q", got, in)
	}
}

func TestNormalizeBase64InString_NewlineNotBase64Unchanged(t *testing.T) {
	// Multi-line natural-language string that does not decode as base64 —
	// must pass through untouched even though it has newlines.
	in := "This is a multi-line description\nthat continues onto a second line\nand even a third for good measure."
	if got := normalizeBase64InString(in); got != in {
		t.Errorf("multi-line non-base64 text mutated: got %q want %q", got, in)
	}
}
