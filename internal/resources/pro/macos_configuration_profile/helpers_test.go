// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package macos_configuration_profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/payloadhelpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/plisthelpers"
)

// Each fixture is the original Definitions-API mobileconfig sent to the
// classic API plus the server-canonical response captured via jamf-cli get.
var fixtureCases = []struct {
	name              string
	stem              string
	hasUpdateResponse bool
}{
	{"PPPC", "DEVONthink__pppcp_profile", true},
	{"ManagedLoginItems", "1Password__managed_login_items_profile", true},
	{"ContentFilter_VendorConfigDropped", "DuckDuckGo__content_filter_profile", false},
	{"SystemExtension_AllowUserOverrides", "KarabinerElements__system_extension_profile", false},
	{"PayloadOrganization_Mutated", "MicrosoftDefender__pppcp_profile", false},
	{"Notifications", "1Password__notifications_profile", false},
	{"ScreenRecording", "1Password__screen_recording_profile", false},
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	p := filepath.Join("testdata", name)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("reading fixture %s: %v", p, err)
	}
	return b
}

func loadInputAndServer(t *testing.T, stem string) (input, serverResp []byte) {
	t.Helper()
	input = readFixture(t, stem+".mobileconfig")
	wire := readFixture(t, stem+".create_response.xml")
	srv, err := payloadhelpers.ExtractServerPayloadFromGeneral(wire)
	if err != nil {
		t.Fatalf("extracting server payload for %s: %v", stem, err)
	}
	return input, srv
}

// TestMaskPayload_SuppressesEveryKnownServerMutation asserts the mask
// neutralises every diff class observed in the 200-profile roundtrip
// corpus. This is the regression net for the "Jamf changed a mutation we
// didn't catch" failure mode — when the test fails on a future SDK bump or
// tenant upgrade, extend the mask in helpers.go.
func TestMaskPayload_SuppressesEveryKnownServerMutation(t *testing.T) {
	for _, tc := range fixtureCases {
		t.Run(tc.name, func(t *testing.T) {
			input, serverResp := loadInputAndServer(t, tc.stem)
			equal, err := payloadhelpers.PayloadsSemanticallyEqual(input, serverResp)
			if err != nil {
				t.Fatalf("comparing payloads: %v", err)
			}
			if !equal {
				ma, _ := payloadhelpers.MaskPayload(input)
				mb, _ := payloadhelpers.MaskPayload(serverResp)
				t.Fatalf("mask did not neutralise diff for %s\nmasked_input=%#v\nmasked_server=%#v", tc.stem, ma, mb)
			}
		})
	}
}

// TestPayloadsSemanticallyEqual_DetectsRealChange asserts the mask does NOT
// over-suppress. Modifying a non-masked field in the input should produce a
// diff so the resource still detects genuine drift.
func TestPayloadsSemanticallyEqual_DetectsRealChange(t *testing.T) {
	_, serverResp := loadInputAndServer(t, "1Password__managed_login_items_profile")
	// Mutate the server payload to flip a non-masked field — Rules array
	// inside PayloadContent[0] has a `Comment` field that survives masking.
	parsed, _, err := plisthelpers.ParsePlist(serverResp)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	pc := parsed["PayloadContent"].([]any)
	first := pc[0].(map[string]any)
	rules := first["Rules"].([]any)
	rule0 := rules[0].(map[string]any)
	rule0["Comment"] = "tampered comment value"
	mutated, err := plisthelpers.MarshalPlist(parsed)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	equal, err := payloadhelpers.PayloadsSemanticallyEqual(serverResp, mutated)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if equal {
		t.Fatal("mask over-suppressed: tampered Rules[0].Comment should produce a diff")
	}
}

// TestMaskPayload_StripsWhitespaceFromNestedStrings — leading/trailing
// whitespace inside any string value (including inside nested arrays like
// PayloadContent[i].Rules[j].Comment) must be trimmed so that the input
// `" Allow X"` equals the server's `"Allow X"`.
func TestMaskPayload_StripsWhitespaceFromNestedStrings(t *testing.T) {
	const plistWithLeadingWS = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadContent</key><array><dict>
<key>PayloadType</key><string>com.apple.servicemanagement</string>
<key>PayloadVersion</key><integer>1</integer>
<key>Rules</key><array><dict>
<key>Comment</key><string>   Allow Launch Item   </string>
<key>RuleType</key><string>BundleIdentifier</string>
<key>RuleValue</key><string>com.example</string>
</dict></array>
</dict></array>
</dict></plist>`
	const plistTrimmed = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadVersion</key><integer>1</integer>
<key>PayloadContent</key><array><dict>
<key>PayloadType</key><string>com.apple.servicemanagement</string>
<key>PayloadVersion</key><integer>1</integer>
<key>Rules</key><array><dict>
<key>Comment</key><string>Allow Launch Item</string>
<key>RuleType</key><string>BundleIdentifier</string>
<key>RuleValue</key><string>com.example</string>
</dict></array>
</dict></array>
</dict></plist>`
	equal, err := payloadhelpers.PayloadsSemanticallyEqual([]byte(plistWithLeadingWS), []byte(plistTrimmed))
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if !equal {
		t.Fatal("whitespace-trim semantics: leading/trailing whitespace in nested string did not collapse")
	}
}

// TestInjectTopLevelIdentifiers_PreservesUUIDAndIdentifier — the helper
// must overwrite the top-level PayloadUUID and PayloadIdentifier of the new
// payload with values from the existing payload, leaving other fields
// untouched.
func TestInjectTopLevelIdentifiers_PreservesUUIDAndIdentifier(t *testing.T) {
	_, existing := loadInputAndServer(t, "1Password__managed_login_items_profile")
	// Build a "new" payload by mutating the existing one's top-level UUIDs.
	parsed, _, err := plisthelpers.ParsePlist(existing)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	parsed["PayloadUUID"] = "00000000-NEW-NEW-NEW-000000000000"
	parsed["PayloadIdentifier"] = "00000000-NEW-NEW-NEW-000000000000"
	newPayload, err := plisthelpers.MarshalPlist(parsed)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	out, err := payloadhelpers.InjectTopLevelIdentifiers(newPayload, existing)
	if err != nil {
		t.Fatalf("inject: %v", err)
	}
	// Re-parse and verify the identifiers came from `existing`.
	check, _, err := plisthelpers.ParsePlist(out)
	if err != nil {
		t.Fatalf("parse output: %v", err)
	}
	existingParsed, _, _ := plisthelpers.ParsePlist(existing)
	if got, want := check["PayloadUUID"], existingParsed["PayloadUUID"]; got != want {
		t.Errorf("PayloadUUID not preserved: got=%v want=%v", got, want)
	}
	if got, want := check["PayloadIdentifier"], existingParsed["PayloadIdentifier"]; got != want {
		t.Errorf("PayloadIdentifier not preserved: got=%v want=%v", got, want)
	}
}

// TestInjectTopLevelIdentifiers_EmptyExisting_NoOp — Create path: there is
// no existing payload to source identifiers from; helper must return the
// new payload unchanged.
func TestInjectTopLevelIdentifiers_EmptyExisting_NoOp(t *testing.T) {
	new := readFixture(t, "1Password__managed_login_items_profile.mobileconfig")
	out, err := payloadhelpers.InjectTopLevelIdentifiers(new, nil)
	if err != nil {
		t.Fatalf("inject: %v", err)
	}
	if string(out) != string(new) {
		t.Fatal("expected no-op when existing payload is empty")
	}
}

// TestInjectTopLevelIdentifiers_ExistingUnparseable_NoOp — never break a
// Create just because state was corrupted; pass the new payload through.
func TestInjectTopLevelIdentifiers_ExistingUnparseable_NoOp(t *testing.T) {
	new := readFixture(t, "1Password__managed_login_items_profile.mobileconfig")
	out, err := payloadhelpers.InjectTopLevelIdentifiers(new, []byte("not a plist"))
	if err != nil {
		t.Fatalf("inject: %v", err)
	}
	if string(out) != string(new) {
		t.Fatal("expected no-op when existing payload is unparseable")
	}
}

// TestExtractServerPayloadFromGeneral_DecodesEntities — the classic API
// returns the inner mobileconfig with XML entity references rather than
// CDATA. Confirm the helper round-trips a real captured response.
func TestExtractServerPayloadFromGeneral_DecodesEntities(t *testing.T) {
	wire := readFixture(t, "DEVONthink__pppcp_profile.create_response.xml")
	payload, err := payloadhelpers.ExtractServerPayloadFromGeneral(wire)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if _, _, err := plisthelpers.ParsePlist(payload); err != nil {
		t.Fatalf("extracted bytes did not parse as plist: %v", err)
	}
}

// TestUpdateIdempotency_CreateResponse_EqualsUpdateResponse — for the two
// fixtures that captured both create and update responses, the parsed
// plists are equal after masking. This confirms the resource's planned
// Update path (inject top-level identifiers; PUT) leaves the server-side
// payload unchanged across repeated applies — the "no ghost profile"
// property.
func TestUpdateIdempotency_CreateResponse_EqualsUpdateResponse(t *testing.T) {
	for _, tc := range fixtureCases {
		if !tc.hasUpdateResponse {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			createWire := readFixture(t, tc.stem+".create_response.xml")
			updateWire := readFixture(t, tc.stem+".update_response.xml")
			createPayload, err := payloadhelpers.ExtractServerPayloadFromGeneral(createWire)
			if err != nil {
				t.Fatalf("create extract: %v", err)
			}
			updatePayload, err := payloadhelpers.ExtractServerPayloadFromGeneral(updateWire)
			if err != nil {
				t.Fatalf("update extract: %v", err)
			}
			equal, err := payloadhelpers.PayloadsSemanticallyEqual(createPayload, updatePayload)
			if err != nil {
				t.Fatalf("compare: %v", err)
			}
			if !equal {
				t.Fatalf("update response diverged from create response for %s — InjectIdentifiers may be insufficient", tc.stem)
			}
		})
	}
}
