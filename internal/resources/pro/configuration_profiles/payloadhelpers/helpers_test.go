// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package payloadhelpers

import (
	"strings"
	"testing"
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
	_, _, err := ParsePlist([]byte("not a plist"))
	if err == nil {
		t.Fatal("expected error on invalid input")
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
	check, _, err := ParsePlist(out)
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
	if _, _, err := ParsePlist(payload); err != nil {
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
