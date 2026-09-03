// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package conformance

// Source loading for the lane conformance check in acc_lanes_test.go: the lane
// table, the two vocabularies declared in internal/testhelpers, and an AST scan
// of the acceptance suite.
//
// Everything here parses source rather than reflecting over it, for one reason
// that admits no workaround: the acceptance suite and every helper it calls sit
// behind //go:build acceptance, and this check deliberately does not set that
// tag — it must run in `make test`, with no credentials and no network, so a
// misfiled test is caught at PR time rather than on a live estate. A build tag
// that is not set means the declarations do not exist at run time, so there is
// nothing to reflect over.

import (
	"encoding/json"
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const (
	// laneTablePath is repo-relative from internal/conformance. The table is the
	// contract this whole file exists to check, and it is read by three
	// consumers — scripts/acclanes, .github/workflows/integration-tests.yml and
	// this test — so it lives as data rather than as Go.
	laneTablePath = "../../.github/acceptance-lanes.json"

	// requireTokensFile holds accPrecheckRequireTokens, the map where the lane
	// vocabulary and the precheck vocabulary meet.
	requireTokensFile = "../testhelpers/accrequire/require.go"

	// legacyEnvFile holds accLegacyEnvNames, whose KEYS are the post-rename
	// acceptance variable names.
	legacyEnvFile = "../testhelpers/accrequire/env.go"

	// testhelpersDir is scanned by naming shape to discover precheck helpers, so
	// that a new product's precheck fails conformance until it has a lane rather
	// than being silently absent from an allow-list.
	testhelpersDir = "../testhelpers"

	// providerSourceDir is scanned for the JAMFPLATFORM_* names the provider
	// itself reads at Configure. Those are the provider's public consumer
	// variables, and deriving them from the provider rather than restating them
	// is what stops a lane declaring a variable nothing reads.
	providerSourceDir = "../provider"

	// internalRoot is the walk root for the acceptance suite. Every acceptance
	// test in the repo lives under internal/.
	internalRoot = ".."
)

// laneDef is one row of the lane table, plus the default lane, which has the
// same shape but is stored separately because it owns everything unclaimed.
type laneDef struct {
	Lane        string   `json:"lane"`
	Packages    []string `json:"packages"`
	Credentials string   `json:"credentials"`
	Require     string   `json:"require"`
	Declares    []string `json:"declares"`
	Lock        bool     `json:"lock"`
	Planned     bool     `json:"planned"`
}

// credentialSet is one entry of credential_sets. Only the fields this check
// asserts against are decoded; evidence and why are prose for a reader.
type credentialSet struct {
	SecretPrefix string   `json:"secret_prefix"`
	Provides     []string `json:"provides"`
}

type laneTable struct {
	CredentialSets map[string]credentialSet `json:"credential_sets"`
	Lanes          []laneDef                `json:"lanes"`
	DefaultLane    laneDef                  `json:"default_lane"`
}

// loadLaneTable reads and minimally validates the lane table.
//
// Only the conditions that would make every later assertion meaningless are
// fatal here — an unreadable or empty table, or a lane with no package prefixes
// that is not the default lane. Everything else is reported as a finding by the
// tests, because a t.Fatalf stops at the first problem while the owner of the
// table wants the whole list.
func loadLaneTable(t *testing.T) laneTable {
	t.Helper()

	raw, err := os.ReadFile(laneTablePath)
	if err != nil {
		t.Fatalf("reading %s: %v", laneTablePath, err)
	}
	var table laneTable
	if err := json.Unmarshal(raw, &table); err != nil {
		t.Fatalf("parsing %s: %v", laneTablePath, err)
	}
	if len(table.Lanes) == 0 || table.DefaultLane.Lane == "" {
		t.Fatalf("%s declares no lanes, so nothing can be checked against it", laneTablePath)
	}
	for _, lane := range table.Lanes {
		if len(lane.Packages) == 0 {
			t.Fatalf("lane %q in %s names no package prefixes, so it can never own a test; only the default lane may be empty", lane.Lane, laneTablePath)
		}
	}
	return table
}

// allLanes returns every lane including the default one, which is the unit most
// assertions want: the default lane is a lane in every respect except that its
// membership is "everything unclaimed" rather than a prefix list.
func (table laneTable) allLanes() []laneDef {
	return append(slices.Clone(table.Lanes), table.DefaultLane)
}

// lanesRequiring returns the lanes whose require token is one of the given
// tokens, in table order.
func (table laneTable) lanesRequiring(tokens []string) []string {
	var out []string
	for _, lane := range table.allLanes() {
		if slices.Contains(tokens, lane.Require) {
			out = append(out, lane.Lane)
		}
	}
	return out
}

// laneForPackage returns the lane that owns a package: the first lane with a
// matching prefix, else the default lane. First match wins, which is why
// TestAcceptanceLaneTableIsUsable also asserts no two lanes claim one package —
// otherwise lane order would be silently deciding it.
func (table laneTable) laneForPackage(pkg string) string {
	for _, lane := range table.Lanes {
		if lane.claims(pkg) {
			return lane.Lane
		}
	}
	return table.DefaultLane.Lane
}

func (lane laneDef) claims(pkg string) bool {
	for _, prefix := range lane.Packages {
		if packageMatchesPrefix(pkg, prefix) {
			return true
		}
	}
	return false
}

// packageMatchesPrefix reports whether a module-relative package path falls
// under a lane prefix, matching at a PATH BOUNDARY rather than as a raw string
// prefix.
//
// This is a hazard the SDK's name-regex form did not have: a lane prefix
// internal/resources/account/ must claim internal/resources/account and
// internal/resources/account/sso_domain, and must NOT claim a future
// internal/resources/account_thing — which a strings.HasPrefix on the bare
// prefix would silently swallow into the account lane, where it would run under
// an organization credential and 403 in a way that reads as a broken endpoint.
// Comparing pkg+"/" against a prefix normalised to end in "/" gets all three
// cases right, and TestAcceptanceLaneTableIsUsable asserts the negative case
// against every prefix the table actually ships.
func packageMatchesPrefix(pkg, prefix string) bool {
	if prefix == "" {
		return false
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return strings.HasPrefix(pkg+"/", prefix)
}

// accTest is one acceptance test function and what its call graph reaches.
//
// Name is the Go function name and Package the module-relative package path
// (internal/resources/pro/category), which is the form the lane table's prefixes
// are written in. Prechecks holds the testhelpers.AccPreCheck* helpers the test
// reaches, sorted; empty means it calls none, which is either a credential-free
// test or a hole, and TestAcceptanceTestsGatingOutsideThePrecheckPath tells
// those apart. Reads holds the credential variables and client constructors the
// test reaches, sorted, and is populated whether or not a precheck was found,
// because it is the evidence an allow-list entry has to carry.
type accTest struct {
	Name      string
	Package   string
	Prechecks []string
	Reads     []string
}

// acceptanceSuite is the parsed suite, indexed the two ways the tests want it.
//
// Packages holds every package with at least one acceptance test, so a lane can
// be asserted to match a package that really exists AND really has tests. A lane
// matching a package with no tests would run nothing and report green, which is
// the failure this file exists to prevent.
type acceptanceSuite struct {
	Tests    []accTest
	Packages []string
}

// scanAcceptanceSuite parses every acceptance-tagged *_test.go under internal/
// and reports, per TestAcc* function, which precheck helpers and which
// credential-bearing calls its call graph reaches.
//
// # Why the walk into t.Run closures matters
//
// ast.Inspect descends into every function literal in the body, which is
// load-bearing twice over. Terraform's own harness takes the precheck as a
// closure — PreCheck: func() { testhelpers.AccPreCheck(t) } — so a body-level
// scan that stopped at the first FuncLit would find no precheck anywhere in the
// suite. And some tests call their precheck inside a t.Run subtest rather than
// at the top of the test, which is legitimate and must not read as "no
// precheck".
//
// # How deep the same-package follow goes
//
// A test's prechecks and credential reads are collected over the FULL
// transitive closure of bare-identifier calls within its own package, not one
// level. One level would have been enough for the cases in the tree today, but
// the closure costs nothing here (a package holds tens of functions, not
// thousands) and the failure mode of stopping too early is the bad one: an
// unfound precheck reads as a credential-free test, which is exactly the
// classification an attacker of this check would want. The closure deliberately
// does NOT cross package boundaries: a call into another internal package cannot
// reach a precheck, because every precheck lives in internal/testhelpers and is
// reached as a qualified testhelpers.AccPreCheck* selector, which is matched
// directly.
//
// Only acceptance-tagged files are scanned, and that is sound rather than a
// simplification: internal/testhelpers is entirely behind //go:build acceptance,
// so a file that calls a precheck must carry the tag or the package would not
// compile.
func scanAcceptanceSuite(t *testing.T, credentialVars []string) acceptanceSuite {
	t.Helper()

	byPackage := map[string][]string{}
	err := filepath.WalkDir(internalRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		tagged, err := requiresAcceptanceTag(p)
		if err != nil {
			return err
		}
		if tagged {
			pkg := modulePackagePath(filepath.Dir(p))
			byPackage[pkg] = append(byPackage[pkg], p)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s for acceptance sources: %v", internalRoot, err)
	}
	if len(byPackage) == 0 {
		t.Fatalf("found no acceptance-tagged *_test.go under %s — has the suite moved, or has the build-constraint detector drifted?", internalRoot)
	}

	suite := acceptanceSuite{}
	for pkg, files := range byPackage {
		funcs := map[string]*funcReach{}
		for _, file := range files {
			parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
			if err != nil {
				t.Fatalf("parsing %s: %v", file, err)
			}
			for _, decl := range parsed.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil || fn.Recv != nil {
					continue
				}
				funcs[fn.Name.Name] = reachOf(fn.Body, credentialVars)
			}
		}

		hasTests := false
		for name := range funcs {
			if !strings.HasPrefix(name, "TestAcc") {
				continue
			}
			hasTests = true
			prechecks, reads := closureOf(funcs, name)
			suite.Tests = append(suite.Tests, accTest{
				Name:      name,
				Package:   pkg,
				Prechecks: prechecks,
				Reads:     reads,
			})
		}
		if hasTests {
			suite.Packages = append(suite.Packages, pkg)
		}
	}

	slices.Sort(suite.Packages)
	slices.SortFunc(suite.Tests, func(a, b accTest) int {
		if c := strings.Compare(a.Package, b.Package); c != 0 {
			return c
		}
		return strings.Compare(a.Name, b.Name)
	})
	return suite
}

// funcReach is what one function's body calls, before the same-package closure
// is taken.
//
// envRead and credentialLits are kept apart rather than folded into one "reads
// os.Getenv(\"JAMFPLATFORM_BASE_URL\")" match, because a credential gate is
// almost never written that way. Both AccPreCheck and internal/provider's
// package-local accPreCheckCredentialsOnly loop over a []string of names and
// call os.Getenv(name), so the variable name is a literal in a slice and never
// the call argument. Requiring an environment read AND a credential-variable
// literal somewhere in the same call graph catches that form, while still
// ignoring a test that merely writes one with t.Setenv.
type funcReach struct {
	prechecks      map[string]bool
	locals         map[string]bool
	constructors   map[string]bool
	credentialLits map[string]bool
	envRead        bool
}

// precheckShape is the naming convention every precheck helper follows.
// Discovering helpers by SHAPE rather than from an allow-list is what makes a
// new product visible: when Jamf Protect or Jamf School lands with an
// AccPreCheckProtect, this picks it up and TestEveryPrecheckHelperOwnsALane
// fails until it has a lane, instead of its tests being treated as
// credential-free and running in the pro lane against Pro credentials.
func isPrecheckName(name string) bool {
	rest, ok := strings.CutPrefix(name, "AccPreCheck")
	if !ok {
		return false
	}
	for _, r := range rest {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return true
}

// credentialConstructors are the calls that turn credentials into a live client.
// jamfplatform.NewClient is the load-bearing one — it is the SDK's only door
// from credentials to a client, so every credentialed request descends from it —
// and testhelpers.NewAcceptanceClient is the suite's wrapper around it. The
// per-namespace wrappers (securitycloud.New, account.New, aigovernance.New) are
// deliberately absent: each takes an already-built client, so listing them would
// add no reachable path that these two do not already cover.
var credentialConstructors = map[string]bool{
	"jamfplatform.NewClient":          true,
	"testhelpers.NewAcceptanceClient": true,
}

// reachOf records what a function body calls: precheck helpers, bare-identifier
// same-package calls, and anything that reaches a credential.
//
// Only a pkg.Fn selector is treated as evidence. A method call on a value says
// nothing about which package minted the receiver, so it is skipped rather than
// guessed at.
func reachOf(body *ast.BlockStmt, credentialVars []string) *funcReach {
	out := &funcReach{
		prechecks:      map[string]bool{},
		locals:         map[string]bool{},
		constructors:   map[string]bool{},
		credentialLits: map[string]bool{},
	}
	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.BasicLit:
			if name, ok := stringLiteral(node); ok && slices.Contains(credentialVars, name) {
				out.credentialLits[name] = true
			}
		case *ast.CallExpr:
			switch fun := node.Fun.(type) {
			case *ast.Ident:
				out.locals[fun.Name] = true
			case *ast.SelectorExpr:
				pkg, ok := fun.X.(*ast.Ident)
				if !ok {
					return true
				}
				qualified := pkg.Name + "." + fun.Sel.Name
				if pkg.Name == "testhelpers" && isPrecheckName(fun.Sel.Name) {
					out.prechecks[fun.Sel.Name] = true
				}
				if credentialConstructors[qualified] {
					out.constructors[qualified] = true
				}
				if isEnvRead(qualified) {
					out.envRead = true
				}
			}
		}
		return true
	})
	return out
}

// isEnvRead reports whether a qualified call reads a process environment
// variable. testhelpers.AccEnv is included because it is the suite's own reader,
// and a gate written against it reaches the credential exactly as os.Getenv
// does.
func isEnvRead(qualified string) bool {
	switch qualified {
	case "os.Getenv", "os.LookupEnv", "testhelpers.AccEnv":
		return true
	}
	return false
}

// closureOf returns the precheck helpers and credential reads a function
// reaches, following bare-identifier calls through its own package.
func closureOf(funcs map[string]*funcReach, start string) (prechecks, reads []string) {
	seenPrechecks := map[string]bool{}
	seenReads := map[string]bool{}
	seenLits := map[string]bool{}
	visited := map[string]bool{}
	envRead := false

	var walk func(string)
	walk = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true
		reach, ok := funcs[name]
		if !ok {
			return
		}
		for p := range reach.prechecks {
			seenPrechecks[p] = true
		}
		for c := range reach.constructors {
			seenReads[c] = true
		}
		for lit := range reach.credentialLits {
			seenLits[lit] = true
		}
		envRead = envRead || reach.envRead
		for local := range reach.locals {
			walk(local)
		}
	}
	walk(start)

	if envRead {
		for lit := range seenLits {
			seenReads[lit] = true
		}
	}
	return setKeys(seenPrechecks), setKeys(seenReads)
}

// setKeys returns a set's keys, sorted. Findings and derived vocabularies are
// both compared and printed, so a stable order is what keeps a failure message
// reproducible between runs.
func setKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// requiresAcceptanceTag reports whether a file is built ONLY with
// -tags=acceptance.
//
// The build expression is parsed and evaluated twice rather than substring-
// matched, because a substring match cannot tell `//go:build acceptance` from
// `//go:build !acceptance`, and the second would put a unit test into the
// acceptance census. Evaluating under a tag set of exactly {acceptance} and
// again under the empty set answers the question the census actually asks: does
// this file exist only when the acceptance tag is set?
func requiresAcceptanceTag(file string) (bool, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return false, err
	}
	for line := range strings.SplitSeq(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package ") {
			return false, nil
		}
		if !constraint.IsGoBuild(line) {
			continue
		}
		expr, err := constraint.Parse(line)
		if err != nil {
			return false, nil
		}
		withTag := expr.Eval(func(tag string) bool { return tag == "acceptance" })
		withoutTag := expr.Eval(func(string) bool { return false })
		return withTag && !withoutTag, nil
	}
	return false, nil
}

// modulePackagePath converts a walk path under internal/ into the
// module-relative form the lane table's package prefixes are written in.
func modulePackagePath(dir string) string {
	rel, err := filepath.Rel(internalRoot, dir)
	if err != nil {
		return filepath.ToSlash(dir)
	}
	if rel == "." {
		return "internal"
	}
	return path.Join("internal", filepath.ToSlash(rel))
}

// parseRequireTokens reads accPrecheckRequireTokens out of
// internal/testhelpers/accrequire/require.go.
//
// Parsed rather than restated. That map is the single place the lane vocabulary
// and the precheck vocabulary meet, so a second copy here would agree with
// itself and prove nothing — which is the whole failure this check exists to
// make impossible.
//
// A nil value (AccPreCheckOffline) comes back as a present key with an empty
// slice, which is the meaningful distinction: the helper is declared and needs
// no credential, as against a helper that is not declared at all.
func parseRequireTokens(t *testing.T) map[string][]string {
	t.Helper()

	lit := findVarCompositeLit(t, requireTokensFile, "accPrecheckRequireTokens")
	out := map[string][]string{}
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := stringLiteral(kv.Key)
		if !ok {
			continue
		}
		tokens := []string{}
		if values, ok := kv.Value.(*ast.CompositeLit); ok {
			for _, v := range values.Elts {
				if token, ok := stringLiteral(v); ok {
					tokens = append(tokens, token)
				}
			}
		}
		out[key] = tokens
	}
	if len(out) == 0 {
		t.Fatalf("parsed no entries from accPrecheckRequireTokens in %s — the parser has drifted from the declaration", requireTokensFile)
	}
	return out
}

// parseLegacyEnvKeys returns the KEYS of accLegacyEnvNames, which are the
// post-rename acceptance variable names. The values are the pre-rename names the
// shim still reads, and a lane must never declare one of those.
func parseLegacyEnvKeys(t *testing.T) []string {
	t.Helper()

	lit := findVarCompositeLit(t, legacyEnvFile, "accLegacyEnvNames")
	var out []string
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if key, ok := stringLiteral(kv.Key); ok {
			out = append(out, key)
		}
	}
	if len(out) == 0 {
		t.Fatalf("parsed no entries from accLegacyEnvNames in %s — the parser has drifted from the declaration", legacyEnvFile)
	}
	slices.Sort(out)
	return out
}

// findVarCompositeLit returns the composite literal a named package-level var is
// initialised with.
func findVarCompositeLit(t *testing.T, file, name string) *ast.CompositeLit {
	t.Helper()

	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
	if err != nil {
		t.Fatalf("parsing %s: %v", file, err)
	}
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, ident := range value.Names {
				if ident.Name != name || i >= len(value.Values) {
					continue
				}
				if lit, ok := value.Values[i].(*ast.CompositeLit); ok {
					return lit
				}
			}
		}
	}
	t.Fatalf("%s declares no package-level var %s initialised with a composite literal", file, name)
	return nil
}

// parsePrecheckHelpers returns the names of every precheck-shaped function
// declared in internal/testhelpers, and the names of every top-level function
// there.
//
// Two answers from one scan because the two tests need opposite directions: a
// precheck-shaped helper that no lane serves is a missing lane, and a key in
// accPrecheckRequireTokens naming no function at all is a stale entry that
// silently exempts whatever reuses the name.
func parsePrecheckHelpers(t *testing.T) (prechecks, allFuncs []string) {
	t.Helper()

	entries, err := os.ReadDir(testhelpersDir)
	if err != nil {
		t.Fatalf("reading %s: %v", testhelpersDir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		file := filepath.Join(testhelpersDir, entry.Name())
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", file, err)
		}
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			allFuncs = append(allFuncs, fn.Name.Name)
			if isPrecheckName(fn.Name.Name) {
				prechecks = append(prechecks, fn.Name.Name)
			}
		}
	}
	if len(prechecks) == 0 {
		t.Fatalf("found no AccPreCheck*-shaped function in %s — the shape detector has drifted from the helpers", testhelpersDir)
	}
	slices.Sort(prechecks)
	slices.Sort(allFuncs)
	return prechecks, allFuncs
}

// parseProviderConsumerVars returns the JAMFPLATFORM_* variables the provider
// itself reads.
//
// These are the provider's public configuration — the provider schema reads them
// at Configure and they are documented for users — so they are the one set of
// names that cannot be renamed, and the only names outside the acceptance
// scheme a lane may legitimately declare or provide. Derived from the provider's
// own source so that a rename there shows up here rather than leaving the lane
// table pointing at a variable nothing reads.
func parseProviderConsumerVars(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir(providerSourceDir)
	if err != nil {
		t.Fatalf("reading %s: %v", providerSourceDir, err)
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file := filepath.Join(providerSourceDir, entry.Name())
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", file, err)
		}
		ast.Inspect(parsed, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok {
				return true
			}
			if name, ok := stringLiteral(lit); ok && isEnvVarName(name) {
				seen[name] = true
			}
			return true
		})
	}
	if len(seen) == 0 {
		t.Fatalf("found no JAMFPLATFORM_* literal in %s — the provider's consumer variables can no longer be derived", providerSourceDir)
	}
	return setKeys(seen)
}

// isEnvVarName reports whether a literal has the shape of a JAMFPLATFORM_*
// environment variable name.
func isEnvVarName(s string) bool {
	rest, ok := strings.CutPrefix(s, "JAMFPLATFORM_")
	if !ok || rest == "" {
		return false
	}
	for _, r := range rest {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}
