// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package payloadhelpers provides shared mobileconfig plist parsing,
// masking, and diff-suppression helpers used by all configuration profile
// resources (macOS and mobile device). All functions are pure — they operate
// on byte slices and maps and carry no resource-specific state.
package payloadhelpers

import (
	"bytes"
	"fmt"
	"html"
	"strings"

	"howett.net/plist"
)

// Server-controlled keys skipped on both sides before diff comparison.
// Only keys Jamf Pro derives on every write/read regardless of input value
// land here.
//
// Keys Jamf Pro only *conditionally* defaults (e.g. `PayloadEnabled` when
// absent, `VendorConfig` only for webcontent-filter payloads) are
// intentionally **not** in this list. The intersection-compare in
// LenientEqualPlist drops asymmetric keys at comparison time, which means:
//
//   - user omits the key → only server side has it → intersection drops it → no spurious diff.
//   - user authors the key → both sides have it → compared → drift detected if the user later edits.
//
// `PayloadOrganization` is in this list — wire-confirmed 2026-05-26 that
// Jamf Pro Classic always overwrites the field with "JAMF Software" on
// both top-level and per-PayloadContent slots, regardless of the
// user-authored value. The previous "conditional default" assumption
// produced persistent payload diffs on every plan that authored an
// organization. Trade-off: a user-authored edit to PayloadOrganization
// will be silently dropped on the wire — but Jamf already drops it
// regardless, so masking the key matches reality. Drift detection on
// every other value the user authors is unaffected.
var (
	maskedTopLevelKeys = map[string]struct{}{
		"PayloadDisplayName":  {}, // server sets from classic <general><name> on every write
		"PayloadIdentifier":   {}, // server assigns a new lowercase UUID on create
		"PayloadUUID":         {}, // server assigns a new lowercase UUID on create
		"PayloadOrganization": {}, // server always overwrites with "JAMF Software"
		"PayloadDescription":  {}, // server always empties the top-level description
		"PayloadEnabled":      {}, // server always strips at top-level (Apple-spec is per-payload, not top-level)
	}
	maskedPayloadContentKeys = map[string]struct{}{
		"PayloadDisplayName":  {}, // server-defaulted per PayloadType
		"PayloadIdentifier":   {}, // server may assign on create
		"PayloadUUID":         {}, // server may assign on create; preserved on update by InjectTopLevelIdentifiers
		"PayloadOrganization": {}, // server always overwrites with "JAMF Software"
	}
	// serverInjectedPayloadTypes are PayloadContent[i].PayloadType values Jamf
	// Pro inserts into the mobileconfig as a side-effect of a *different*
	// classic-API field. They are dropped from both sides of mask comparison
	// because the user never authored them — they exist purely because the
	// user set the sibling classic-API knob (e.g. self_service.security
	// authorization_password materialises as a com.apple.profileRemovalPassword
	// PayloadContent entry). Dropping them means the masked plan and the
	// masked server response stay length-aligned through Create/Update.
	//
	// Trade-off: a user who *manually* authors a payload entry of one of these
	// types (rare but possible) loses drift detection on it. The
	// MaskPayload-design note above explicitly rejects skip-lists for keys
	// users might author — this list intentionally accepts the carve-out
	// because every entry here is paired with a first-class classic-API field
	// the resource already exposes, and the in-payload form is redundant.
	serverInjectedPayloadTypes = map[string]struct{}{
		"com.apple.profileRemovalPassword": {}, // mirrors self_service.authorization_password
	}
)

// ParsePlist decodes a mobileconfig XML plist into a Go map. The returned
// int is howett.net/plist's format identifier (plist.XMLFormat,
// plist.BinaryFormat, etc.) — typed as int because the library exposes
// formats as untyped int constants.
func ParsePlist(raw []byte) (map[string]any, int, error) {
	var out map[string]any
	format, err := plist.Unmarshal(raw, &out)
	if err != nil {
		return nil, 0, fmt.Errorf("parsing plist: %w", err)
	}
	return out, format, nil
}

// MarshalPlist serialises a plist dict back to XML form. Used by the input
// builder when re-emitting a payload after identifier injection.
func MarshalPlist(m map[string]any) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := plist.NewEncoderForFormat(buf, plist.XMLFormat)
	enc.Indent("\t")
	if err := enc.Encode(m); err != nil {
		return nil, fmt.Errorf("encoding plist: %w", err)
	}
	return buf.Bytes(), nil
}

// CanonicalisePlistXML parses a mobileconfig plist and re-emits it as
// tab-indented XML. Used on the state-builder side to normalise Jamf's
// compact single-line wire form into the same shape user-authored
// payloads typically take (Apple's standard pretty-printed mobileconfig).
// When state and plan share formatting, Terraform's diff narrows from a
// whole-payload swap to the specific keys that changed.
//
// Falls back to returning the input unchanged if the plist fails to
// parse — the caller still has a usable string, and the legibility
// improvement is a UX nicety, not a correctness gate.
func CanonicalisePlistXML(raw []byte) []byte {
	parsed, _, err := ParsePlist(raw)
	if err != nil {
		return raw
	}
	out, err := MarshalPlist(parsed)
	if err != nil {
		return raw
	}
	return out
}

// MaskPayload returns a deep-cloned representation of the input plist with
// every server-controlled key dropped and every string value trimmed. The
// result is suitable for equality comparison against another masked payload
// — equality implies the two payloads are semantically the same modulo
// Jamf's well-known server-side normalisations.
//
// See STYLE_GUIDE.md §Configuration profile payload diff suppression for
// the diff-class catalogue this mask is designed to neutralise.
func MaskPayload(raw []byte) (map[string]any, error) {
	parsed, _, err := ParsePlist(raw)
	if err != nil {
		return nil, err
	}
	return maskTopLevel(parsed), nil
}

func maskTopLevel(p map[string]any) map[string]any {
	out := make(map[string]any, len(p))
	for k, v := range p {
		if _, masked := maskedTopLevelKeys[k]; masked {
			continue
		}
		if k == "PayloadContent" {
			if arr, ok := v.([]any); ok {
				out[k] = maskPayloadContent(arr)
				continue
			}
		}
		trimmed := trimAny(v)
		if isEmpty(trimmed) {
			continue
		}
		out[k] = trimmed
	}
	return out
}

func maskPayloadContent(items []any) []any {
	out := make([]any, 0, len(items))
	for _, it := range items {
		dict, ok := it.(map[string]any)
		if !ok {
			out = append(out, trimAny(it))
			continue
		}
		if ptype, ok := dict["PayloadType"].(string); ok {
			if _, injected := serverInjectedPayloadTypes[ptype]; injected {
				continue
			}
		}
		masked := make(map[string]any, len(dict))
		for k, v := range dict {
			if _, drop := maskedPayloadContentKeys[k]; drop {
				continue
			}
			trimmed := trimAny(v)
			if isEmpty(trimmed) {
				continue
			}
			masked[k] = trimmed
		}
		out = append(out, masked)
	}
	return out
}

// isEmpty reports whether a plist value is equivalent to "user did not
// author anything" — empty string, nil, or empty collection.
func isEmpty(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case string:
		return t == ""
	case []any:
		return len(t) == 0
	case map[string]any:
		return len(t) == 0
	default:
		return false
	}
}

// trimAny recursively trims leading/trailing whitespace from every string
// value in the plist tree.
func trimAny(v any) any {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = trimAny(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[k] = trimAny(val)
		}
		return out
	default:
		return v
	}
}

// PayloadsSemanticallyEqual returns true when the two raw mobileconfig
// payloads are equal after skipping server-controlled fields and treating
// asymmetric dict keys as a no-op (intersection compare).
//
// The intersection compare is the formal expression of the user's
// "ignore overridden fields, no mapping tables" directive: when Jamf Pro
// adds or drops a key the user didn't author, both sides differ in their
// key-sets — but a real semantic change is in the *value* of a key that
// is on both sides. The compare therefore walks the union of keys at every
// dict, but only fails when a key present on both sides has differing
// values.
//
// Trade-off: a user who edits only a key Jamf Pro itself derives (e.g.
// tunes `PayloadOrganization` from empty to a value, then back) won't see
// a diff — the server will substitute its own value, so flagging that as
// drift would produce a persistent diff loop.
func PayloadsSemanticallyEqual(a, b []byte) (bool, error) {
	ma, err := MaskPayload(a)
	if err != nil {
		return false, fmt.Errorf("masking left side: %w", err)
	}
	mb, err := MaskPayload(b)
	if err != nil {
		return false, fmt.Errorf("masking right side: %w", err)
	}
	return LenientEqualPlist(ma, mb), nil
}

// LenientEqualPlist compares two parsed-and-masked plist trees with
// intersection semantics: dict keys present on only one side are ignored;
// shared keys must compare equal. Arrays compare positionally. Scalars
// compare via numericEqual for ints (howett.net/plist returns int64 or
// uint64 depending on sign).
//
// Known limitation: keys server adds out-of-band (admin edits the
// payload directly in the Jamf Pro UI) at depths the mask doesn't cover
// will not surface as drift — the intersection compare ignores
// state-only keys. The trade-off keeps the corpus quiet on Jamf's many
// conditional-default injections (top-level `PayloadRemovalDisallowed`,
// per-payload `PayloadEnabled`, PPPC service strips, etc.) without
// having to enumerate every one. Detecting deep out-of-band UI edits
// requires a depth-aware variant; pending design.
func LenientEqualPlist(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok {
			return false
		}
		for k, va := range av {
			if vb, exists := bv[k]; exists {
				if !LenientEqualPlist(va, vb) {
					return false
				}
			}
		}
		return true
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !LenientEqualPlist(av[i], bv[i]) {
				return false
			}
		}
		return true
	case uint64:
		return numericEqual(int64(av), b)
	case int64:
		return numericEqual(av, b)
	case int:
		return numericEqual(int64(av), b)
	default:
		return a == b
	}
}

func numericEqual(a int64, b any) bool {
	switch bv := b.(type) {
	case uint64:
		return a >= 0 && uint64(a) == bv
	case int64:
		return a == bv
	case int:
		return a == int64(bv)
	default:
		return false
	}
}

// InjectTopLevelIdentifierValues overwrites the top-level PayloadUUID and
// PayloadIdentifier of newPayload with the supplied strings. The values
// should be the server-canonical identifiers sourced from state (e.g.
// state.General.UUID). Without this step, every PUT produces a new top-level
// UUID server-side and devices treat the update as a fresh install.
//
// If both uuid and identifier are empty (Create path), newPayload is
// returned unchanged.
func InjectTopLevelIdentifierValues(newPayload []byte, uuid, identifier string) ([]byte, error) {
	if uuid == "" && identifier == "" {
		return newPayload, nil
	}
	next, format, err := ParsePlist(newPayload)
	if err != nil {
		return nil, fmt.Errorf("parsing new payload for identifier injection: %w", err)
	}
	if uuid != "" {
		next["PayloadUUID"] = uuid
	}
	if identifier != "" {
		next["PayloadIdentifier"] = identifier
	}
	buf := &bytes.Buffer{}
	enc := plist.NewEncoderForFormat(buf, format)
	enc.Indent("\t")
	if err := enc.Encode(next); err != nil {
		return nil, fmt.Errorf("re-serialising payload after identifier injection: %w", err)
	}
	return buf.Bytes(), nil
}

// InjectTopLevelIdentifiers is the existingPayload-based wrapper used by
// unit tests. CRUD callers use InjectTopLevelIdentifierValues directly so
// they can pass server-canonical identifiers sourced from state.General.UUID.
func InjectTopLevelIdentifiers(newPayload, existingPayload []byte) ([]byte, error) {
	if len(existingPayload) == 0 {
		return newPayload, nil
	}
	exist, _, err := ParsePlist(existingPayload)
	if err != nil {
		return newPayload, nil //nolint:nilerr // best-effort
	}
	uuid, _ := exist["PayloadUUID"].(string)
	identifier, _ := exist["PayloadIdentifier"].(string)
	return InjectTopLevelIdentifierValues(newPayload, uuid, identifier)
}

// ExtractServerPayloadFromGeneral pulls the mobileconfig plist text out of a
// classic API GET response. The server returns the payload inside
// <general><payloads>...</payloads></general> as XML-entity-encoded text (not
// CDATA). This helper exists for tests that work directly with captured wire XML.
func ExtractServerPayloadFromGeneral(wireXML []byte) ([]byte, error) {
	const (
		openTag  = "<payloads>"
		closeTag = "</payloads>"
	)
	si := bytes.Index(wireXML, []byte(openTag))
	ei := bytes.Index(wireXML, []byte(closeTag))
	if si < 0 || ei < 0 || si >= ei {
		return nil, fmt.Errorf("no <payloads> block found")
	}
	inner := wireXML[si+len(openTag) : ei]
	inner = bytes.TrimSpace(inner)
	if bytes.HasPrefix(inner, []byte("<![CDATA[")) && bytes.HasSuffix(inner, []byte("]]>")) {
		return inner[len("<![CDATA[") : len(inner)-len("]]>")], nil
	}
	return []byte(html.UnescapeString(string(inner))), nil
}
