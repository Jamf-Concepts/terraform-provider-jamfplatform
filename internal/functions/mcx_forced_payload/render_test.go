// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mcx_forced_payload

import (
	"testing"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/plisthelpers"
)

// navigate returns the mcx_preference_settings dict from a rendered payload,
// failing the test if the envelope structure is wrong.
func navigateSettings(t *testing.T, out []byte, domain string) map[string]any {
	t.Helper()
	parsed, _, err := plisthelpers.ParsePlist(out)
	if err != nil {
		t.Fatalf("output is not valid plist: %v", err)
	}
	content, ok := parsed["PayloadContent"].([]any)
	if !ok || len(content) != 1 {
		t.Fatalf("PayloadContent: expected 1-element array, got %#v", parsed["PayloadContent"])
	}
	inner, ok := content[0].(map[string]any)
	if !ok {
		t.Fatalf("PayloadContent[0]: not a dict: %#v", content[0])
	}
	if inner["PayloadType"] != "com.apple.ManagedClient.preferences" {
		t.Fatalf("inner PayloadType: got %v", inner["PayloadType"])
	}
	pc, ok := inner["PayloadContent"].(map[string]any)
	if !ok {
		t.Fatalf("inner PayloadContent: not a dict: %#v", inner["PayloadContent"])
	}
	dom, ok := pc[domain].(map[string]any)
	if !ok {
		t.Fatalf("domain %q dict missing: %#v", domain, pc)
	}
	forced, ok := dom["Forced"].([]any)
	if !ok || len(forced) != 1 {
		t.Fatalf("Forced: expected 1-element array, got %#v", dom["Forced"])
	}
	f0, ok := forced[0].(map[string]any)
	if !ok {
		t.Fatalf("Forced[0]: not a dict: %#v", forced[0])
	}
	settings, ok := f0["mcx_preference_settings"].(map[string]any)
	if !ok {
		t.Fatalf("mcx_preference_settings missing: %#v", f0)
	}
	return settings
}

func TestRender_BuildsManagedClientPreferencesEnvelope(t *testing.T) {
	out, err := renderMCXForcedPayload("com.example.app", map[string]any{
		"AdminBase": "https://admin.example.com",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	settings := navigateSettings(t, out, "com.example.app")
	if settings["AdminBase"] != "https://admin.example.com" {
		t.Fatalf("AdminBase: got %v", settings["AdminBase"])
	}
}

// asInt64 accepts howett's int64/uint64 integer representations and fails if the
// value came back as a float64 (i.e. was rendered as <real>, not <integer>).
func asInt64(t *testing.T, v any) int64 {
	t.Helper()
	switch n := v.(type) {
	case int64:
		return n
	case uint64:
		return int64(n)
	case float64:
		t.Fatalf("value rendered as <real> (%v); want <integer>", n)
	default:
		t.Fatalf("value is not numeric: %#v", v)
	}
	return 0
}

func TestRender_WholeNumberRendersAsInteger(t *testing.T) {
	out, err := renderMCXForcedPayload("com.example.app", map[string]any{
		// Terraform hands whole numbers to the provider as float64.
		"RotateWithinHours": float64(24),
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	settings := navigateSettings(t, out, "com.example.app")
	if got := asInt64(t, settings["RotateWithinHours"]); got != 24 {
		t.Fatalf("RotateWithinHours: got %d, want 24", got)
	}
}

func TestRender_FractionalNumberRendersAsReal(t *testing.T) {
	out, err := renderMCXForcedPayload("com.example.app", map[string]any{
		"Ratio": float64(1.5),
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	settings := navigateSettings(t, out, "com.example.app")
	if got, ok := settings["Ratio"].(float64); !ok || got != 1.5 {
		t.Fatalf("Ratio: got %#v, want float64(1.5)", settings["Ratio"])
	}
}

func TestRender_StringArray(t *testing.T) {
	out, err := renderMCXForcedPayload("com.example.app", map[string]any{
		"Browsers": []any{"edge", "chrome", "firefox"},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	settings := navigateSettings(t, out, "com.example.app")
	arr, ok := settings["Browsers"].([]any)
	if !ok || len(arr) != 3 || arr[0] != "edge" || arr[2] != "firefox" {
		t.Fatalf("Browsers: got %#v", settings["Browsers"])
	}
}

func TestRender_NestedDict(t *testing.T) {
	// A key whose value is a nested dict must recurse into the sub-dict.
	out, err := renderMCXForcedPayload("com.example.app", map[string]any{
		"policy": map[string]any{"mode": "strict"},
		"env":    map[string]any{"FEATURE_FLAG": "true"},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	settings := navigateSettings(t, out, "com.example.app")
	policy, ok := settings["policy"].(map[string]any)
	if !ok || policy["mode"] != "strict" {
		t.Fatalf("policy: got %#v", settings["policy"])
	}
}

func TestRender_EscapesXMLMetacharacters(t *testing.T) {
	// howett escapes on write; the value must round-trip byte-for-byte.
	const raw = `a < b & c > d`
	out, err := renderMCXForcedPayload("com.example.app", map[string]any{"Expr": raw})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	settings := navigateSettings(t, out, "com.example.app")
	if settings["Expr"] != raw {
		t.Fatalf("Expr did not round-trip: got %q, want %q", settings["Expr"], raw)
	}
}

func TestRender_RejectsEmptyDomainAndPrefs(t *testing.T) {
	if _, err := renderMCXForcedPayload("", map[string]any{"a": "b"}); err == nil {
		t.Fatal("expected error for empty domain")
	}
	if _, err := renderMCXForcedPayload("com.example.app", map[string]any{}); err == nil {
		t.Fatal("expected error for empty preferences")
	}
}

// Pin the identity the mcx envelope derives. This is the highest-blast-radius
// path: existing deployed profiles are keyed on these exact identifiers, so a
// change to the seed formula would churn every profile on next apply. The bare
// structural tests above passed both before and after the shared-core refactor
// and so could not catch such a change — this asserts the actual values.
//
//	top-level PayloadIdentifier = domain (verbatim)
//	top-level PayloadUUID       = uuidv5(dns, domain)
//	inner PayloadIdentifier/UUID = uuidv5(dns, "<domain>|com.apple.ManagedClient.preferences|0")
//
// The expected UUIDs are hard-coded golden literals, computed independently with
// Python's uuid.uuid5(NAMESPACE_DNS, ...) — NOT via the production helper — so a
// change to the derivation is caught rather than tracked.
func TestRender_PinsEnvelopeIdentity(t *testing.T) {
	const domain = "com.example.app"
	const wantProfileUUID = "280e37e9-46f9-5e07-8121-4dd6c340d0f6"
	const wantInnerUUID = "c2073495-dd85-567a-8214-21d687c5b9ef"

	out, err := renderMCXForcedPayload(domain, map[string]any{"AdminBase": "x"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	parsed, _, err := plisthelpers.ParsePlist(out)
	if err != nil {
		t.Fatalf("output is not valid plist: %v", err)
	}

	if parsed["PayloadIdentifier"] != domain {
		t.Fatalf("top-level PayloadIdentifier: got %v, want %q", parsed["PayloadIdentifier"], domain)
	}
	if parsed["PayloadUUID"] != wantProfileUUID {
		t.Fatalf("top-level PayloadUUID: got %v, want %v", parsed["PayloadUUID"], wantProfileUUID)
	}

	inner := parsed["PayloadContent"].([]any)[0].(map[string]any)
	if inner["PayloadUUID"] != wantInnerUUID {
		t.Fatalf("inner PayloadUUID: got %v, want %v", inner["PayloadUUID"], wantInnerUUID)
	}
	if inner["PayloadIdentifier"] != wantInnerUUID {
		t.Fatalf("inner PayloadIdentifier: got %v, want %v", inner["PayloadIdentifier"], wantInnerUUID)
	}
}
