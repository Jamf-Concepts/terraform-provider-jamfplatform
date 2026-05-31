// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_configuration_profile

import (
	"testing"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/plisthelpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/resources/pro/configuration_profiles/payloadhelpers"
)

const minimalMobileconfig = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadUUID</key><string>AAAABBBB-1111-2222-3333-444455556666</string>
<key>PayloadIdentifier</key><string>AAAABBBB-1111-2222-3333-444455556666</string>
<key>PayloadDisplayName</key><string>Test Profile</string>
<key>PayloadContent</key><array/>
</dict></plist>`

// minimalWireXML is a representative classic-API GET response that wraps an
// entity-encoded mobileconfig inside <general><payloads>.
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

func TestPayloadsSemanticallyEqual_SamePayload(t *testing.T) {
	eq, err := payloadhelpers.PayloadsSemanticallyEqual([]byte(minimalMobileconfig), []byte(minimalMobileconfig))
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if !eq {
		t.Fatal("identical payloads should be equal")
	}
}

func TestPayloadsSemanticallyEqual_MaskedKeysDiffer_StillEqual(t *testing.T) {
	const a = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadUUID</key><string>AAAA-AAAA</string>
<key>PayloadIdentifier</key><string>AAAA-AAAA</string>
<key>PayloadDisplayName</key><string>Name A</string>
<key>PayloadContent</key><array/>
</dict></plist>`
	const b = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadUUID</key><string>BBBB-BBBB</string>
<key>PayloadIdentifier</key><string>BBBB-BBBB</string>
<key>PayloadDisplayName</key><string>Name B</string>
<key>PayloadContent</key><array/>
</dict></plist>`
	eq, err := payloadhelpers.PayloadsSemanticallyEqual([]byte(a), []byte(b))
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if !eq {
		t.Fatal("diff only in masked keys should be equal")
	}
}

func TestPayloadsSemanticallyEqual_DetectsRealChange(t *testing.T) {
	const a = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadContent</key><array/>
</dict></plist>`
	const b = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>2</integer>
<key>PayloadContent</key><array/>
</dict></plist>`
	eq, err := payloadhelpers.PayloadsSemanticallyEqual([]byte(a), []byte(b))
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if eq {
		t.Fatal("changed PayloadVersion should not be equal")
	}
}

// TestMarshalPlist_RoundTrip verifies parsePlist + marshalPlist preserves key values.
func TestMarshalPlist_RoundTrip(t *testing.T) {
	parsed, _, err := plisthelpers.ParsePlist([]byte(minimalMobileconfig))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, err := plisthelpers.MarshalPlist(parsed)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	reparsed, _, err := plisthelpers.ParsePlist(out)
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	if got, want := reparsed["PayloadType"], parsed["PayloadType"]; got != want {
		t.Errorf("PayloadType: got=%v want=%v", got, want)
	}
}

// TestInjectTopLevelIdentifiers_Preserves — injecting with existing payload
// overwrites PayloadUUID / PayloadIdentifier from the existing source.
func TestInjectTopLevelIdentifiers_Preserves(t *testing.T) {
	existing := []byte(minimalMobileconfig)
	parsed, _, err := plisthelpers.ParsePlist(existing)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	parsed["PayloadUUID"] = "NEW-UUID"
	parsed["PayloadIdentifier"] = "NEW-IDENTIFIER"
	newPayload, err := plisthelpers.MarshalPlist(parsed)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	out, err := payloadhelpers.InjectTopLevelIdentifiers(newPayload, existing)
	if err != nil {
		t.Fatalf("inject: %v", err)
	}
	check, _, err := plisthelpers.ParsePlist(out)
	if err != nil {
		t.Fatalf("parse output: %v", err)
	}
	if got, want := check["PayloadUUID"], "AAAABBBB-1111-2222-3333-444455556666"; got != want {
		t.Errorf("PayloadUUID: got=%v want=%v", got, want)
	}
	if got, want := check["PayloadIdentifier"], "AAAABBBB-1111-2222-3333-444455556666"; got != want {
		t.Errorf("PayloadIdentifier: got=%v want=%v", got, want)
	}
}

// TestInjectTopLevelIdentifiers_NilExisting_NoOp — create path; no existing state.
func TestInjectTopLevelIdentifiers_NilExisting_NoOp(t *testing.T) {
	new := []byte(minimalMobileconfig)
	out, err := payloadhelpers.InjectTopLevelIdentifiers(new, nil)
	if err != nil {
		t.Fatalf("inject: %v", err)
	}
	if string(out) != string(new) {
		t.Fatal("expected no-op when existing is nil")
	}
}

// TestExtractServerPayloadFromGeneral_DecodesEntities — verifies the helper
// decodes entity-encoded plist from a classic-API GET response.
func TestExtractServerPayloadFromGeneral_DecodesEntities(t *testing.T) {
	payload, err := payloadhelpers.ExtractServerPayloadFromGeneral([]byte(minimalWireXML))
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if _, _, err := plisthelpers.ParsePlist(payload); err != nil {
		t.Fatalf("extracted bytes are not valid plist: %v", err)
	}
}
