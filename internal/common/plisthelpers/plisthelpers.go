// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package plisthelpers provides generic, resource-agnostic plist (Apple
// Property List) primitives: parse, marshal, canonicalise, and structural
// equality. It carries no mobileconfig- or resource-specific knowledge — the
// configuration-profile masking / lenient-compare logic that layers on top of
// these primitives lives in the payloadhelpers package, which imports this one.
package plisthelpers

import (
	"bytes"
	"fmt"

	"howett.net/plist"
)

// ParsePlist decodes a plist (XML or binary) into a Go map. The returned int is
// howett.net/plist's format identifier (plist.XMLFormat, plist.BinaryFormat,
// etc.) — typed as int because the library exposes formats as untyped int
// constants. A bare `<dict>…</dict>` fragment (no <plist> wrapper) parses fine.
func ParsePlist(raw []byte) (map[string]any, int, error) {
	var out map[string]any
	format, err := plist.Unmarshal(raw, &out)
	if err != nil {
		return nil, 0, fmt.Errorf("parsing plist: %w", err)
	}
	return out, format, nil
}

// MarshalPlist serialises a plist dict back to tab-indented XML form.
func MarshalPlist(m map[string]any) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := plist.NewEncoderForFormat(buf, plist.XMLFormat)
	enc.Indent("\t")
	if err := enc.Encode(m); err != nil {
		return nil, fmt.Errorf("encoding plist: %w", err)
	}
	return buf.Bytes(), nil
}

// CanonicalisePlistXML parses a plist and re-emits it as tab-indented XML so two
// documents that differ only in formatting (whitespace, indentation, line
// endings, trailing newline) render identically. Falls back to returning the
// input unchanged when it fails to parse — the caller still has a usable string.
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

// Equal is a structural-equality compare for parsed plist trees: dicts require
// matching keysets, arrays compare positionally, and the int64/uint64/int trio
// howett.net/plist emits compares numerically. Use this when every key is
// author-controlled and there are no server-injected keys to tolerate (the
// common case for opaque user-authored plist content). For mobileconfig
// payloads — where Jamf Pro injects/defaults keys — use the intersection-aware
// comparator in payloadhelpers instead.
func Equal(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, va := range av {
			vb, exists := bv[k]
			if !exists || !Equal(va, vb) {
				return false
			}
		}
		return true
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !Equal(av[i], bv[i]) {
				return false
			}
		}
		return true
	case uint64:
		return NumericEqual(int64(av), b)
	case int64:
		return NumericEqual(av, b)
	case int:
		return NumericEqual(int64(av), b)
	default:
		return a == b
	}
}

// NumericEqual reports whether int64 a equals b across the int64/uint64/int
// representations howett.net/plist may emit for the same plist <integer>.
func NumericEqual(a int64, b any) bool {
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

// SemanticallyEqual reports whether two plist documents are structurally equal
// once parsed — erasing all formatting differences (whitespace, indentation,
// line endings, trailing newline, dict key order). The bool ok is false when
// either input fails to parse as a plist, signalling the caller to fall back to
// a byte/string comparison for non-plist content.
func SemanticallyEqual(a, b []byte) (equal bool, ok bool) {
	pa, _, err := ParsePlist(a)
	if err != nil {
		return false, false
	}
	pb, _, err := ParsePlist(b)
	if err != nil {
		return false, false
	}
	return Equal(pa, pb), true
}
