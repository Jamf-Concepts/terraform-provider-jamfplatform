//go:build acclanes

// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Command acclanes splits an acceptance test scope into per-product lanes and
// prints the GitHub Actions matrix include[] for them.
//
// It is the second half of the acceptance plan step: scripts/acctargets decides
// WHAT must run, acclanes decides HOW that is split across jobs. Keeping them
// separate keeps scope computation ignorant of lanes, and lane assignment
// ignorant of git.
//
// # Lanes match package prefixes, not test names
//
// This is the one substantive divergence from jamfplatform-go-sdk, whose own
// acclanes it is otherwise a port of. That suite is a single package, so its
// lanes are test-name regexps and each job receives a `-run` alternation of
// every test name it owns. That brings a constraint with it: `go test -run`
// matches with RE2, so a lane pattern must be RE2-compilable, and the SDK's
// tool compiles every lane pattern with the same regexp package that will run
// them so a dialect difference cannot mis-file a test into a lane whose
// credential cannot reach its endpoints.
//
// None of that applies here. This repo is already split into per-family
// packages, so a lane names package path PREFIXES and each job receives the
// package list directly — there is no `-run` alternation, no pattern to
// compile, and therefore no RE2 concern in lane matching at all. Prefix
// matching is done at a path boundary instead, which is the hazard that
// replaces it: a lane owning `internal/resources/account/` must not claim
// `internal/resources/account_thing`.
//
// One part of the SDK's reasoning does survive intact. The default lane is
// "everything no named lane claimed", and that is a PARTITION of a real package
// list rather than a complement of the named lanes' patterns. As a pattern it
// would need negative lookahead, which RE2 rejects outright; as a partition it
// is a fall-through branch, and it makes the full-suite and change-scoped runs
// one code path.
//
// # Why Go rather than a shell or Python script
//
// Enumerating the full suite means asking the toolchain which packages carry
// acceptance tests, which is a `go list -tags acceptance -json` call and a
// build-constraint scan — the same idiom scripts/acctargets uses, deliberately,
// so the two tools agree on what "the acceptance suite" is. Being a Go command
// also puts the partition under `go test`, so it is covered by CI rather than
// by whoever last ran the script by hand.
//
// Output on stdout is the matrix include[] array and nothing else, because the
// workflow captures stdout verbatim. An empty scope prints `[]`.
//
// Usage:
//
//	go run -tags acclanes ./scripts/acclanes -scope "$(go run -tags acctargets ./scripts/acctargets)"
//
// The scope is whatever acctargets printed: `./...`, a space-separated list of
// package import paths, or the empty string. The -table default is relative to
// the repo root, which is where the workflow invokes this from.
//
// The build tag keeps the tool out of `go build ./...` while still letting
// `go test -tags acclanes ./scripts/acclanes` exercise it. `ignore` would do
// the first job but not the second: passing `-tags ignore` to a package-shaped
// build also satisfies the tag on the standard library's own generator files,
// which then fail to load.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// credentialSet is one entry of the lane table's credential_sets map. Only the
// field a consumer acts on is decoded: `provides`, `evidence` and `why` are for
// the reader and for .github/workflows, and are deliberately not modelled here.
type credentialSet struct {
	SecretPrefix string `json:"secret_prefix"`
}

// laneDef is one row of .github/acceptance-lanes.json. That file is the single
// source of truth, shared with internal/conformance, which asserts the
// package-based partition agrees with the precheck helper each package's tests
// actually call.
type laneDef struct {
	Lane string `json:"lane"`
	// Packages are module-relative path PREFIXES, matched at a path boundary.
	// The default lane must declare none: it is everything unclaimed.
	Packages []string `json:"packages"`
	// Credentials keys into laneTable.CredentialSets. A planned lane carries
	// the empty string, because wiring a credential set ahead of the product is
	// what the stale-row check exists to reject.
	Credentials string `json:"credentials"`
	Require     string `json:"require"`
	// Declares are the entitlement declarations the lane needs over and above
	// its credential set. Carried into the matrix so the workflow can surface
	// which variables a lane's step must set.
	Declares []string `json:"declares"`
	Lock     bool     `json:"lock"`
	// TimeoutMinutes is the lane's own GitHub job ceiling. It has to come from
	// the table rather than being a constant on the job, because the lanes
	// differ by two orders of magnitude in size — and it cannot be left to
	// `go test -timeout`, which is per TEST BINARY and so bounds one package
	// rather than the lane.
	TimeoutMinutes int `json:"timeout_minutes"`
	// Planned reserves a lane's name and package prefixes before its product
	// exists. A planned lane must match nothing; if it matches, partition fails
	// rather than emitting a job for a lane with no credential wired. Reserving
	// the prefixes is what stops a new product's packages landing in the pro
	// lane and running against Pro credentials, where the 403s read as broken
	// endpoints rather than as a misfiled lane.
	Planned bool `json:"planned"`
}

type laneTable struct {
	CredentialSets map[string]credentialSet `json:"credential_sets"`
	Lanes          []laneDef                `json:"lanes"`
	DefaultLane    laneDef                  `json:"default_lane"`
}

// matrixEntry is one GitHub matrix include[] element. Field names are the
// `matrix.<field>` the workflow reads, so they are chosen to read well there.
//
// Count is carried for the job summary: a lane's size is the first thing a
// reader wants when a run is slower or emptier than expected. Packages is the
// literal `go test` argument, in ./internal/... form so the command line in the
// job log is readable. Declares is space-separated rather than a list because a
// matrix value must be a scalar for the workflow to interpolate it.
//
// Credentials and SecretPrefix are both carried on purpose. SecretPrefix is
// what the workflow indexes secrets with —
// `${{ secrets[format('{0}_BASE_URL', matrix.secret_prefix)] }}` — so it has to
// arrive as its own scalar field; Credentials is the set's human-readable name
// and is what the job name and the run summary read.
type matrixEntry struct {
	Lane         string `json:"lane"`
	Require      string `json:"require"`
	Lock         bool   `json:"lock"`
	Credentials  string `json:"credentials"`
	SecretPrefix string `json:"secret_prefix"`
	Count        int    `json:"count"`
	Packages     string `json:"packages"`
	Declares     string `json:"declares"`
	// TimeoutMinutes reaches the workflow as `timeout-minutes: ${{
	// matrix.timeout_minutes }}`, so a lane that plans 110 packages and one
	// that plans two do not share a ceiling.
	TimeoutMinutes int `json:"timeout_minutes"`
}

// defaultTablePath is relative to the repo root, which is where the workflow
// invokes `go run -tags acclanes ./scripts/acclanes` from.
const defaultTablePath = ".github/acceptance-lanes.json"

// scopeAll is acctargets' full-suite output.
const scopeAll = "./..."

func loadTable(path string) (laneTable, error) {
	var table laneTable
	raw, err := os.ReadFile(path)
	if err != nil {
		return table, fmt.Errorf("reading lane table: %w", err)
	}
	// The table's top-level "_comment" array is documentation for whoever edits
	// it, and is ignored here the way any unknown field is.
	if err := json.Unmarshal(raw, &table); err != nil {
		return table, fmt.Errorf("parsing lane table: %w", err)
	}
	if len(table.Lanes) == 0 || table.DefaultLane.Lane == "" {
		return table, fmt.Errorf("lane table declares no lanes")
	}
	return table, nil
}

// claims reports whether a lane prefix owns a module-relative package path.
// The prefix must match at a path BOUNDARY, so `internal/resources/account/`
// claims `internal/resources/account/sso_domain` and the package
// `internal/resources/account` itself, but never `internal/resources/account_thing`.
// A trailing slash in the table is conventional rather than load-bearing.
func claims(prefix, rel string) bool {
	p := strings.Trim(prefix, "/")
	if p == "" {
		return false
	}
	return rel == p || strings.HasPrefix(rel, p+"/")
}

// partition assigns each package to the first lane one of whose prefixes claims
// it, else the default lane. First-match-wins makes the table readable as a
// priority list; the conformance test rejects a table where two lanes claim the
// same package, so order never silently decides anything here.
func partition(table laneTable, module string, imports []string) ([]matrixEntry, error) {
	rels := make([]string, 0, len(imports))
	for _, imp := range imports {
		rel, err := relative(module, imp)
		if err != nil {
			return nil, err
		}
		rels = append(rels, rel)
	}
	// Sorted so the matrix is byte-identical for the same package set however
	// the scope was produced. Ordering only affects presentation: `go test`
	// takes the packages as a set, and -p=1 already serialises the locked lane.
	slices.Sort(rels)
	rels = slices.Compact(rels)

	buckets := map[string][]string{}
	for _, rel := range rels {
		placed := false
		for _, lane := range table.Lanes {
			if slices.ContainsFunc(lane.Packages, func(p string) bool { return claims(p, rel) }) {
				buckets[lane.Lane] = append(buckets[lane.Lane], rel)
				placed = true
				break
			}
		}
		if !placed {
			buckets[table.DefaultLane.Lane] = append(buckets[table.DefaultLane.Lane], rel)
		}
	}

	// A planned lane that matched packages means the product arrived and the
	// wiring did not. Fail the plan step: running those tests in any lane would
	// use a credential nobody chose for them.
	for _, lane := range table.Lanes {
		if lane.Planned && len(buckets[lane.Lane]) > 0 {
			return nil, fmt.Errorf("lane %q is still marked planned but now claims %d package(s) including %q: add a precheck helper in internal/testhelpers, add the lane's row to the conformance table, give it a require token, name a credential set in credential_sets, wire its secrets on the workflow step, then drop \"planned\" from the lane table",
				lane.Lane, len(buckets[lane.Lane]), buckets[lane.Lane][0])
		}
	}

	// Emit in table order, default last, and drop empty lanes: a job that runs
	// zero tests reports a passing check having asserted nothing, which is the
	// failure mode JAMFPLATFORM_ACC_REQUIRE exists to close.
	var out []matrixEntry
	for _, lane := range append(append([]laneDef{}, table.Lanes...), table.DefaultLane) {
		claimed := buckets[lane.Lane]
		if len(claimed) == 0 {
			continue
		}
		prefix, err := secretPrefix(table, lane)
		if err != nil {
			return nil, err
		}
		// A hard error rather than a default, for the same reason secretPrefix
		// is: the workflow interpolates this into `timeout-minutes:`, where an
		// empty value is a workflow syntax error naming the expression rather
		// than the lane, and a silent 0 would be read by GitHub as "no limit"
		// — restoring exactly the 6h exposure the field exists to close.
		if lane.TimeoutMinutes <= 0 {
			return nil, fmt.Errorf("lane %q claims packages but declares no positive timeout_minutes: give it a job ceiling in the lane table, sized from how long the lane actually takes (go test -timeout cannot do this — it is per test binary, so it bounds one package and not the lane)", lane.Lane)
		}
		args := make([]string, 0, len(claimed))
		for _, rel := range claimed {
			args = append(args, "./"+rel)
		}
		out = append(out, matrixEntry{
			Lane:           lane.Lane,
			Require:        lane.Require,
			Lock:           lane.Lock,
			Credentials:    lane.Credentials,
			SecretPrefix:   prefix,
			Count:          len(claimed),
			Packages:       strings.Join(args, " "),
			Declares:       strings.Join(lane.Declares, " "),
			TimeoutMinutes: lane.TimeoutMinutes,
		})
	}
	return out, nil
}

// secretPrefix resolves an emitted lane's credential set to the secret-name
// prefix the workflow indexes with.
//
// Both failure modes are hard errors rather than an empty string, because an
// empty prefix would have the workflow evaluate `secrets['_BASE_URL']`, get
// nothing, and fail every test in the lane with "credentials not configured" —
// which reads as a missing secret rather than as a broken lane table. A planned
// lane legitimately has no credential set, but a planned lane never reaches
// here: partition rejects one that claimed a package before it emits anything.
func secretPrefix(table laneTable, lane laneDef) (string, error) {
	if lane.Credentials == "" {
		return "", fmt.Errorf("lane %q claims packages but names no credential set: give it a credentials key from credential_sets (or mark it planned if its product has not landed)", lane.Lane)
	}
	set, known := table.CredentialSets[lane.Credentials]
	if !known {
		return "", fmt.Errorf("lane %q names credential set %q, which credential_sets does not declare", lane.Lane, lane.Credentials)
	}
	if set.SecretPrefix == "" {
		return "", fmt.Errorf("credential set %q (used by lane %q) declares no secret_prefix", lane.Credentials, lane.Lane)
	}
	return set.SecretPrefix, nil
}

// relative strips the module path from an import path. Both forms are accepted
// because acctargets prints full import paths while a human running this by
// hand reaches for ./internal/...; anything outside the module is an error
// rather than a silent default-lane assignment, since a scope this tool cannot
// place is a scope it must not narrow.
func relative(module, imp string) (string, error) {
	rel := strings.TrimSpace(imp)
	if rel == "" {
		return "", fmt.Errorf("empty package path in scope")
	}
	rel = strings.TrimSuffix(strings.TrimPrefix(rel, "./"), "/")
	switch {
	case module == "":
	case rel == module:
		return "", fmt.Errorf("package %q is the module root and carries no acceptance tests", imp)
	case strings.HasPrefix(rel, module+"/"):
		rel = strings.TrimPrefix(rel, module+"/")
	case strings.Contains(strings.SplitN(rel, "/", 2)[0], "."):
		// A dot in the first path segment means a domain, so this is a full
		// import path belonging to some other module.
		return "", fmt.Errorf("package %q is not in module %q", imp, module)
	}
	if rel == "" {
		return "", fmt.Errorf("empty package path in scope")
	}
	return rel, nil
}

// --- scope resolution -------------------------------------------------------

// goListPackage is the subset of `go list -json` fields we consume, matching
// scripts/acctargets so both tools read the suite the same way.
type goListPackage struct {
	ImportPath   string
	Dir          string
	TestGoFiles  []string
	XTestGoFiles []string
}

// resolveScope turns acctargets' output into the package list to split. An
// explicit list is consumed as given rather than re-derived, so acclanes can
// never widen what acctargets chose; `./...` is enumerated from the toolchain.
func resolveScope(scope string) ([]string, error) {
	if strings.TrimSpace(scope) != scopeAll {
		imports := strings.Fields(scope)
		if len(imports) == 0 {
			return nil, fmt.Errorf("could not parse scope %q", scope)
		}
		return imports, nil
	}

	pkgs, err := listPackages()
	if err != nil {
		return nil, fmt.Errorf("listing packages: %w", err)
	}
	imports := acceptanceCandidates(pkgs)
	// An empty list under ./... must fail loudly. Emitting an empty matrix
	// would skip the entire acceptance suite and still report success — the
	// same skip-into-green failure the require tokens exist to close.
	if len(imports) == 0 {
		return nil, fmt.Errorf("scope is %s but no package carries an //go:build acceptance test file — `go list -tags acceptance -json ./...` found nothing to run", scopeAll)
	}
	return imports, nil
}

func listPackages() ([]goListPackage, error) {
	// -tags acceptance so the acceptance-only test files are visible at all:
	// without it go list reports them as IgnoredGoFiles and every package looks
	// testless.
	cmd := exec.Command("go", "list", "-tags", "acceptance", "-json", "./...")
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	var pkgs []goListPackage
	dec := json.NewDecoder(bufio.NewReader(stdout))
	for dec.More() {
		var p goListPackage
		if err := dec.Decode(&p); err != nil {
			return nil, err
		}
		pkgs = append(pkgs, p)
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}
	return pkgs, nil
}

// acceptanceCandidates returns the import paths of packages carrying at least
// one test file guarded by `//go:build acceptance`.
//
// The build-tag scan rather than a bare "has test files" check is what keeps
// this tool's ./... answer the same shape as acctargets' change-scoped one:
// acctargets restricts its output to acceptance candidates, so if ./... here
// admitted unit-test-only packages the two paths would disagree about what the
// acceptance suite is, and the locked pro lane would serialise a pile of
// offline unit tests behind the tenant.
func acceptanceCandidates(pkgs []goListPackage) []string {
	var out []string
	for _, p := range pkgs {
		files := append(append([]string{}, p.XTestGoFiles...), p.TestGoFiles...)
		for _, f := range files {
			if hasAcceptanceTag(filepath.Join(p.Dir, f)) {
				out = append(out, p.ImportPath)
				break
			}
		}
	}
	return out
}

func hasAcceptanceTag(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	lines := 0
	for sc.Scan() && lines < 20 { // build constraints live at the top of the file
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "//go:build ") && strings.Contains(line, "acceptance") {
			return true
		}
		lines++
	}
	return false
}

func modulePath() (string, error) {
	out, err := exec.Command("go", "list", "-m").Output()
	if err != nil {
		return "", fmt.Errorf("reading module path: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func main() {
	table := flag.String("table", defaultTablePath, "path to the lane table, relative to the repo root")
	scope := flag.String("scope", scopeAll, "scope as printed by acctargets: ./... , a space-separated package list, or empty")
	flag.Parse()

	if err := run(*table, *scope); err != nil {
		fmt.Fprintln(os.Stderr, "acclanes:", err)
		os.Exit(1)
	}
}

func run(tablePath, scope string) error {
	// Nothing affected is not a failure: acctargets prints an empty scope for a
	// change that touches no package, and an empty matrix is the correct answer.
	if strings.TrimSpace(scope) == "" {
		fmt.Println("[]")
		return nil
	}

	table, err := loadTable(tablePath)
	if err != nil {
		return err
	}
	module, err := modulePath()
	if err != nil {
		return err
	}
	imports, err := resolveScope(scope)
	if err != nil {
		return err
	}
	entries, err := partition(table, module, imports)
	if err != nil {
		return err
	}
	// Marshal the empty slice as [] rather than null: a matrix value of null is
	// a workflow error, where [] is an honestly empty matrix.
	if entries == nil {
		entries = []matrixEntry{}
	}
	out, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}
