// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package accrequire

import (
	"slices"
	"testing"
)

// The require gate is the mechanism this package exists for: an unset credential
// is a skip locally and a FAILURE in a pipeline that wired the secret, because a
// skip there is invisible — the package prints `ok` and the check goes green
// having asserted nothing. That makes an untested gate a contradiction: the one
// thing worse than the bug it prevents is the same bug hiding inside the
// prevention.
//
// These tests are untagged on purpose, so `make test` runs them with no
// credentials and no estate. They exist because three mutations were shown to
// survive the entire suite before they were written: dropping the whitespace and
// case normalisation in parseRequiredSets, replacing promotionToken's decision
// with a constant false, and inverting AccEnv's legacy fallback (see
// env_test.go). Every case below fails under at least one of those, which is the
// property to preserve when editing this file — a test here that no mutation
// breaks is decoration.

func TestParseRequiredSetsNormalisesTokens(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		raw  string
		want []string
	}{
		"single token": {
			raw:  "pro-tenant",
			want: []string{"pro-tenant"},
		},
		// The shape a hand-written lane row invites. A comma-separated list is
		// normally typed with a space after the comma, and the space must not
		// become part of the token.
		"space after the comma": {
			raw:  "securitycloud, pro-tenant",
			want: []string{"pro-tenant", "securitycloud"},
		},
		"surrounding whitespace": {
			raw:  "  platform  ",
			want: []string{"platform"},
		},
		"mixed case": {
			raw:  "Pro-Tenant",
			want: []string{"pro-tenant"},
		},
		// A trailing comma must not register the empty token, which would make
		// required[""] true and promote a precheck whose token list is nil.
		"trailing comma": {
			raw:  "platform,",
			want: []string{"platform"},
		},
		"only separators": {
			raw:  " , , ",
			want: nil,
		},
		// The local-contributor contract: nothing declared, nothing required, so
		// every gate stays a skip.
		"unset": {
			raw:  "",
			want: nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := parseRequiredSets(tc.raw)
			if len(got) != len(tc.want) {
				t.Fatalf("parseRequiredSets(%q) produced %d token(s) %v, want %d %v", tc.raw, len(got), setKeys(got), len(tc.want), tc.want)
			}
			for _, token := range tc.want {
				if !got[token] {
					t.Errorf("parseRequiredSets(%q) did not register %q; it registered %v", tc.raw, token, setKeys(got))
				}
			}
		})
	}
}

func TestPromotionTokenMatchesExactlyAndPromotesOnlyDeclaredSets(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		precheck    string
		required    string
		wantToken   string
		wantPromote bool
		wantKnown   bool
	}{
		// Nothing declared: every precheck skips, which is what lets a
		// contributor with no estate run `make testacc`.
		"nothing required skips": {
			precheck:  "AccPreCheck",
			required:  "",
			wantKnown: true,
		},
		"own token promotes": {
			precheck:    "AccPreCheckSecurityCloud",
			required:    "securitycloud",
			wantToken:   "securitycloud",
			wantPromote: true,
			wantKnown:   true,
		},
		// The whole point of one token per lane: the pro lane must not fail for
		// an organization secret it never uses.
		"another lane's token does not promote": {
			precheck:  "AccPreCheckAccount",
			required:  "platform",
			wantKnown: true,
		},
		// Exact match, both directions. A substring or prefix match would have
		// the pro lane demand the pro-tenant lane's credentials and vice versa.
		"pro does not promote pro-tenant": {
			precheck:  "AccTenantIDOrSkip",
			required:  "pro",
			wantKnown: true,
		},
		"pro-tenant does not promote a bare pro": {
			precheck:    "AccPreCheck",
			required:    "pro",
			wantKnown:   true,
			wantPromote: false,
		},
		// AccPreCheck is the generic gate every family but Jamf Account routes
		// through, so each scoped lane's token must reach it.
		"generic gate promotes for every scoped lane": {
			precheck:    "AccPreCheck",
			required:    "pro-tenant",
			wantToken:   "pro-tenant",
			wantPromote: true,
			wantKnown:   true,
		},
		// The offline functions need no credential, so no lane may fail them.
		"offline precheck never promotes": {
			precheck:  "AccPreCheckOffline",
			required:  "platform,organization,securitycloud,aigovernance,environment,pro-tenant",
			wantKnown: true,
		},
		// A precheck missing from the map has lost its gate. Reported as
		// unknown so SkipOrFailUnset can fail loudly rather than degrade to a
		// skip.
		"undeclared precheck is not known": {
			precheck:  "AccPreCheckSomethingNew",
			required:  "platform",
			wantKnown: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			token, promote, known := promotionToken(tc.precheck, parseRequiredSets(tc.required))
			if known != tc.wantKnown {
				t.Fatalf("promotionToken(%q, %q) known = %t, want %t", tc.precheck, tc.required, known, tc.wantKnown)
			}
			if promote != tc.wantPromote {
				t.Errorf("promotionToken(%q, %q) promote = %t, want %t — a wrong answer here is either a lane failing for another lane's secret or a skip reporting success having asserted nothing", tc.precheck, tc.required, promote, tc.wantPromote)
			}
			if token != tc.wantToken {
				t.Errorf("promotionToken(%q, %q) token = %q, want %q", tc.precheck, tc.required, token, tc.wantToken)
			}
		})
	}
}

// TestEveryPrecheckTokenIsReachable pins the map's own consistency: a token
// nobody can supply is a gate that never promotes, and a duplicate is a sign a
// lane was copied rather than added.
//
// It deliberately does not compare against .github/acceptance-lanes.json —
// internal/conformance/acc_lanes_test.go owns that comparison, and duplicating
// it here would give two places to update and no extra coverage.
func TestEveryPrecheckTokenIsReachable(t *testing.T) {
	t.Parallel()

	for precheck, tokens := range accPrecheckRequireTokens {
		seen := map[string]bool{}
		for _, token := range tokens {
			if token == "" {
				t.Errorf("%s declares an empty require token, which parseRequiredSets can never produce, so that entry can never promote", precheck)
			}
			if !parseRequiredSets(token)[token] {
				t.Errorf("%s declares token %q, which parseRequiredSets does not round-trip — it is mis-cased or carries whitespace, so JAMFPLATFORM_ACC_REQUIRE can never name it", precheck, token)
			}
			if seen[token] {
				t.Errorf("%s declares token %q twice", precheck, token)
			}
			seen[token] = true
		}
	}
}

// setKeys renders a token set for a failure message, sorted so the message is
// stable.
func setKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
