// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package validators

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// label63 and label64 sit either side of the per-label bound the endpoints enforce.
// The cyrillic fixtures are two bytes per rune, so each one is over the byte bound
// while inside the character bound the endpoints actually apply: cyrillic40 is a
// 40-character label of 80 bytes, and name207 is a 207-character name of 407 bytes.
var (
	label63    = strings.Repeat("a", 63)
	label64    = strings.Repeat("a", 64)
	name253    = strings.Repeat(label63+".", 3) + strings.Repeat("a", 61)
	name254    = strings.Repeat("a", 254)
	cyrillic40 = strings.Repeat("б", 40)
	cyrillic64 = strings.Repeat("б", 64)
	name207    = strings.Repeat(strings.Repeat("б", 60)+".", 3) + strings.Repeat("б", 20) + ".com"
)

// TestDNSHostname pins the grammar to the inputs probed against both the search
// domain and the hostname mappings endpoints on 2026-08-29. Each case names the
// status the wire returned, so a future change here has to argue with the server
// rather than with a guess: `accepted` cases were answered 204, `refused` cases 400.
// A second round on 2026-08-29 added the digit-leading final labels (400 from both
// endpoints) and the multi-byte length cases (204 from both), which together
// established that the final-label rule is "begins with a digit" and that the length
// bounds count characters. The 64-rune multi-byte label is the one case here the wire
// did not answer: it follows from the character bound the other two proved.
func TestDNSHostname(t *testing.T) {
	cases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"multi label", "corp.example.com", false},
		{"single label", "corp", false},
		{"trailing root dot", "example.com.", false},
		{"interior underscore", "under_score.example.com", false},
		{"interior hyphen", "has-hyphen.example.com", false},
		{"punycode", "xn--bcher-kva.example.com", false},
		{"non-ascii", "ünïcode.example.com", false},
		{"uppercase preserved", "CORP.Example.COM", false},
		{"numeric first label", "123.foo", false},
		{"63-character label", label63 + ".example.com", false},
		{"253 characters", name253, false},
		{"40-rune multi-byte label", cyrillic40 + ".example.com", false},
		{"207 runes over the byte bound", name207, false},

		{"empty", "", true},
		{"wildcard", "*.example.com", true},
		{"leading dot", ".example.com", true},
		{"empty interior label", "a..b.com", true},
		{"leading hyphen", "-lead.example.com", true},
		{"trailing hyphen", "trail-.example.com", true},
		{"leading underscore", "_underscore-lead.com", true},
		{"trailing underscore", "trail_.com", true},
		{"numeric final label", "foo.123", true},
		{"digit-leading final label", "foo.1abc", true},
		{"digit-leading short final label", "foo.9x", true},
		{"dotted quad", "1.2.3.4", true},
		{"five numeric labels", "1.2.3.4.5", true},
		{"out-of-range quad", "999.999.999.999", true},
		{"spaces", "not a domain!!", true},
		{"64-character label", label64 + ".example.com", true},
		{"64-rune multi-byte label", cyrillic64 + ".example.com", true},
		{"254 characters", name254, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			DNSHostname().ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("hostname"),
				ConfigValue: types.StringValue(tc.value),
			}, resp)
			if got := resp.Diagnostics.HasError(); got != tc.wantErr {
				t.Fatalf("HasError() = %t, want %t (diags: %v)", got, tc.wantErr, resp.Diagnostics)
			}
		})
	}
}

// TestDNSHostnameDefersOnNullAndUnknown pins STYLE_GUIDE §Config-time validators:
// a value the config has not supplied yet is the server's business, not this
// validator's.
func TestDNSHostnameDefersOnNullAndUnknown(t *testing.T) {
	for name, value := range map[string]types.String{
		"null":    types.StringNull(),
		"unknown": types.StringUnknown(),
	} {
		t.Run(name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			DNSHostname().ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("hostname"),
				ConfigValue: value,
			}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("expected no diagnostics, got %v", resp.Diagnostics)
			}
		})
	}
}

// TestDNSHostnameProblemNamesTheFault keeps the diagnostics distinguishable: the
// endpoints collapse every one of these into the same opaque message, so the whole
// value of validating here is lost if the provider does the same.
func TestDNSHostnameProblemNamesTheFault(t *testing.T) {
	cases := map[string]string{
		"*.example.com":          "Wildcards are not accepted",
		".example.com":           "empty label",
		"-lead.example.com":      "begin and end with",
		"foo.123":                "begin with a digit",
		"foo.1abc":               "begin with a digit",
		label64 + ".example.com": "63 characters",
		name254:                  "253 characters",
	}
	for value, want := range cases {
		t.Run(value, func(t *testing.T) {
			got := dnsHostnameProblem(value)
			if !strings.Contains(got, want) {
				t.Fatalf("dnsHostnameProblem(%q) = %q, want it to mention %q", value, got, want)
			}
		})
	}
}

// TestFirstAndLastRuneDecode pins the boundary helpers to whole runes rather than
// bytes. dnsHostnameProblem cannot tell the difference — byte-indexing a multi-byte
// rune yields a value above 127, which isDNSLabelBoundary accepts for the wrong
// reason — so the intent is only testable on the helpers directly.
func TestFirstAndLastRuneDecode(t *testing.T) {
	if got := firstRune("ünicode"); got != 'ü' {
		t.Errorf("firstRune(%q) = %q, want %q", "ünicode", got, 'ü')
	}
	if got := lastRune("unicodë"); got != 'ë' {
		t.Errorf("lastRune(%q) = %q, want %q", "unicodë", got, 'ë')
	}
	if got := firstRune("abc"); got != 'a' {
		t.Errorf("firstRune(%q) = %q, want %q", "abc", got, 'a')
	}
	if got := lastRune("abc"); got != 'c' {
		t.Errorf("lastRune(%q) = %q, want %q", "abc", got, 'c')
	}
}
