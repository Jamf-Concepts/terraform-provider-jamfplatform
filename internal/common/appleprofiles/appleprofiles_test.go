// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package appleprofiles

import (
	"encoding/json"
	"strings"
	"testing"
)

// settings decodes a JSON object the way the provider does before validating it, so the tests carry
// the same float64-shaped numbers the real path sees.
func settings(t *testing.T, encoded string) map[string]any {
	t.Helper()
	var decoded map[string]any
	if err := json.Unmarshal([]byte(encoded), &decoded); err != nil {
		t.Fatalf("failed to decode settings: %v", err)
	}
	return decoded
}

func TestTableLoads(t *testing.T) {
	commit, release := Provenance()
	if len(commit) != 40 {
		t.Errorf("expected a 40-character upstream commit, got %q", commit)
	}
	if release == "" {
		t.Error("expected the upstream release recorded")
	}
	if got := len(PayloadTypes()); got < 100 {
		t.Errorf("expected the table to carry Apple's payload types, got %d", got)
	}
}

func TestValidate_CleanPayload(t *testing.T) {
	problems := Validate("com.apple.applicationaccess", settings(t, `{"allowCamera":true,"allowSafariPrivateBrowsing":false}`))
	if len(problems) != 0 {
		t.Errorf("expected no problems, got %v", problems)
	}
}

func TestValidate_UnknownKey(t *testing.T) {
	problems := Validate("com.apple.applicationaccess", settings(t, `{"bogusKeyOneTwoThree":"x"}`))
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %v", problems)
	}
	if problems[0].Kind != UnknownKey {
		t.Errorf("expected UnknownKey, got %v", problems[0].Kind)
	}
	if !problems[0].Advisory() {
		t.Error("expected an unknown key to be advisory — a key Apple added since this snapshot looks identical")
	}
}

func TestValidate_MiscasedKey(t *testing.T) {
	problems := Validate("com.apple.applicationaccess", settings(t, `{"AllowCamera":true}`))
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %v", problems)
	}
	if problems[0].Kind != MiscasedKey {
		t.Errorf("expected MiscasedKey, got %v", problems[0].Kind)
	}
	if problems[0].Canonical != "allowCamera" {
		t.Errorf("expected Apple's spelling offered, got %q", problems[0].Canonical)
	}
	if problems[0].Advisory() {
		t.Error("expected a miscased key not to be advisory — Jamf respells it, so the plan never converges")
	}
}

func TestValidate_WrongType(t *testing.T) {
	problems := Validate("com.apple.applicationaccess", settings(t, `{"allowScreenShot":"yes"}`))
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %v", problems)
	}
	if problems[0].Kind != WrongType {
		t.Errorf("expected WrongType, got %v", problems[0].Kind)
	}
	if !strings.Contains(problems[0].Detail, "a boolean") || !strings.Contains(problems[0].Detail, "a string") {
		t.Errorf("expected the detail to name both types, got %q", problems[0].Detail)
	}
}

func TestValidate_IntegerAcceptsWholeNumberRejectsFraction(t *testing.T) {
	if problems := Validate("com.apple.applicationaccess", settings(t, `{"enforcedSoftwareUpdateDelay":30}`)); len(problems) != 0 {
		t.Errorf("expected a whole number to satisfy an integer, got %v", problems)
	}
	problems := Validate("com.apple.applicationaccess", settings(t, `{"enforcedSoftwareUpdateDelay":30.5}`))
	if len(problems) != 1 || problems[0].Kind != WrongType {
		t.Errorf("expected a fractional value to fail an integer, got %v", problems)
	}
}

func TestValidate_MissingRequiredKey(t *testing.T) {
	problems := Validate("com.apple.notificationsettings", settings(t, `{}`))
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %v", problems)
	}
	if problems[0].Kind != MissingRequiredKey || problems[0].Path != "NotificationSettings" {
		t.Errorf("expected the required top-level key reported, got %v", problems[0])
	}
}

func TestValidate_MissingRequiredKeyInsideArrayEntry(t *testing.T) {
	problems := Validate("com.apple.notificationsettings", settings(t, `{"NotificationSettings":[{"AlertType":1}]}`))
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %v", problems)
	}
	if problems[0].Path != "NotificationSettings[0].BundleIdentifier" {
		t.Errorf("expected the nested required key reported with an indexed path, got %q", problems[0].Path)
	}
	if problems[0].Kind != MissingRequiredKey {
		t.Errorf("expected MissingRequiredKey, got %v", problems[0].Kind)
	}
}

func TestValidate_NullIsTreatedAsAbsent(t *testing.T) {
	// A null is discarded by Jamf, so it is never a type error — but under a required key it is a
	// missing key, because the stored payload will not carry it.
	problems := Validate("com.apple.notificationsettings", settings(t,
		`{"NotificationSettings":[{"BundleIdentifier":"com.example.app","PreviewType":null,"GroupingType":null}]}`))
	if len(problems) != 0 {
		t.Errorf("expected nulls to be ignored, got %v", problems)
	}

	problems = Validate("com.apple.notificationsettings", settings(t, `{"NotificationSettings":null}`))
	if len(problems) != 1 || problems[0].Kind != MissingRequiredKey {
		t.Errorf("expected a null required key reported as missing, got %v", problems)
	}
}

func TestValidate_UnknownPayloadType(t *testing.T) {
	problems := Validate("com.example.notarealpayload", settings(t, `{"foo":"bar"}`))
	if len(problems) != 1 || problems[0].Kind != UnknownPayloadType {
		t.Fatalf("expected UnknownPayloadType, got %v", problems)
	}
	if !problems[0].Advisory() {
		t.Error("expected an unknown payload type to be advisory — Jamf may support a type Apple does not document")
	}
	if problems[0].Path != "" {
		t.Errorf("expected an empty path for a payload-level problem, got %q", problems[0].Path)
	}
}

func TestValidate_MiscasedPayloadType(t *testing.T) {
	// Jamf matches the payload type case-sensitively and rejects the write, unlike a key.
	problems := Validate("com.apple.managedclient.preferences", settings(t, `{}`))
	if len(problems) != 1 || problems[0].Kind != MiscasedPayloadType {
		t.Fatalf("expected MiscasedPayloadType, got %v", problems)
	}
	if problems[0].Canonical != "com.apple.ManagedClient.preferences" {
		t.Errorf("expected Apple's spelling offered, got %q", problems[0].Canonical)
	}
	if problems[0].Advisory() {
		t.Error("expected a miscased payload type not to be advisory — Jamf rejects the write")
	}
}

func TestValidate_FreeFormSubtreeIsNotValidated(t *testing.T) {
	// Everything below an MCX preference domain is free-form: Jamf stored arbitrary keys there
	// verbatim, and even accepted an element that omitted mcx_preference_settings, so validating
	// past the wildcard would manufacture findings the service does not agree with.
	problems := Validate("com.apple.ManagedClient.preferences", settings(t,
		`{"PayloadContent":{"com.example.notarealapp":{"Forced":[{"mcx_preference_settings":{"totallyMadeUp":"whatever","nested":{"deep":[1,2,{"x":true}]}}}]}}}`))
	if len(problems) != 0 {
		t.Errorf("expected an MCX payload to pass untouched, got %v", problems)
	}

	problems = Validate("com.apple.ManagedClient.preferences", settings(t,
		`{"PayloadContent":{"com.example.app":{"Forced":[{"wrongInnerKey":{"a":1}}]}}}`))
	if len(problems) != 0 {
		t.Errorf("expected no findings below the wildcard, got %v", problems)
	}
}

func TestValidate_MCXRequiresPayloadContent(t *testing.T) {
	// The envelope itself is still validated: Jamf rejects an MCX payload with no PayloadContent.
	problems := Validate("com.apple.ManagedClient.preferences", settings(t, `{}`))
	if len(problems) != 1 || problems[0].Kind != MissingRequiredKey || problems[0].Path != "PayloadContent" {
		t.Errorf("expected PayloadContent reported as required, got %v", problems)
	}
}

func TestValidate_CommonPayloadKeysAccepted(t *testing.T) {
	// Jamf echoes an authored value for the common metadata keys, so an author who sets one must not
	// be told it is unrecognised — and must not be told they are required either, since the provider
	// and the service supply them.
	problems := Validate("com.apple.applicationaccess", settings(t,
		`{"payloadDisplayName":"My Own Name","PayloadOrganization":"Acme","allowCamera":true}`))
	for _, problem := range problems {
		if problem.Kind == UnknownKey || problem.Kind == MissingRequiredKey {
			t.Errorf("unexpected problem for a common payload key: %v", problem)
		}
	}
}

func TestValidate_WildcardPayloadAcceptsAnyKey(t *testing.T) {
	problems := Validate("com.apple.firstactiveethernet.managed", settings(t, `{"anythingAtAll":123}`))
	if len(problems) != 0 {
		t.Errorf("expected a wildcard payload to accept any key, got %v", problems)
	}
}

func TestValidate_ProblemsAreOrderedByPath(t *testing.T) {
	problems := Validate("com.apple.applicationaccess", settings(t, `{"zzBogus":1,"aaBogus":2}`))
	if len(problems) != 2 {
		t.Fatalf("expected 2 problems, got %v", problems)
	}
	if problems[0].Path != "aaBogus" || problems[1].Path != "zzBogus" {
		t.Errorf("expected problems ordered by path, got %q then %q", problems[0].Path, problems[1].Path)
	}
}

func TestValidate_IntegerOutOfJamfRange(t *testing.T) {
	// Apple declares CacheLimit unbounded; Jamf stores a 32-bit signed integer. Wire probing found
	// 2147483647 accepted and 2147483648 rejected with a validation failure.
	if problems := Validate("com.apple.AssetCache.managed", settings(t, `{"CacheLimit":2147483647}`)); len(problems) != 0 {
		t.Errorf("expected int32 max to be accepted, got %v", problems)
	}

	problems := Validate("com.apple.AssetCache.managed", settings(t, `{"CacheLimit":2147483648}`))
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %v", problems)
	}
	if problems[0].Kind != IntegerOutOfRange {
		t.Errorf("expected IntegerOutOfRange, got %v", problems[0].Kind)
	}
	if problems[0].Advisory() {
		t.Error("expected an out-of-range integer not to be advisory — Jamf rejects the write")
	}
	if !strings.Contains(problems[0].Detail, "2147483648") {
		t.Errorf("expected the offending value in the detail without exponent notation, got %q", problems[0].Detail)
	}
}

func TestValidate_NegativeIntegerOutOfJamfRange(t *testing.T) {
	problems := Validate("com.apple.AssetCache.managed", settings(t, `{"CacheLimit":-2147483649}`))
	if len(problems) != 1 || problems[0].Kind != IntegerOutOfRange {
		t.Errorf("expected IntegerOutOfRange below the 32-bit floor, got %v", problems)
	}
}

func TestValidate_IntegerRangeCheckedInsideNesting(t *testing.T) {
	// The bound applies wherever an integer appears, not only at the top level.
	problems := Validate("com.apple.notificationsettings", settings(t,
		`{"NotificationSettings":[{"BundleIdentifier":"com.example.app","AlertType":2147483648}]}`))
	if len(problems) != 1 || problems[0].Kind != IntegerOutOfRange {
		t.Fatalf("expected IntegerOutOfRange, got %v", problems)
	}
	if problems[0].Path != "NotificationSettings[0].AlertType" {
		t.Errorf("expected an indexed path, got %q", problems[0].Path)
	}
}
