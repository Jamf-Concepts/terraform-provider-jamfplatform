// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobileconfig

import (
	"testing"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/plisthelpers"
)

func parseDoc(t *testing.T, out []byte) map[string]any {
	t.Helper()
	parsed, _, err := plisthelpers.ParsePlist(out)
	if err != nil {
		t.Fatalf("output is not valid plist: %v", err)
	}
	return parsed
}

func payloadArray(t *testing.T, doc map[string]any) []any {
	t.Helper()
	pc, ok := doc["PayloadContent"].([]any)
	if !ok {
		t.Fatalf("top-level PayloadContent is not an array: %#v", doc["PayloadContent"])
	}
	return pc
}

func TestAssemble_WrapsMultiplePayloads(t *testing.T) {
	out, err := Assemble(Profile{
		DisplayName: "Example Dock + Custom",
		Identifier:  "com.example.mixed",
		Payloads: []map[string]any{
			{"PayloadType": "com.apple.dock", "tilesize": float64(48)},
			{"PayloadType": "com.acme.custom", "flag": true},
		},
	})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	doc := parseDoc(t, out)

	if doc["PayloadType"] != "Configuration" {
		t.Fatalf("top-level PayloadType: got %v, want Configuration", doc["PayloadType"])
	}
	pc := payloadArray(t, doc)
	if len(pc) != 2 {
		t.Fatalf("expected 2 payloads, got %d", len(pc))
	}

	first := pc[0].(map[string]any)
	if first["PayloadType"] != "com.apple.dock" {
		t.Fatalf("payload[0] PayloadType: got %v", first["PayloadType"])
	}
	// tilesize (whole number) must render as <integer>, not <real>.
	if _, isFloat := first["tilesize"].(float64); isFloat {
		t.Fatalf("tilesize rendered as <real>; want <integer>")
	}
	// Identity keys must be injected when the author omits them.
	if _, ok := first["PayloadUUID"]; !ok {
		t.Fatalf("payload[0] missing injected PayloadUUID")
	}
	if _, ok := first["PayloadIdentifier"]; !ok {
		t.Fatalf("payload[0] missing injected PayloadIdentifier")
	}
	if first["PayloadVersion"] == nil {
		t.Fatalf("payload[0] missing injected PayloadVersion")
	}
}

func TestAssemble_DeterministicOutput(t *testing.T) {
	p := Profile{
		Identifier: "com.example.dock",
		Payloads:   []map[string]any{{"PayloadType": "com.apple.dock", "tilesize": float64(48)}},
	}
	a, err := Assemble(p)
	if err != nil {
		t.Fatalf("assemble a: %v", err)
	}
	b, err := Assemble(p)
	if err != nil {
		t.Fatalf("assemble b: %v", err)
	}
	if string(a) != string(b) {
		t.Fatal("identical input produced different output; identity is not deterministic (would churn plans)")
	}
}

func TestAssemble_RespectsAuthorSuppliedIdentity(t *testing.T) {
	out, err := Assemble(Profile{
		Identifier: "com.example.dock",
		Payloads: []map[string]any{{
			"PayloadType":       "com.apple.dock",
			"PayloadUUID":       "AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE",
			"PayloadIdentifier": "com.example.custom.dock",
		}},
	})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	p0 := payloadArray(t, parseDoc(t, out))[0].(map[string]any)
	if p0["PayloadUUID"] != "AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE" {
		t.Fatalf("author PayloadUUID overwritten: got %v", p0["PayloadUUID"])
	}
	if p0["PayloadIdentifier"] != "com.example.custom.dock" {
		t.Fatalf("author PayloadIdentifier overwritten: got %v", p0["PayloadIdentifier"])
	}
}

func TestAssemble_Guards(t *testing.T) {
	if _, err := Assemble(Profile{}); err == nil {
		t.Fatal("expected error for zero payloads")
	}
	if _, err := Assemble(Profile{Payloads: []map[string]any{{"tilesize": float64(48)}}}); err == nil {
		t.Fatal("expected error for payload missing PayloadType")
	}
	// identifier is mandatory: it seeds every identity UUID, and omitting it
	// risks cross-profile identifier collisions.
	if _, err := Assemble(Profile{
		Payloads: []map[string]any{{"PayloadType": "com.apple.dock", "tilesize": float64(48)}},
	}); err == nil {
		t.Fatal("expected error when identifier is omitted")
	}
	if _, err := Assemble(Profile{
		Identifier: "   ",
		Payloads:   []map[string]any{{"PayloadType": "com.apple.dock"}},
	}); err == nil {
		t.Fatal("expected error when identifier is blank/whitespace")
	}
}

// Pin the exact injected identity values. Bare structural tests ("a PayloadUUID
// exists") passed for both the pre- and post-refactor code and so could not
// catch an identity-seed change that would churn every deployed profile. These
// assert the actual derived UUIDs, locking the seed formula:
//
//	profile UUID / identifier  = uuidv5(dns, identifier)  [identifier verbatim]
//	payload UUID / identifier  = uuidv5(dns, "<id>|<PayloadType>|<index>")
//
// Golden literals computed independently with Python's uuid.uuid5(NAMESPACE_DNS,
// ...) rather than via the production uuidv5 — so the assertion cross-checks the
// derivation instead of tautologically re-deriving it.
func TestAssemble_PinsDerivedIdentity(t *testing.T) {
	const id = "com.example.dock"
	const wantProfileUUID = "b66842cc-32f8-5342-bce3-f05c7f509986"
	const wantPayloadUUID = "bffd4905-a0a7-5eff-b0e6-270ef987544d"

	out, err := Assemble(Profile{
		Identifier: id,
		Payloads:   []map[string]any{{"PayloadType": "com.apple.dock", "tilesize": float64(48)}},
	})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	doc := parseDoc(t, out)

	if doc["PayloadUUID"] != wantProfileUUID {
		t.Fatalf("top-level PayloadUUID: got %v, want %v", doc["PayloadUUID"], wantProfileUUID)
	}
	if doc["PayloadIdentifier"] != id {
		t.Fatalf("top-level PayloadIdentifier: got %v, want %v (verbatim identifier)", doc["PayloadIdentifier"], id)
	}

	p0 := payloadArray(t, doc)[0].(map[string]any)
	if p0["PayloadUUID"] != wantPayloadUUID {
		t.Fatalf("payload[0] PayloadUUID: got %v, want %v", p0["PayloadUUID"], wantPayloadUUID)
	}
	if p0["PayloadIdentifier"] != wantPayloadUUID {
		t.Fatalf("payload[0] PayloadIdentifier: got %v, want %v", p0["PayloadIdentifier"], wantPayloadUUID)
	}
}

// Distinct profiles that share a leading payload type must NOT collide on
// identity (the B1 regression): different identifiers must yield different
// top-level and payload UUIDs.
func TestAssemble_DistinctIdentifiersDoNotCollide(t *testing.T) {
	mk := func(id string) map[string]any {
		out, err := Assemble(Profile{
			Identifier: id,
			Payloads:   []map[string]any{{"PayloadType": "com.apple.dock"}},
		})
		if err != nil {
			t.Fatalf("assemble %s: %v", id, err)
		}
		return parseDoc(t, out)
	}
	a := mk("com.example.team-a.dock")
	b := mk("com.example.team-b.dock")
	if a["PayloadUUID"] == b["PayloadUUID"] {
		t.Fatal("distinct identifiers produced the same top-level PayloadUUID (collision)")
	}
	pa := payloadArray(t, a)[0].(map[string]any)
	pb := payloadArray(t, b)[0].(map[string]any)
	if pa["PayloadUUID"] == pb["PayloadUUID"] {
		t.Fatal("distinct identifiers produced the same payload PayloadUUID (collision)")
	}
}
