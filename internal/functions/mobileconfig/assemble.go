// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package mobileconfig assembles a complete macOS configuration profile
// (.mobileconfig) plist from native HCL payload objects. It is the shared core
// behind the jamfplatform::mobileconfig and jamfplatform::mcx_forced_payload
// provider functions.
package mobileconfig

import (
	"crypto/sha1"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/plisthelpers"
)

// int64 range expressed as float64 bounds for the whole-number heuristic.
// maxInt64ExclusiveFloat is 2^63 (one past math.MaxInt64, which is not
// representable as float64); minInt64Float is -2^63 == math.MinInt64 exactly.
const (
	maxInt64ExclusiveFloat = float64(1 << 63)
	minInt64Float          = -float64(1 << 63)
)

// dnsNamespace is the RFC 4122 DNS namespace UUID, matching Terraform's
// uuidv5("dns", ...).
var dnsNamespace = [16]byte{
	0x6b, 0xa7, 0xb8, 0x10, 0x9d, 0xad, 0x11, 0xd1,
	0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8,
}

// Profile is the decoded input to Assemble: optional top-level metadata plus the
// list of payload dictionaries (each an arbitrary object carrying at least a
// PayloadType key, using Apple's real payload key names).
type Profile struct {
	DisplayName       string
	Identifier        string
	Organization      string
	Description       string
	Scope             string
	RemovalDisallowed *bool
	Payloads          []map[string]any
}

// Assemble builds a complete .mobileconfig plist document from the profile. It
// normalizes number types (whole ⇒ <integer>, fractional ⇒ <real>) and injects
// the per-payload and top-level identity keys the author omits (deterministic
// UUIDs seeded on the profile identity so edits update in place rather than
// recreate).
func Assemble(p Profile) ([]byte, error) {
	if len(p.Payloads) == 0 {
		return nil, fmt.Errorf("at least one payload is required")
	}

	// identifier seeds every deterministic UUID in the profile. It is required:
	// without it, distinct profiles that share a leading payload type would seed
	// from that type and collide on PayloadIdentifier, so a Mac receiving both
	// would silently keep only one. Making it mandatory removes that trap.
	seed := strings.TrimSpace(p.Identifier)
	if seed == "" {
		return nil, fmt.Errorf("identifier is required: it seeds the profile and payload identifiers, and omitting it risks identity collisions between distinct profiles")
	}

	payloads := make([]any, 0, len(p.Payloads))
	for i, raw := range p.Payloads {
		norm, ok := normalizeValue(raw).(map[string]any)
		if !ok {
			return nil, fmt.Errorf("payload %d must be an object", i)
		}

		ptype, ok := norm["PayloadType"].(string)
		if !ok || strings.TrimSpace(ptype) == "" {
			return nil, fmt.Errorf("payload %d must have a non-empty string PayloadType", i)
		}

		payloadUUID := uuidv5(dnsNamespace, seed+"|"+ptype+"|"+strconv.Itoa(i))
		setIfAbsent(norm, "PayloadUUID", payloadUUID)
		setIfAbsent(norm, "PayloadIdentifier", payloadUUID)
		setIfAbsent(norm, "PayloadVersion", 1)
		setIfAbsent(norm, "PayloadEnabled", true)

		payloads = append(payloads, norm)
	}

	profileUUID := uuidv5(dnsNamespace, seed)
	profileIdentifier := seed

	scope := p.Scope
	if scope == "" {
		scope = "System"
	}
	removalDisallowed := true
	if p.RemovalDisallowed != nil {
		removalDisallowed = *p.RemovalDisallowed
	}

	doc := map[string]any{
		"PayloadContent":           payloads,
		"PayloadEnabled":           true,
		"PayloadIdentifier":        profileIdentifier,
		"PayloadRemovalDisallowed": removalDisallowed,
		"PayloadScope":             scope,
		"PayloadType":              "Configuration",
		"PayloadUUID":              profileUUID,
		"PayloadVersion":           1,
	}
	if p.DisplayName != "" {
		doc["PayloadDisplayName"] = p.DisplayName
	}
	if p.Organization != "" {
		doc["PayloadOrganization"] = p.Organization
	}
	if p.Description != "" {
		doc["PayloadDescription"] = p.Description
	}

	return plisthelpers.MarshalPlist(doc)
}

func setIfAbsent(m map[string]any, key string, val any) {
	if _, ok := m[key]; !ok {
		m[key] = val
	}
}

// normalizeValue walks a decoded HCL value and coerces whole-number float64s to
// int64 so the plist encoder emits <integer> rather than <real>. Terraform
// numbers arrive as a single numeric type, so this is the convention that
// recovers plist's: whole ⇒ <integer>, fractional ⇒ <real>. Recurses into
// nested dicts and arrays.
func normalizeValue(v any) any {
	switch t := v.(type) {
	case float64:
		// ponytail: whole-float ⇒ integer heuristic, bounded to int64 range.
		// Bounds are the powers of two either side of the int64 range: as
		// float64, math.MaxInt64 rounds up to 2^63, so `t <= math.MaxInt64`
		// would admit t == 2^63, which overflows on int64(t). Compare against
		// 2^63 / -2^63 strictly instead. Values above 2^53 already lost
		// integer precision upstream in the big.Float→float64 hop; documented.
		if t == math.Trunc(t) && t >= minInt64Float && t < maxInt64ExclusiveFloat {
			return int64(t)
		}
		return t
	case map[string]any:
		m := make(map[string]any, len(t))
		for k, val := range t {
			m[k] = normalizeValue(val)
		}
		return m
	case []any:
		s := make([]any, len(t))
		for i, val := range t {
			s[i] = normalizeValue(val)
		}
		return s
	default:
		return v
	}
}

// uuidv5 computes an RFC 4122 version-5 (SHA-1, name-based) UUID. Stdlib only —
// the provider has no UUID dependency and this is all that is needed.
func uuidv5(namespace [16]byte, name string) string {
	h := sha1.New()
	h.Write(namespace[:])
	h.Write([]byte(name))
	sum := h.Sum(nil)

	var u [16]byte
	copy(u[:], sum[:16])
	u[6] = (u[6] & 0x0f) | 0x50 // version 5
	u[8] = (u[8] & 0x3f) | 0x80 // RFC 4122 variant

	return fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}
