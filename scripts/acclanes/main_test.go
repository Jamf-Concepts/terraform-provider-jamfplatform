//go:build acclanes

// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// realTable is the table CI actually uses, read relative to this package
// directory. Tests read it rather than a fixture so a change to the shipped
// lanes is exercised here too: the lane table is the thing most likely to be
// edited without touching this tool.
const realTable = "../../" + defaultTablePath

// module is this repo's module path. partition takes import paths and strips it,
// so the tests feed the same full paths acctargets prints rather than the
// relative form, and thereby exercise the stripping.
const module = "github.com/Jamf-Concepts/terraform-provider-jamfplatform"

func mustLoadReal(t *testing.T) laneTable {
	t.Helper()
	table, err := loadTable(realTable)
	if err != nil {
		t.Fatalf("loading %s: %v", realTable, err)
	}
	return table
}

// imports turns module-relative package paths into the full import paths
// acctargets emits.
func imports(rel ...string) []string {
	out := make([]string, 0, len(rel))
	for _, r := range rel {
		out = append(out, module+"/"+r)
	}
	return out
}

// laneCounts indexes a matrix by lane name.
func laneCounts(entries []matrixEntry) map[string]int {
	out := map[string]int{}
	for _, e := range entries {
		out[e.Lane] = e.Count
	}
	return out
}

func TestPartitionSplitsByProductAndDefaultsToPro(t *testing.T) {
	table := mustLoadReal(t)
	entries, err := partition(table, module, imports(
		"internal/resources/account/sso_domain",
		"internal/actions/account/sso_domain",
		"internal/resources/security_cloud/dns_zone",
		"internal/resources/security_cloud/ztna_gateway",
		"internal/resources/ai_governance/policy",
		"internal/resources/blueprints/blueprint",
		"internal/resources/cbengine/benchmark",
		"internal/provider",
		"internal/resources/pro/tenant_id",
		"internal/resources/pro/script",
		"internal/resources/pro/category",
		"internal/functions/mobileconfig",
	))
	if err != nil {
		t.Fatalf("partition: %v", err)
	}

	got := laneCounts(entries)
	want := map[string]int{
		"account":       2,
		"securitycloud": 2,
		"aigovernance":  1,
		"platform-env":  2,
		"pro-tenant":    2,
		"pro":           3, // script, category and the offline function package
	}
	for lane, n := range want {
		if got[lane] != n {
			t.Errorf("lane %q: got %d packages, want %d (all lanes: %v)", lane, got[lane], n, got)
		}
	}
	if len(got) != len(want) {
		t.Errorf("got %d lanes %v, want exactly %d", len(got), got, len(want))
	}
}

// The default lane must be last so the workflow's summary reads product lanes
// first and the long serial Pro lane last, and because the default lane is the
// fall-through: emitting it before the lanes that could have claimed its
// packages would misrepresent the priority the table encodes.
func TestDefaultLaneIsEmittedLast(t *testing.T) {
	table := mustLoadReal(t)
	entries, err := partition(table, module, imports(
		"internal/resources/pro/script",
		"internal/resources/account/sso_domain",
	))
	if err != nil {
		t.Fatalf("partition: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d lanes, want 2: %+v", len(entries), entries)
	}
	if entries[len(entries)-1].Lane != table.DefaultLane.Lane {
		t.Errorf("last lane is %q, want the default lane %q", entries[len(entries)-1].Lane, table.DefaultLane.Lane)
	}
}

// The `go test` argument must be the ./internal/... form, space separated, so
// the command line in the job log is readable and copy-pasteable.
func TestPackagesAreEmittedAsGoTestArguments(t *testing.T) {
	table := mustLoadReal(t)
	entries, err := partition(table, module, imports(
		"internal/resources/pro/category",
		"internal/resources/pro/script",
	))
	if err != nil {
		t.Fatalf("partition: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d lanes, want 1: %+v", len(entries), entries)
	}
	want := "./internal/resources/pro/category ./internal/resources/pro/script"
	if entries[0].Packages != want {
		t.Errorf("packages = %q, want %q", entries[0].Packages, want)
	}
}

// A lane with nothing to run must not appear. A job that executes zero tests
// reports a passing check having asserted nothing, which is exactly the
// skip-into-green failure the require mechanism exists to prevent.
func TestPartitionDropsEmptyLanes(t *testing.T) {
	table := mustLoadReal(t)
	entries, err := partition(table, module, imports("internal/resources/security_cloud/dns_zone"))
	if err != nil {
		t.Fatalf("partition: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d lanes, want 1: %+v", len(entries), entries)
	}
	if entries[0].Lane != "securitycloud" {
		t.Errorf("got lane %q, want securitycloud", entries[0].Lane)
	}
	if entries[0].Lock {
		t.Error("the securitycloud lane must not hold the shared Pro tenant lock — that is the point of splitting it out")
	}
}

// The Pro lane is the only one that mutates the shared tenant, so it is the only
// one that may serialise. If this inverts, either every lane queues behind Pro
// again or Pro stops being serialised.
func TestOnlyTheDefaultLaneTakesTheTenantLock(t *testing.T) {
	table := mustLoadReal(t)
	for _, lane := range table.Lanes {
		if lane.Lock {
			t.Errorf("lane %q claims the tenant lock; only the default lane may", lane.Lane)
		}
	}
	if !table.DefaultLane.Lock {
		t.Error("default lane must hold the tenant lock")
	}
}

// Every lane needs its own require token, or a missing credential skips the lane
// green instead of failing it.
func TestEveryLaneDeclaresARequireToken(t *testing.T) {
	table := mustLoadReal(t)
	for _, lane := range allLanes(table) {
		if lane.Require == "" {
			t.Errorf("lane %q has no require token", lane.Lane)
		}
	}
}

// Require tokens must be distinct: the whole point is that a lane fails for its
// own missing credential and never for another lane's.
func TestRequireTokensAreUnique(t *testing.T) {
	table := mustLoadReal(t)
	seen := map[string]string{}
	for _, lane := range allLanes(table) {
		if prior, dup := seen[lane.Require]; dup {
			t.Errorf("lanes %q and %q share require token %q", prior, lane.Lane, lane.Require)
		}
		seen[lane.Require] = lane.Lane
	}
}

// An active lane must name a credential set that exists, and a planned lane must
// name none: wiring a credential set ahead of the product is the stale row the
// table's own note forbids.
func TestCredentialSetsMatchLaneStatus(t *testing.T) {
	table := mustLoadReal(t)
	for _, lane := range allLanes(table) {
		switch {
		case lane.Planned:
			if lane.Credentials != "" {
				t.Errorf("planned lane %q names credential set %q; do not wire a credential set before the product lands", lane.Lane, lane.Credentials)
			}
		case lane.Credentials == "":
			t.Errorf("active lane %q names no credential set", lane.Lane)
		default:
			set, known := table.CredentialSets[lane.Credentials]
			if !known {
				t.Errorf("lane %q names credential set %q, absent from credential_sets", lane.Lane, lane.Credentials)
				continue
			}
			if set.SecretPrefix == "" {
				t.Errorf("credential set %q has no secret_prefix", lane.Credentials)
			}
		}
	}
}

// The default lane must never declare explicit prefixes: it is "everything
// unclaimed", and giving it prefixes would let a new family silently join it
// while looking as though it had been placed deliberately.
func TestDefaultLaneDeclaresNoPrefixes(t *testing.T) {
	table := mustLoadReal(t)
	if len(table.DefaultLane.Packages) != 0 {
		t.Errorf("default lane %q declares prefixes %v; it must be everything unclaimed", table.DefaultLane.Lane, table.DefaultLane.Packages)
	}
}

// The workflow selects a lane's credentials by dynamic secret indexing —
// secrets[format('{0}_BASE_URL', matrix.secret_prefix)] — so every emitted entry
// must carry the resolved prefix.
func TestSecretPrefixIsResolvedOntoEveryEntry(t *testing.T) {
	table := mustLoadReal(t)
	entries, err := partition(table, module, imports(
		"internal/resources/account/sso_domain",
		"internal/resources/pro/tenant_id",
		"internal/resources/pro/script",
	))
	if err != nil {
		t.Fatalf("partition: %v", err)
	}
	want := map[string]string{
		"account":    "JAMFPLATFORM_ACC_ORGANIZATION",
		"pro-tenant": "JAMFPLATFORM_ACC_PRO_TENANT",
		"pro":        "JAMFPLATFORM_ACC_ENVIRONMENT",
	}
	for _, e := range entries {
		if e.SecretPrefix == "" {
			t.Errorf("lane %q emitted an empty secret_prefix: the workflow would index secrets['_BASE_URL'] and report a missing secret rather than a broken table", e.Lane)
		}
		if w, checked := want[e.Lane]; checked && e.SecretPrefix != w {
			t.Errorf("lane %q: secret_prefix = %q, want %q", e.Lane, e.SecretPrefix, w)
		}
	}
}

// An emitted lane with no credential set is a hard error rather than an empty
// prefix, because an empty prefix fails every test in the lane with
// "credentials not configured" — a missing-secret symptom for a table bug.
func TestEmittedLaneWithoutCredentialsIsAnError(t *testing.T) {
	table := laneTable{
		CredentialSets: map[string]credentialSet{"ENVIRONMENT": {SecretPrefix: "P"}},
		Lanes:          []laneDef{{Lane: "orphan", Packages: []string{"internal/resources/pro/"}, Require: "orphan"}},
		DefaultLane:    laneDef{Lane: "pro", Credentials: "ENVIRONMENT", Require: "platform", Lock: true},
	}
	_, err := partition(table, module, imports("internal/resources/pro/script"))
	if err == nil {
		t.Fatal("expected an error for an emitted lane naming no credential set")
	}
	for _, want := range []string{"orphan", "credential set"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q; got: %v", want, err)
		}
	}
}

// A credentials value that credential_sets does not declare is the other half of
// the same failure: it resolves to nothing, so it must be named at plan time.
func TestUnknownCredentialSetIsAnError(t *testing.T) {
	table := laneTable{
		CredentialSets: map[string]credentialSet{"ENVIRONMENT": {SecretPrefix: "P"}},
		Lanes:          []laneDef{{Lane: "typo", Packages: []string{"internal/resources/pro/"}, Credentials: "ENVIRONMNET", Require: "typo"}},
		DefaultLane:    laneDef{Lane: "pro", Credentials: "ENVIRONMENT", Require: "platform", Lock: true},
	}
	_, err := partition(table, module, imports("internal/resources/pro/script"))
	if err == nil {
		t.Fatal("expected an error for a credentials value absent from credential_sets")
	}
	for _, want := range []string{"typo", "ENVIRONMNET", "credential_sets"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q; got: %v", want, err)
		}
	}
}

// Entitlement declarations reach the matrix as a scalar so the workflow can
// surface them on the step; a lane with none must emit the empty string rather
// than null, which a matrix value may not be.
func TestDeclaresAreEmittedAsASpaceSeparatedScalar(t *testing.T) {
	table := mustLoadReal(t)
	entries, err := partition(table, module, imports(
		"internal/resources/ai_governance/policy",
		"internal/resources/blueprints/blueprint",
	))
	if err != nil {
		t.Fatalf("partition: %v", err)
	}
	for _, e := range entries {
		switch e.Lane {
		case "aigovernance":
			if e.Declares != "JAMFPLATFORM_ACC_AIGOVERNANCE_ENVIRONMENT_ID" {
				t.Errorf("aigovernance declares = %q", e.Declares)
			}
		case "platform-env":
			if e.Declares != "" {
				t.Errorf("platform-env declares = %q, want empty", e.Declares)
			}
		}
	}
}

// Prefix matching happens at a path BOUNDARY. This is the hazard that replaces
// the SDK's RE2 concern: a lane owning internal/resources/account/ must not
// claim a sibling package whose name merely starts with "account".
func TestPrefixMatchingHappensAtAPathBoundary(t *testing.T) {
	table := mustLoadReal(t)
	entries, err := partition(table, module, imports("internal/resources/account_thing"))
	if err != nil {
		t.Fatalf("partition: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d lanes, want 1: %+v", len(entries), entries)
	}
	if entries[0].Lane != table.DefaultLane.Lane {
		t.Errorf("internal/resources/account_thing landed in lane %q; a prefix must match at a path boundary, so it belongs to the default lane", entries[0].Lane)
	}

	for _, tc := range []struct {
		prefix, rel string
		want        bool
	}{
		{"internal/resources/account/", "internal/resources/account/sso_domain", true},
		{"internal/resources/account/", "internal/resources/account", true},
		{"internal/resources/account/", "internal/resources/account_thing", false},
		{"internal/resources/account/", "internal/resources/accounts/x", false},
		{"internal/resources/account", "internal/resources/account/sso_domain", true},
		{"internal/provider/", "internal/provider", true},
		{"internal/provider/", "internal/providerdata", false},
		{"", "internal/resources/pro/script", false},
	} {
		if got := claims(tc.prefix, tc.rel); got != tc.want {
			t.Errorf("claims(%q, %q) = %v, want %v", tc.prefix, tc.rel, got, tc.want)
		}
	}
}

// Two lanes claiming the same package resolve first-match-wins, which makes the
// table readable as a priority list. The builder does NOT reject the overlap:
// forbidding it is the conformance test's job, because it is a property of the
// table rather than of a particular scope, and a plan step that failed on an
// overlap no scope exercised would be reporting a table bug at random times.
func TestOverlappingLanesResolveFirstMatchWins(t *testing.T) {
	table := laneTable{
		CredentialSets: map[string]credentialSet{"A": {SecretPrefix: "PREFIX_A"}, "B": {SecretPrefix: "PREFIX_B"}},
		Lanes: []laneDef{
			{Lane: "first", Packages: []string{"internal/resources/pro/"}, Credentials: "A", Require: "first", TimeoutMinutes: 30},
			{Lane: "second", Packages: []string{"internal/resources/pro/script"}, Credentials: "B", Require: "second", TimeoutMinutes: 30},
		},
		DefaultLane: laneDef{Lane: "pro", Credentials: "A", Require: "platform", Lock: true, TimeoutMinutes: 30},
	}
	entries, err := partition(table, module, imports("internal/resources/pro/script"))
	if err != nil {
		t.Fatalf("partition: %v", err)
	}
	if len(entries) != 1 || entries[0].Lane != "first" {
		t.Fatalf("got %+v, want the single lane %q (first match in table order)", entries, "first")
	}
}

// A lane that claims packages must declare its own job ceiling. Emitting one
// without it would interpolate an empty `timeout-minutes:` — a workflow syntax
// error naming the expression rather than the lane — and a zero would be read as
// "no limit", restoring the 6h exposure the field exists to close. Same
// treatment as a missing secret_prefix, and for the same reason.
func TestLaneWithoutATimeoutFailsThePlanStep(t *testing.T) {
	for name, minutes := range map[string]int{"absent": 0, "negative": -1} {
		t.Run(name, func(t *testing.T) {
			table := laneTable{
				CredentialSets: map[string]credentialSet{"A": {SecretPrefix: "PREFIX_A"}},
				Lanes: []laneDef{
					{Lane: "claimant", Packages: []string{"internal/resources/pro/"}, Credentials: "A", Require: "claimant", TimeoutMinutes: minutes},
				},
				DefaultLane: laneDef{Lane: "pro", Credentials: "A", Require: "platform", Lock: true, TimeoutMinutes: 30},
			}
			_, err := partition(table, module, imports("internal/resources/pro/script"))
			if err == nil {
				t.Fatal("partition succeeded for a lane with no timeout_minutes; it must refuse rather than emit a job with no ceiling")
			}
			for _, want := range []string{"claimant", "timeout_minutes", "per test binary"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error does not mention %q, so it does not say which lane or why go test -timeout is not a substitute: %v", want, err)
				}
			}
		})
	}
}

// Every shipped lane carries a ceiling, planned lanes included: a planned lane
// that suddenly matches a package already fails the plan step, and it should not
// fail twice for two different reasons.
func TestShippedLanesAllDeclareATimeout(t *testing.T) {
	table := mustLoadReal(t)
	for _, lane := range append(append([]laneDef{}, table.Lanes...), table.DefaultLane) {
		if lane.TimeoutMinutes <= 0 {
			t.Errorf("lane %q declares timeout_minutes %d; give it a positive job ceiling", lane.Lane, lane.TimeoutMinutes)
		}
		// GitHub's own default is 6h. A lane at or above that has opted out of
		// the protection while appearing to have it.
		if lane.TimeoutMinutes >= 360 {
			t.Errorf("lane %q declares timeout_minutes %d, at or above GitHub's 6h default, so the ceiling is decorative", lane.Lane, lane.TimeoutMinutes)
		}
	}
}

// The shipped table must claim no package twice. This mirrors the conformance
// test's assertion and is cheap to keep here, because a table whose lanes
// overlap makes the emitted matrix depend on row order rather than on intent.
func TestShippedLanePrefixesDoNotOverlap(t *testing.T) {
	table := mustLoadReal(t)
	for i, a := range table.Lanes {
		for j, b := range table.Lanes {
			if i >= j {
				continue
			}
			for _, pa := range a.Packages {
				for _, pb := range b.Packages {
					if claims(pa, strings.Trim(pb, "/")) || claims(pb, strings.Trim(pa, "/")) {
						t.Errorf("lanes %q (%s) and %q (%s) overlap", a.Lane, pa, b.Lane, pb)
					}
				}
			}
		}
	}
}

// A planned lane reserves a name before its product exists, so it must match
// nothing and must not produce a job.
func TestPlannedLanesAreReservedAndEmitNoJob(t *testing.T) {
	table := mustLoadReal(t)

	var planned []string
	for _, lane := range table.Lanes {
		if lane.Planned {
			planned = append(planned, lane.Lane)
		}
	}
	if len(planned) == 0 {
		t.Skip("no planned lanes reserved")
	}

	entries, err := partition(table, module, imports(
		"internal/resources/pro/script",
		"internal/resources/account/sso_domain",
	))
	if err != nil {
		t.Fatalf("partition: %v", err)
	}
	for _, e := range entries {
		for _, name := range planned {
			if e.Lane == name {
				t.Errorf("planned lane %q emitted a matrix entry", name)
			}
		}
	}
}

// The moment a planned lane's product arrives, the plan step must refuse rather
// than silently route the new packages somewhere. This is the readiness
// guarantee: the first Protect package cannot run until its credential is wired.
func TestPlannedLaneThatMatchesPackagesFailsThePlanStep(t *testing.T) {
	table := mustLoadReal(t)

	var probe string
	for _, lane := range table.Lanes {
		if lane.Planned && lane.Lane == "protect" {
			probe = "internal/resources/protect/plan"
		}
	}
	if probe == "" {
		t.Skip("no planned protect lane to probe")
	}

	_, err := partition(table, module, imports("internal/resources/pro/script", probe))
	if err == nil {
		t.Fatal("expected the plan step to fail when a planned lane claims a package")
	}
	for _, want := range []string{"planned", "protect", probe, "precheck helper", "require token", "credential set", "secrets"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q so the reader knows what to wire; got: %v", want, err)
		}
	}
}

func TestResolveScopeTakesAnExplicitListAsGiven(t *testing.T) {
	got, err := resolveScope("  " + module + "/internal/resources/pro/script  " + module + "/internal/resources/pro/category ")
	if err != nil {
		t.Fatalf("resolveScope: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %v, want the two listed packages", got)
	}
}

// An empty scope is not a failure — acctargets prints it for a change that
// touches no package — and must yield an empty matrix, not a missing one.
func TestEmptyScopePrintsAnEmptyMatrix(t *testing.T) {
	out := captureStdout(t, func() {
		if err := run(realTable, "   "); err != nil {
			t.Fatalf("run: %v", err)
		}
	})
	if strings.TrimSpace(out) != "[]" {
		t.Errorf("got %q, want []", out)
	}
}

// ./... with nothing listed must fail loudly. Emitting an empty matrix would
// skip the entire acceptance suite and still report success.
//
// listPackages shells out to the real toolchain, so the empty answer is staged
// with a throwaway module that holds one package and no acceptance test rather
// than by stubbing `go list` — which would test the stub.
func TestFullScopeWithNoPackagesIsAnError(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	write("go.mod", "module example.test\n\ngo 1.26\n")
	write("lonely.go", "package lonely\n")
	write("lonely_test.go", "package lonely\n\nimport \"testing\"\n\nfunc TestUnit(t *testing.T) {}\n")
	inDir(t, dir)

	_, err := resolveScope(scopeAll)
	if err == nil {
		t.Fatal("expected an error when ./... enumerates no acceptance package")
	}
	if !strings.Contains(err.Error(), "acceptance") {
		t.Errorf("error should name the acceptance build tag as the thing it looked for; got: %v", err)
	}
}

// A package carrying only unit tests must not be enumerated as part of the
// acceptance suite: acctargets restricts its change-scoped output the same way,
// and admitting unit-test-only packages here would have the two paths disagree
// about what the suite is — and would serialise a pile of offline tests behind
// the tenant lock in the pro lane.
func TestAcceptanceCandidatesRequireTheBuildTag(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	write("unit_test.go", "package x\n")
	write("acc_test.go", "//go:build acceptance\n\npackage x\n")

	if got := acceptanceCandidates([]goListPackage{{ImportPath: "x", Dir: dir, TestGoFiles: []string{"unit_test.go"}}}); len(got) != 0 {
		t.Errorf("unit-test-only package was admitted: %v", got)
	}
	if got := acceptanceCandidates([]goListPackage{{ImportPath: "x", Dir: dir, TestGoFiles: []string{"unit_test.go", "acc_test.go"}}}); len(got) != 1 {
		t.Errorf("acceptance-tagged package was not admitted: %v", got)
	}
}

// The shipped table must exist at the path the workflow uses, relative to the
// repo root — a broken default is a plan step that fails only in CI.
func TestShippedTableResolvesFromTheDefaultFlagPath(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "..", filepath.FromSlash(defaultTablePath))); err != nil {
		t.Fatalf("shipped lane table not where the -table default expects it: %v", err)
	}
	if _, err := loadTable(realTable); err != nil {
		t.Fatalf("shipped lane table does not load: %v", err)
	}
}

// The table's top-level "_comment" array is documentation. The loader must
// ignore it rather than choke on it.
func TestLoaderIgnoresTheCommentArray(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lanes.json")
	body := `{"_comment":["prose","more prose"],
	  "credential_sets":{"E":{"secret_prefix":"P","provides":["X"],"evidence":"e","why":"w"}},
	  "lanes":[{"lane":"a","packages":["internal/x/"],"credentials":"E","require":"a","declares":[],"lock":false,"why":"w"}],
	  "default_lane":{"lane":"pro","packages":[],"credentials":"E","require":"platform","declares":[],"lock":true,"why":"w"}}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	table, err := loadTable(path)
	if err != nil {
		t.Fatalf("loadTable: %v", err)
	}
	if len(table.Lanes) != 1 || table.DefaultLane.Lane != "pro" {
		t.Errorf("unexpected table: %+v", table)
	}
}

func TestLoadTableRejectsATableWithNoLanes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lanes.json")
	if err := os.WriteFile(path, []byte(`{"lanes":[],"default_lane":{}}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := loadTable(path); err == nil {
		t.Fatal("expected an error for a table declaring no lanes")
	}
}

// A scope token outside the module is an error rather than a default-lane
// assignment: a package this tool cannot place is one it must not narrow.
func TestRelativeAcceptsBothFormsAndRejectsForeignPaths(t *testing.T) {
	for _, tc := range []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: module + "/internal/resources/pro/script", want: "internal/resources/pro/script"},
		{in: "./internal/resources/pro/script", want: "internal/resources/pro/script"},
		{in: "internal/resources/pro/script", want: "internal/resources/pro/script"},
		{in: module, wantErr: true},
		{in: "github.com/hashicorp/terraform-plugin-framework/types", wantErr: true},
		{in: "  ", wantErr: true},
	} {
		got, err := relative(module, tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("relative(%q) = %q, want an error", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("relative(%q): %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("relative(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// A duplicated package must not be counted twice: acctargets can emit a package
// once per seeding rule, and a lane whose count disagrees with its `go test`
// argument misleads the run summary.
func TestPartitionDeduplicatesAndSortsPackages(t *testing.T) {
	table := mustLoadReal(t)
	entries, err := partition(table, module, imports(
		"internal/resources/pro/script",
		"internal/resources/pro/category",
		"internal/resources/pro/script",
	))
	if err != nil {
		t.Fatalf("partition: %v", err)
	}
	if len(entries) != 1 || entries[0].Count != 2 {
		t.Fatalf("got %+v, want one lane of 2 packages", entries)
	}
	if entries[0].Packages != "./internal/resources/pro/category ./internal/resources/pro/script" {
		t.Errorf("packages = %q, want sorted and deduplicated", entries[0].Packages)
	}
}

// --- helpers ----------------------------------------------------------------

func allLanes(table laneTable) []laneDef {
	return append(append([]laneDef{}, table.Lanes...), table.DefaultLane)
}

// inDir changes the process working directory for the duration of a test.
func inDir(t *testing.T, dir string) {
	t.Helper()
	prior, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(prior); err != nil {
			t.Fatalf("restoring cwd: %v", err)
		}
	})
}

// captureStdout collects what fn prints, because the workflow captures stdout
// verbatim as the matrix and anything extra on it is a parse failure.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	prior := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		var sb strings.Builder
		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			sb.Write(buf[:n])
			if err != nil {
				break
			}
		}
		done <- sb.String()
	}()
	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("closing pipe: %v", err)
	}
	os.Stdout = prior
	return <-done
}
