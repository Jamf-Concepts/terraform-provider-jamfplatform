// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package conformance

// Lane conformance for the acceptance suite.
//
// This file carries NO acceptance build tag on purpose: it needs no credentials
// and no network, and it must run in `make test` and in CI's normal test job.
// The whole point is to catch a misfiled test at PR time rather than on a live
// estate.
//
// # What a lane is, and why the check exists
//
// The acceptance matrix splits the suite by product space, and lane membership
// is decided by a PACKAGE PATH PREFIX in .github/acceptance-lanes.json. That is
// where this diverges from jamfplatform-go-sdk: the SDK's acceptance suite is
// one package, so it splits on a test-NAME regex and its credential is chosen by
// an acc*Client factory the test body calls. This repo is already split into
// per-family packages, and the credential is chosen by the WORKFLOW, per lane —
// a test only expresses which credential it needs by which precheck helper it
// calls. So the SDK's "name lane == factory lane" becomes, here, "package lane
// == precheck lane", and that is the load-bearing assertion below.
//
// It is only sound while the package layout actually tracks the credential a
// test needs. If a test lands in a package whose lane runs a credential its
// precheck cannot accept, CI runs it under the wrong integration and the 403s
// read as a broken endpoint rather than as a misfiled test.
//
// Four further gaps are closed here, because each of them is a way for a lane to
// skip GREEN — which is the failure this whole file exists to make impossible:
//
//   - The lane name and the require token are DIFFERENT vocabularies (lane
//     `account` uses require token `organization`, lane `pro` uses `platform`),
//     and the token is written twice: once in the JSON, once in
//     accPrecheckRequireTokens. Disagree on one character and accRequiredSets
//     misses the set, SkipOrFailUnset degrades from t.Fatalf to t.Skipf, and the
//     whole lane reports green having run nothing.
//     TestLaneRequireTokensMatchThePrecheckVocabulary compares the two.
//
//   - A lane whose prefixes match no package runs nothing and still reports
//     success, and a lane still marked `planned` that has started matching
//     packages runs a new product's tests under another product's credential.
//     TestAcceptanceLaneTableIsUsable covers both, plus the path-boundary
//     hazard that prefix matching introduces and a name regex did not have.
//
//   - A new product's precheck helper that no lane serves would leave its tests
//     looking credential-free. TestEveryPrecheckHelperOwnsALane discovers
//     helpers by naming shape rather than from an allow-list, so the omission is
//     a failure rather than a silence.
//
//   - A test can reach credentials WITHOUT calling any testhelpers.AccPreCheck*
//     helper — through a package-local precheck, a direct os.Getenv, or its own
//     jamfplatform.NewClient. Such a test is indistinguishable from a genuinely
//     credential-free one, so it is detected and must be allow-listed by name.
//     TestAcceptanceTestsGatingOutsideThePrecheckPath.

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// mixedPrecheckPackages names the packages whose tests legitimately call more
// than one precheck helper, so that one of them demands a lane the package does
// not select, and says why that resolves honestly.
//
// An entry is an assertion that the package's lane still satisfies the precheck
// its tests cannot run without, and that the extra precheck's own tests SKIP
// rather than fail when the lane does not satisfy it. That is a claim about two
// credential sets and one workflow step, which no amount of parsing can verify,
// so each one is spelled out for a reviewer rather than pattern-matched.
//
// Prefer splitting the package, or moving it to the lane that satisfies every
// precheck it calls. An entry here costs a reviewer reading the lane's
// credentials by hand.
var mixedPrecheckPackages = map[string]string{
	// Calls AccPreCheck for the blueprint itself and AccPreCheckAIGovernance for
	// the handful of tests covering the com.jamf.ai-governance component. It
	// resolves in the platform-env lane because that lane sets
	// JAMFPLATFORM_ENVIRONMENT_ID, which is what AccPreCheck requires; the
	// AI-Governance component tests then skip honestly unless the AI Governance
	// declaration is also present on the step, which is a visible skip in the
	// lane summary rather than a failure. Splitting the package is the
	// alternative and is worse: the component tests build a whole blueprint and
	// would duplicate its fixture.
	"internal/resources/blueprints/blueprint": "AccPreCheck (satisfied by platform-env's JAMFPLATFORM_ENVIRONMENT_ID) plus AccPreCheckAIGovernance for the com.jamf.ai-governance component tests, which skip honestly without the AI Governance declaration",
}

// credentialGateAllowList names the acceptance tests that reach credentials
// without going through a testhelpers.AccPreCheck* helper, and why no helper can
// serve them. Keyed on the package-qualified test name, because a bare test name
// is not unique across 130 packages.
//
// Each entry must state the variables the test reads, because that is precisely
// what the precheck path would otherwise have proved: a test that gates on
// credentials itself is asserting, by hand, that its package's lane supplies
// what it reads. Nothing here can check that claim — the whole point is that the
// gate is outside the mechanism — so the reviewer checks it, and the entry is
// what they read.
//
// Before this check existed such a test was indistinguishable from a genuine
// credential-free stub: it calls no precheck, so the package/precheck comparison
// has nothing to compare and passes it silently.
var credentialGateAllowList = map[string]string{
	// Clears BOTH scope variables for the duration, so it cannot use
	// testhelpers.AccPreCheck — that helper skips unless one of them is set. Its
	// package-local accPreCheckCredentialsOnly reads JAMFPLATFORM_BASE_URL,
	// _CLIENT_ID and _CLIENT_SECRET, which every credential set provides, so the
	// pro-tenant lane its package selects supplies all three.
	"internal/provider.TestAccProviderScope_OrganizationScopeRejectedPerConstruct": "needs the credentials WITHOUT a scope, which AccPreCheck cannot express; its package-local accPreCheckCredentialsOnly reads JAMFPLATFORM_BASE_URL, JAMFPLATFORM_CLIENT_ID and JAMFPLATFORM_CLIENT_SECRET",

	// The three impact dependency-index tests share one liveCache helper, which
	// builds the client with jamfplatform.NewClient directly rather than through
	// testhelpers.NewAcceptanceClient: they exercise NewTenantCache and the real
	// policy sweep, so they need WithMinRequestInterval(0) — the provider's own
	// default and the setting the sweep's concurrency bound was chosen against —
	// which the suite-wide client does not carry. They read
	// JAMFPLATFORM_BASE_URL, _CLIENT_ID, _CLIENT_SECRET and one of
	// _ENVIRONMENT_ID / _TENANT_ID, all supplied by the pro lane their
	// unclaimed package selects.
	"internal/common/impact.TestAcceptance_DependencyIndex_SweepsLiveTenant":          "builds its own client for WithMinRequestInterval(0) via liveCache; reads JAMFPLATFORM_BASE_URL, JAMFPLATFORM_CLIENT_ID, JAMFPLATFORM_CLIENT_SECRET and one of JAMFPLATFORM_ENVIRONMENT_ID / JAMFPLATFORM_TENANT_ID",
	"internal/common/impact.TestAcceptance_DependencyIndex_PackagesAreFound":          "builds its own client for WithMinRequestInterval(0) via liveCache; reads JAMFPLATFORM_BASE_URL, JAMFPLATFORM_CLIENT_ID, JAMFPLATFORM_CLIENT_SECRET and one of JAMFPLATFORM_ENVIRONMENT_ID / JAMFPLATFORM_TENANT_ID",
	"internal/common/impact.TestAcceptance_DependencyReport_RendersForRealDependency": "builds its own client for WithMinRequestInterval(0) via liveCache; reads JAMFPLATFORM_BASE_URL, JAMFPLATFORM_CLIENT_ID, JAMFPLATFORM_CLIENT_SECRET and one of JAMFPLATFORM_ENVIRONMENT_ID / JAMFPLATFORM_TENANT_ID",
}

// TestAcceptanceTestPackagesAgreeWithTheirPrecheck is the load-bearing
// assertion: the lane a test's PACKAGE puts it in must be a lane whose
// credential the test's PRECHECK accepts.
//
// The demanded lanes are derived from accPrecheckRequireTokens rather than from
// a table here, because that map is the single place the lane vocabulary and the
// precheck vocabulary meet — a second copy in this file would agree with itself
// and prove nothing. A precheck with no tokens (AccPreCheckOffline) may run in
// any lane, which is the honest reading: the provider-defined functions run
// offline with no API client and no provider configuration, so no credential can
// be wrong for them.
//
// The check is deliberately one-directional in the cases where the vocabulary is
// one-to-many. AccPreCheck names every scoped lane's token, so a Pro test
// misfiled into internal/resources/security_cloud/ is NOT caught here — the
// securitycloud lane does satisfy AccPreCheck. The reverse, a Security Cloud
// test landing in the pro lane, IS caught, and that is the direction that
// matters: it is the one where a test runs against an estate that cannot serve
// it.
//
// A stale mixedPrecheckPackages entry is reported as a finding too, and is as
// bad as a missing one: it silently exempts whatever later reuses the package
// path, which for a renamed family is not hypothetical.
func TestAcceptanceTestPackagesAgreeWithTheirPrecheck(t *testing.T) {
	table := loadLaneTable(t)
	tokens := parseRequireTokens(t)
	suite := scanAcceptanceSuite(t, credentialVariables(table))

	if len(suite.Tests) == 0 {
		t.Fatal("parsed no TestAcc* functions from the acceptance suite")
	}

	var findings []string
	offending := map[string]bool{}
	gated := 0

	for _, test := range suite.Tests {
		lane := table.laneForPackage(test.Package)
		if len(test.Prechecks) > 0 {
			gated++
		}
		for _, precheck := range test.Prechecks {
			demanded, declared := tokens[precheck]
			if !declared {
				offending[test.Package] = true
				findings = append(findings, fmt.Sprintf(
					"%s.%s: calls testhelpers.%s, which accPrecheckRequireTokens does not declare, so JAMFPLATFORM_ACC_REQUIRE cannot promote its skips to failures and the lane would report success having run nothing. Add %q to accPrecheckRequireTokens in %s with the require token(s) of the lane(s) that can serve it",
					test.Package, test.Name, precheck, precheck, requireTokensFile))
				continue
			}
			if len(demanded) == 0 {
				continue
			}
			allowed := table.lanesRequiring(demanded)
			if slices.Contains(allowed, lane) {
				continue
			}
			offending[test.Package] = true
			findings = append(findings, fmt.Sprintf(
				"%s.%s: its package selects lane %q, but it calls testhelpers.%s, which requires one of the tokens [%s] and so belongs in lane(s) [%s]. Move the package under one of those lanes' `packages` prefixes in %s, or change the precheck to the one this lane's credential can serve",
				test.Package, test.Name, lane, precheck,
				strings.Join(demanded, " "), strings.Join(allowed, " "), laneTablePath))
		}
	}

	var kept []string
	for _, finding := range findings {
		exempt := false
		for pkg := range mixedPrecheckPackages {
			if strings.HasPrefix(finding, pkg+".") {
				exempt = true
				break
			}
		}
		if !exempt {
			kept = append(kept, finding)
		}
	}

	for pkg, why := range mixedPrecheckPackages {
		switch {
		case !slices.Contains(suite.Packages, pkg):
			kept = append(kept, fmt.Sprintf(
				"%s: in mixedPrecheckPackages (%s) but no such package has acceptance tests — drop the entry, or it will exempt whatever reuses the path",
				pkg, why))
		case !offending[pkg]:
			kept = append(kept, fmt.Sprintf(
				"%s: in mixedPrecheckPackages (%s) but its prechecks now all agree with its lane — drop the entry, or it will exempt the next disagreement",
				pkg, why))
		}
	}

	slices.Sort(kept)
	for _, finding := range kept {
		t.Error(finding)
	}
	if len(kept) > 0 {
		return
	}
	t.Logf("%d acceptance tests across %d packages, %d gated by a precheck helper, %d package(s) allow-listed for mixed prechecks; every package's lane agrees with its precheck",
		len(suite.Tests), len(suite.Packages), gated, len(mixedPrecheckPackages))
}

// TestLaneTableHelpersResolveSyntheticTables exercises lanesRequiring and
// laneForPackage directly, against tables written here rather than against the
// shipped one.
//
// It exists because the helpers were only reachable through the shipped table,
// and that made their coverage an accident of the table's contents. Neutering
// lanesRequiring to `true` does currently fail
// TestAcceptanceTestPackagesAgreeWithTheirPrecheck — but by way of the
// stale-allow-list rule, which notices that mixedPrecheckPackages' one entry has
// stopped disagreeing. Empty that allow-list — which is the state the comment on
// mixedPrecheckPackages actively recommends reaching, by splitting the blueprint
// package — and the neutered comparison passes while reporting "every package's
// lane agrees with its precheck". So the load-bearing assertion would become
// unguarded by the very cleanup the file asks for, which is the worst possible
// trigger. Synthetic tables make the helpers' behaviour independent of that.
func TestLaneTableHelpersResolveSyntheticTables(t *testing.T) {
	t.Parallel()

	table := laneTable{
		Lanes: []laneDef{
			{Lane: "alpha", Packages: []string{"internal/resources/alpha/"}, Require: "alpha-token"},
			{Lane: "beta", Packages: []string{"internal/resources/beta/"}, Require: "shared-token"},
			{Lane: "gamma", Packages: []string{"internal/resources/beta/narrower"}, Require: "shared-token"},
		},
		DefaultLane: laneDef{Lane: "default", Require: "default-token"},
	}

	t.Run("lanesRequiring", func(t *testing.T) {
		t.Parallel()

		for name, tc := range map[string]struct {
			tokens []string
			want   []string
		}{
			// The case a neutered comparison gets wrong: a token no lane
			// requires must resolve to NO lane, not to every lane.
			"unclaimed token":             {tokens: []string{"nobody"}, want: nil},
			"one lane":                    {tokens: []string{"alpha-token"}, want: []string{"alpha"}},
			"two lanes share a token":     {tokens: []string{"shared-token"}, want: []string{"beta", "gamma"}},
			"default lane is included":    {tokens: []string{"default-token"}, want: []string{"default"}},
			"several tokens, table order": {tokens: []string{"default-token", "alpha-token"}, want: []string{"alpha", "default"}},
			"no tokens at all":            {tokens: nil, want: nil},
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				if got := table.lanesRequiring(tc.tokens); !slices.Equal(got, tc.want) {
					t.Errorf("lanesRequiring(%v) = %v, want %v", tc.tokens, got, tc.want)
				}
			})
		}
	})

	t.Run("laneForPackage", func(t *testing.T) {
		t.Parallel()

		for pkg, want := range map[string]string{
			"internal/resources/alpha":         "alpha",
			"internal/resources/alpha/thing":   "alpha",
			"internal/resources/beta/narrower": "beta", // first match wins
			"internal/resources/beta/other":    "beta",
			"internal/resources/alphabet":      "default", // path boundary
			"internal/resources/pro/category":  "default",
		} {
			t.Run(pkg, func(t *testing.T) {
				t.Parallel()

				if got := table.laneForPackage(pkg); got != want {
					t.Errorf("laneForPackage(%q) = %q, want %q", pkg, got, want)
				}
			})
		}
	})
}

// TestLaneRequireTokensMatchThePrecheckVocabulary pins the two vocabularies
// together.
//
// A lane's `require` token is written twice and the two spellings are not the
// same word: .github/acceptance-lanes.json says `require: "organization"` for
// lane `account`, and accPrecheckRequireTokens says
// "AccPreCheckAccount": {"organization"}. Nothing but this test compares them,
// and a one-character disagreement is silent AND green: accRequiredSets misses
// the set, SkipOrFailUnset takes t.Skipf instead of t.Fatalf, and the lane
// reports success having run nothing — the exact skip-into-green failure the
// require mechanism exists to prevent.
//
// PLANNED lanes are exempt from the forward direction, and that exemption is the
// table's own design rather than a loophole: a planned lane reserves a name and
// its package prefixes before the product exists, and the lane table says in so
// many words that its precheck helper and require token are wired when the
// product lands. Requiring a precheck for `protect`, `school` and `android`
// today would fail on the shipped table and teach the reader to ignore this
// test. The moment a planned lane matches a package,
// TestAcceptanceLaneTableIsUsable fails and names the wiring to finish,
// including this token.
//
// A lane with an EMPTY require token is skipped rather than reported here, for
// the same reason: TestAcceptanceLaneTableIsUsable reports it with the
// consequence spelled out, and repeating it would double the output for one
// fault.
func TestLaneRequireTokensMatchThePrecheckVocabulary(t *testing.T) {
	table := loadLaneTable(t)
	tokens := parseRequireTokens(t)

	used := map[string][]string{}
	for precheck, list := range tokens {
		for _, token := range list {
			used[token] = append(used[token], precheck)
		}
	}

	var findings []string

	for _, lane := range table.allLanes() {
		if lane.Require == "" {
			continue
		}
		if lane.Planned {
			continue
		}
		if len(used[lane.Require]) == 0 {
			findings = append(findings, fmt.Sprintf(
				"lane %q declares require token %q, which no precheck helper claims in accPrecheckRequireTokens. JAMFPLATFORM_ACC_REQUIRE=%s would therefore promote nothing, and a missing credential in that lane would skip green. Add %q to the entry for the precheck the lane's tests call, in %s",
				lane.Lane, lane.Require, lane.Require, lane.Require, requireTokensFile))
		}
	}

	declared := map[string]bool{}
	for _, lane := range table.allLanes() {
		declared[lane.Require] = true
	}
	for token, prechecks := range used {
		if declared[token] {
			continue
		}
		slices.Sort(prechecks)
		findings = append(findings, fmt.Sprintf(
			"require token %q is claimed by precheck helper(s) [%s] but no lane in %s declares it, so no lane ever sets JAMFPLATFORM_ACC_REQUIRE=%s and those skips can never become failures. Either add the token to the lane that runs those tests, or correct the spelling in accPrecheckRequireTokens",
			token, strings.Join(prechecks, " "), laneTablePath, token))
	}

	slices.Sort(findings)
	for _, finding := range findings {
		t.Error(finding)
	}
	if len(findings) > 0 {
		return
	}
	t.Logf("%d lanes and %d precheck helpers agree on %d require token(s)", len(table.allLanes()), len(tokens), len(declared))
}

// TestAcceptanceLaneTableIsUsable guards the table itself.
//
// Every assertion here is a way the table can be wrong while every workflow it
// feeds still reports success: a lane matching nothing runs nothing, two lanes
// claiming one package let lane order decide silently, a planned lane that has
// started matching runs a new product under another product's credential, and a
// prefix that matches across a path boundary swallows a sibling family whole.
//
// The boundary case is asserted against the prefixes the table actually ships
// rather than against a synthetic example: for every prefix, a sibling package
// whose name merely starts with the prefix's last segment must not be claimed.
func TestAcceptanceLaneTableIsUsable(t *testing.T) {
	table := loadLaneTable(t)
	suite := scanAcceptanceSuite(t, credentialVariables(table))
	legacyKeys := parseLegacyEnvKeys(t)
	consumerVars := parseProviderConsumerVars(t)

	var findings []string

	claimedBy := map[string][]string{}
	for _, lane := range table.Lanes {
		for _, pkg := range suite.Packages {
			if lane.claims(pkg) {
				claimedBy[pkg] = append(claimedBy[pkg], lane.Lane)
			}
		}
	}

	for _, lane := range table.Lanes {
		for _, prefix := range lane.Packages {
			var matched []string
			for _, pkg := range suite.Packages {
				if packageMatchesPrefix(pkg, prefix) {
					matched = append(matched, pkg)
				}
			}
			switch {
			case lane.Planned && len(matched) > 0:
				findings = append(findings, fmt.Sprintf(
					"lane %q is still marked \"planned\" but its prefix %q now matches %d package(s) with acceptance tests (%s): the product has arrived, so finish the wiring — add a testhelpers.AccPreCheck%s helper, give it the require token %q in accPrecheckRequireTokens, set this lane's `credentials` to a key in `credential_sets`, wire the JAMFPLATFORM_ACC_%s_* secrets in .github/workflows/acceptance.yml, then drop \"planned\" from %s",
					lane.Lane, prefix, len(matched), strings.Join(matched, " "),
					strings.ToUpper(lane.Lane[:1])+lane.Lane[1:], lane.Require,
					strings.ToUpper(lane.Require), laneTablePath))
			case !lane.Planned && len(matched) == 0:
				findings = append(findings, fmt.Sprintf(
					"lane %q declares prefix %q, which matches no package holding acceptance tests. A lane that runs nothing reports a passing check having asserted nothing — correct the prefix, or mark the lane \"planned\": true until its product lands",
					lane.Lane, prefix))
			}

			sibling := strings.TrimSuffix(prefix, "/") + "_conformance_probe"
			if packageMatchesPrefix(sibling, prefix) {
				findings = append(findings, fmt.Sprintf(
					"lane %q prefix %q matches the sibling package %q, so prefix matching is not happening at a path boundary and a future family whose name merely starts with this one would be swallowed into this lane. Fix packageMatchesPrefix",
					lane.Lane, prefix, sibling))
			}
		}
	}

	for _, lane := range table.allLanes() {
		if lane.Require == "" {
			findings = append(findings, fmt.Sprintf(
				"lane %q declares no require token, so JAMFPLATFORM_ACC_REQUIRE can never name it and a missing credential would skip the whole lane green",
				lane.Lane))
		}
		switch {
		case lane.Planned && lane.Credentials != "":
			findings = append(findings, fmt.Sprintf(
				"lane %q is marked \"planned\" but names credential set %q. Do not wire a credential set ahead of the product: the set would then be referenced by a lane that runs nothing, which is exactly the stale wiring this table exists to reject",
				lane.Lane, lane.Credentials))
		case !lane.Planned && lane.Credentials == "":
			findings = append(findings, fmt.Sprintf(
				"lane %q names no credential set, so the workflow has nothing to authenticate it with. Set `credentials` to a key in `credential_sets`",
				lane.Lane))
		case !lane.Planned:
			if _, ok := table.CredentialSets[lane.Credentials]; !ok {
				findings = append(findings, fmt.Sprintf(
					"lane %q names credential set %q, which %s does not declare in `credential_sets` (it declares [%s])",
					lane.Lane, lane.Credentials, laneTablePath, strings.Join(setKeys(credentialSetNames(table)), " ")))
			}
		}

		for _, name := range lane.Declares {
			if slices.Contains(consumerVars, name) || slices.Contains(legacyKeys, name) {
				continue
			}
			findings = append(findings, fmt.Sprintf(
				"lane %q declares %q, which is neither a provider consumer variable (read by internal/provider) nor a key in accLegacyEnvNames in %s. Either it is a pre-rename name — use the post-rename key — or nothing reads it and the lane's declaration is inert",
				lane.Lane, name, legacyEnvFile))
		}
	}

	for name, set := range table.CredentialSets {
		if len(set.Provides) == 0 {
			findings = append(findings, fmt.Sprintf("credential set %q provides nothing, so a lane using it authenticates with no variables", name))
		}
		for _, provided := range set.Provides {
			if slices.Contains(consumerVars, provided) {
				continue
			}
			findings = append(findings, fmt.Sprintf(
				"credential set %q provides %q, which internal/provider never reads. A credential set may only provide the provider's own consumer variables — the aligned JAMFPLATFORM_ACC_%s_* secrets are mapped onto those names by .github/workflows/acceptance.yml",
				name, provided, set.SecretPrefix))
		}
	}

	for _, pkg := range setKeys(mapOfClaims(claimedBy)) {
		if lanes := claimedBy[pkg]; len(lanes) > 1 {
			findings = append(findings, fmt.Sprintf(
				"package %s is claimed by %d lanes (%s). First match wins, so lane order in %s is silently deciding which credential it runs under — narrow the prefixes so exactly one lane owns it",
				pkg, len(lanes), strings.Join(lanes, " "), laneTablePath))
		}
	}

	if len(table.DefaultLane.Packages) > 0 {
		findings = append(findings, fmt.Sprintf(
			"the default lane %q names explicit package prefixes (%s). It must stay \"everything unclaimed\", so that a new family cannot silently join it: an unclaimed package lands here and its precheck then disagrees with this lane's credential, which is a conformance failure rather than a green run",
			table.DefaultLane.Lane, strings.Join(table.DefaultLane.Packages, " ")))
	}
	if !table.DefaultLane.Lock {
		findings = append(findings, fmt.Sprintf(
			"the default lane %q must hold the shared-tenant lock: it is the lane that mutates the Jamf Pro tenant, and without lock:true the matrix would run it concurrently with itself",
			table.DefaultLane.Lane))
	}
	for _, lane := range table.Lanes {
		if lane.Lock {
			findings = append(findings, fmt.Sprintf(
				"lane %q holds lock:true, but only the default lane may: the lock exists to serialise the lane that mutates the shared Jamf Pro tenant, and a second locking lane queues behind it for a fixture it does not share",
				lane.Lane))
		}
	}

	slices.Sort(findings)
	for _, finding := range findings {
		t.Error(finding)
	}
	if len(findings) > 0 {
		return
	}
	planned := 0
	for _, lane := range table.Lanes {
		if lane.Planned {
			planned++
		}
	}
	t.Logf("%d lanes (%d planned) and %d credential sets partition %d packages holding acceptance tests, with no overlap and no dead prefix",
		len(table.allLanes()), planned, len(table.CredentialSets), len(suite.Packages))
}

// TestEveryPrecheckHelperOwnsALane closes the gap that adding a product opens.
//
// accPrecheckRequireTokens is hand-written, so a new product is two edits: a
// precheck helper in internal/testhelpers and an entry there. Miss the second
// and SkipOrFailUnset takes its unknown-precheck path — which does fail, but
// only once the helper is actually reached on a run with credentials, so a lane
// whose secret was never wired would report green having never called it. Worse,
// TestAcceptanceTestPackagesAgreeWithTheirPrecheck would have no tokens to
// compare, so the new product's package could sit in any lane at all.
//
// Helpers are discovered by naming SHAPE rather than from a list here, which is
// what makes the omission visible: an AccPreCheckProtect appearing in
// internal/testhelpers fails this test until it has an entry and a lane.
//
// The reverse direction matters as much. A stale key is not merely untidy:
// SkipOrFailUnset looks its precheck argument up in that map by string, so a key
// whose function has been renamed away leaves the new name unmapped while the
// map still looks populated.
func TestEveryPrecheckHelperOwnsALane(t *testing.T) {
	table := loadLaneTable(t)
	tokens := parseRequireTokens(t)
	prechecks, allFuncs := parsePrecheckHelpers(t)

	declared := map[string]bool{}
	for _, lane := range table.allLanes() {
		declared[lane.Require] = true
	}

	var findings []string

	for _, precheck := range prechecks {
		list, ok := tokens[precheck]
		if !ok {
			findings = append(findings, fmt.Sprintf(
				"testhelpers.%s is a precheck helper but accPrecheckRequireTokens does not declare it, so JAMFPLATFORM_ACC_REQUIRE cannot promote its skips to failures. Add it in %s with the require token(s) of the lane(s) whose credential it accepts, and give its product a lane in %s",
				precheck, requireTokensFile, laneTablePath))
			continue
		}
		for _, token := range list {
			if !declared[token] {
				findings = append(findings, fmt.Sprintf(
					"testhelpers.%s claims require token %q, which no lane in %s declares — the token can never be set, so the helper's skips can never become failures",
					precheck, token, laneTablePath))
			}
		}
	}

	for precheck := range tokens {
		if !slices.Contains(allFuncs, precheck) {
			findings = append(findings, fmt.Sprintf(
				"accPrecheckRequireTokens declares %q, but %s has no such function. Drop the stale key, or correct it to the helper's current name — SkipOrFailUnset resolves this map by string, so a stale key leaves the real helper unmapped",
				precheck, testhelpersDir))
		}
	}

	slices.Sort(findings)
	for _, finding := range findings {
		t.Error(finding)
	}
	if len(findings) > 0 {
		return
	}
	t.Logf("%d precheck-shaped helpers, %d accPrecheckRequireTokens entries, every one mapped to a declared lane", len(prechecks), len(tokens))
}

// TestAcceptanceTestsGatingOutsideThePrecheckPath finds the tests that reach
// credentials without a testhelpers.AccPreCheck* helper in their call graph.
//
// This is the provider's equivalent of the SDK's directClientAllowList, and it
// exists because such a test looks EXACTLY like a genuinely credential-free
// stub: it calls no precheck, so the package/precheck comparison has nothing to
// compare and passes it in silence. Three shapes reach credentials outside the
// mechanism — a package-local precheck, a direct os.Getenv (or testhelpers.AccEnv)
// of a credential variable, and a jamfplatform.NewClient of the test's own — and
// all three are detected the same way: by what the call graph touches, not by
// what it is called.
//
// The credential variables are taken from the lane table's own
// credential_sets[].provides rather than restated, so a credential set that
// grows a variable extends this check with it.
//
// An allow-list entry states the variables the test reads, because that hand
// assertion is what replaces the precheck's own. Prefer routing a new test
// through a precheck helper; an entry here costs a reviewer checking the lane's
// credentials against the list by hand.
func TestAcceptanceTestsGatingOutsideThePrecheckPath(t *testing.T) {
	table := loadLaneTable(t)
	suite := scanAcceptanceSuite(t, credentialVariables(table))

	ungated := map[string][]string{}
	for _, test := range suite.Tests {
		if len(test.Prechecks) > 0 || len(test.Reads) == 0 {
			continue
		}
		ungated[test.Package+"."+test.Name] = test.Reads
	}

	var findings []string
	for _, qualified := range setKeys(mapOfClaims(ungated)) {
		if _, ok := credentialGateAllowList[qualified]; ok {
			continue
		}
		findings = append(findings, fmt.Sprintf(
			"%s: reaches credentials (%s) without calling any testhelpers.AccPreCheck* helper, so nothing checks its lane against the credential it reads, and JAMFPLATFORM_ACC_REQUIRE cannot promote its skip to a failure. Route it through a precheck helper, or add it to credentialGateAllowList with the variables it reads and why no helper can serve it",
			qualified, strings.Join(ungated[qualified], ", ")))
	}

	for qualified, why := range credentialGateAllowList {
		if _, ok := ungated[qualified]; !ok {
			findings = append(findings, fmt.Sprintf(
				"%s: in credentialGateAllowList (%s) but it no longer gates on credentials outside a precheck helper — either the test is gone or it now calls one. Drop the entry, or it will exempt whatever reuses the name",
				qualified, why))
		}
	}

	slices.Sort(findings)
	for _, finding := range findings {
		t.Error(finding)
	}
	if len(findings) > 0 {
		return
	}
	t.Logf("%d acceptance tests reach credentials outside a precheck helper, all %d allow-listed by name", len(ungated), len(credentialGateAllowList))
}

// credentialVariables returns every provider variable the table's credential
// sets supply. These are the names a gate outside the precheck path would have
// to read, so they are what the scan looks for.
func credentialVariables(table laneTable) []string {
	seen := map[string]bool{}
	for _, set := range table.CredentialSets {
		for _, name := range set.Provides {
			seen[name] = true
		}
	}
	return setKeys(seen)
}

// credentialSetNames returns the declared credential set keys as a set, for a
// diagnostic that has to list them.
func credentialSetNames(table laneTable) map[string]bool {
	out := map[string]bool{}
	for name := range table.CredentialSets {
		out[name] = true
	}
	return out
}

// mapOfClaims converts a keyed slice map into a set, so setKeys can give the
// findings a stable order.
func mapOfClaims(m map[string][]string) map[string]bool {
	out := make(map[string]bool, len(m))
	for k := range m {
		out[k] = true
	}
	return out
}
