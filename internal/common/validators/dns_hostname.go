// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package validators

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// maxDNSNameLength is the longest name the Jamf Security Cloud custom DNS
// endpoints accept, counted in characters. The search-domain endpoint states it
// outright ("size must be between 1 and 253"), which is a Java `@Size` on the
// string rather than a wire-format bound, so it counts characters and not bytes —
// a 52-character Cyrillic name of 92 bytes is accepted — and a trailing root dot
// counts towards it.
const maxDNSNameLength = 253

// maxDNSLabelLength is the longest single dot-separated label accepted, counted in
// characters for the same reason as maxDNSNameLength. 63 is the DNS limit and the
// endpoints enforce it: a 64-character label is refused, a 63-character one
// accepted.
const maxDNSLabelLength = 63

// dnsHostnameValidator checks that a string attribute holds a DNS host name of the
// form the Jamf Security Cloud custom DNS endpoints accept.
type dnsHostnameValidator struct{}

// DNSHostname returns a validator.String enforcing DNS host name form.
//
// Extracted at its second consumer per STYLE_GUIDE §Shared abstractions: the Jamf
// Security Cloud search domain and hostname mappings resources both need it.
//
// Why it exists: the endpoints answer every malformed name with one opaque
// `400 [INVALID_FIELD] … Invalid field value.` carrying a null field, so nothing in
// the response says which attribute was wrong or what form was wanted. Only the
// empty and over-long cases name the field, and only on the search domain.
//
// The accepted grammar was established by probing 20 inputs against both endpoints
// on 2026-08-29; every verdict matched, which is why one validator serves both:
//
//   - up to 253 characters overall, each label 1 to 63 — characters, not bytes
//   - a single trailing dot is allowed (the root label); no other empty label is
//   - labels hold letters, digits, hyphen, underscore and any non-ASCII rune —
//     `xn--bcher-kva.example.com` and `ünïcode.example.com` are both accepted, and
//     case is stored verbatim rather than normalised
//   - a label may not begin or end with a hyphen or an underscore
//   - the final label may not begin with a digit, which is what separates the
//     accepted `123.foo` from the refused `foo.123`, `foo.1abc` and `1.2.3.4` —
//     only the *final* label is held to it, so a leading numeric label is fine
//   - a single label is fine: `corp` is accepted
//
// That is Guava's `InternetDomainName.isValid` without the public-suffix check,
// matching a Java service. Stated so a future divergence is diagnosable rather than
// mysterious: if the endpoints start accepting or refusing something this rejects,
// the suspect is a Guava version or a switch away from it, not a typo here.
//
// Wildcards are refused by both endpoints, so they are refused here. A ZTNA app's
// hostnames are a different vocabulary that does take them — this validator is not
// the one for that field.
//
// Null and unknown values defer to the server, per STYLE_GUIDE §Config-time
// validators.
func DNSHostname() validator.String {
	return dnsHostnameValidator{}
}

// Description returns a plain-text description of the validator.
func (dnsHostnameValidator) Description(_ context.Context) string {
	return "must be a DNS host name of up to 253 characters, with labels of up to 63 characters " +
		"separated by dots, no wildcards, and a final label that does not begin with a digit"
}

// MarkdownDescription returns the markdown description of the validator.
func (v dnsHostnameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString implements validator.String.
func (v dnsHostnameValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if reason := dnsHostnameProblem(value); reason != "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid DNS host name",
			reason+" Got: "+value,
		)
	}
}

// dnsHostnameProblem returns a sentence describing the first thing wrong with name,
// or the empty string when the name is acceptable. Split out from ValidateString so
// the grammar can be unit-tested against the probed inputs without building a
// framework request for each one.
//
// Order matters within a label: the character check runs before the boundary check
// so a wildcard is reported as a wildcard. Checked the other way round, `*.foo.com`
// comes back as "must begin and end with a letter", which sends the reader looking
// for the wrong mistake — and a wildcard is the likely one here, because a ZTNA
// app's hostnames do accept them.
func dnsHostnameProblem(name string) string {
	if name == "" {
		return "A host name is required."
	}
	if utf8.RuneCountInString(name) > maxDNSNameLength {
		return "A host name may be at most 253 characters."
	}

	labels := strings.Split(strings.TrimSuffix(name, "."), ".")
	for _, label := range labels {
		if label == "" {
			return "A host name may not contain an empty label, and may not begin with a dot."
		}
		if utf8.RuneCountInString(label) > maxDNSLabelLength {
			return "Each label in a host name may be at most 63 characters."
		}
		for _, r := range label {
			if !isDNSLabelRune(r) {
				return "A host name may contain only letters, digits, hyphens, underscores and non-ASCII " +
					"characters, separated by dots. Wildcards are not accepted."
			}
		}
		if !isDNSLabelBoundary(firstRune(label)) || !isDNSLabelBoundary(lastRune(label)) {
			return "Each label in a host name must begin and end with a letter, a digit or a non-ASCII character."
		}
	}
	if final := labels[len(labels)-1]; isASCIIDigit(firstRune(final)) {
		return "The final label in a host name may not begin with a digit, which also rules out a bare " +
			"IP address."
	}
	return ""
}

// isDNSLabelRune reports whether r may appear anywhere in a label. Every non-ASCII
// rune passes: the endpoints accept them without converting to punycode, and
// second-guessing which ones a Java IDN library would take would reject names that
// work.
func isDNSLabelRune(r rune) bool {
	return isDNSLabelBoundary(r) || r == '-' || r == '_'
}

// isDNSLabelBoundary reports whether r may open or close a label — everything a
// label may contain except the hyphen and the underscore.
func isDNSLabelBoundary(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	case r > 127:
		return true
	default:
		return false
	}
}

// firstRune returns the opening rune of a non-empty string. Indexing the string
// directly would hand a boundary check the first *byte* of a multi-byte rune, which
// happens to exceed 127 and so happens to pass — a coincidence, not a check.
func firstRune(s string) rune {
	r, _ := utf8.DecodeRuneInString(s)
	return r
}

// lastRune returns the final rune of a non-empty string.
func lastRune(s string) rune {
	runes := []rune(s)
	return runes[len(runes)-1]
}

// isASCIIDigit reports whether r is an ASCII digit. Deliberately ASCII-only, for
// the same reason isDNSLabelRune waves every non-ASCII rune through: probing
// covered ASCII digits, so restricting a non-ASCII decimal digit here would be a
// guess at a Java library's `CharMatcher.digit()` that could refuse a name the
// endpoints accept.
func isASCIIDigit(r rune) bool {
	return r >= '0' && r <= '9'
}
