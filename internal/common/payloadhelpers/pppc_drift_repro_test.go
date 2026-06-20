// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package payloadhelpers

import "testing"

const pppcBaseline = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadIdentifier</key><string>top-id</string>
<key>PayloadUUID</key><string>top-uuid</string>
<key>PayloadContent</key><array><dict>
<key>PayloadType</key><string>com.apple.TCC.configuration-profile-policy</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadIdentifier</key><string>tcc-id</string>
<key>PayloadUUID</key><string>tcc-uuid</string>
<key>Services</key><dict>
<key>ScreenCapture</key><array><dict>
<key>Authorization</key><string>Allow</string>
<key>Identifier</key><string>com.example.app</string>
<key>CodeRequirement</key><string>req</string>
<key>IdentifierType</key><string>bundleID</string>
</dict></array>
</dict>
</dict></array>
</dict></plist>`

const pppcPlusAccessibility = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadIdentifier</key><string>top-id</string>
<key>PayloadUUID</key><string>top-uuid</string>
<key>PayloadContent</key><array><dict>
<key>PayloadType</key><string>com.apple.TCC.configuration-profile-policy</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadIdentifier</key><string>tcc-id</string>
<key>PayloadUUID</key><string>tcc-uuid</string>
<key>Services</key><dict>
<key>ScreenCapture</key><array><dict>
<key>Authorization</key><string>Allow</string>
<key>Identifier</key><string>com.example.app</string>
<key>CodeRequirement</key><string>req</string>
<key>IdentifierType</key><string>bundleID</string>
</dict></array>
<key>Accessibility</key><array><dict>
<key>Allowed</key><true/>
<key>Identifier</key><string>com.example.app</string>
<key>CodeRequirement</key><string>req</string>
<key>IdentifierType</key><string>bundleID</string>
</dict></array>
</dict>
</dict></array>
</dict></plist>`

// Documents the limitation of the legacy two-way PayloadsSemanticallyEqual
// when faced with admin-side UI additions to a PPPC profile. Intersection
// semantics drop the asymmetric `Accessibility` key, so the two-way compare
// reports equal even though the server gained a service since plan was
// authored. The three-way ThreeWayCompare (threeway.go) is the path that
// reliably surfaces this — see TestThreeWayCompare_AdminAddedService.
//
// This test pins the legacy behaviour so a future change to
// PayloadsSemanticallyEqual (e.g. removing intersection semantics in favour
// of a universal strict body compare) cannot silently regress the Jamf-side
// strip cases that intersection semantics also absorbs.
func TestPPPC_LegacyTwoWayMissesAdminAdd(t *testing.T) {
	eq, err := PayloadsSemanticallyEqual([]byte(pppcBaseline), []byte(pppcPlusAccessibility))
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if !eq {
		t.Fatal("legacy two-way compare is expected to miss admin-added services; ThreeWayCompare is the fix path")
	}
}
