//go:build acctargets

// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Command acctargets prints the minimal set of Go packages whose acceptance
// tests must run for a change set: the changed packages plus every package
// that (transitively) depends on them. It exists so surgical PRs don't pay for
// the full ~2h serial acceptance suite.
//
// The dependency graph has one trap: internal/provider imports every resource
// package purely to register it, and internal/testhelpers imports the provider
// to build the test provider factories. So every acceptance test binary
// transitively "depends on" every resource. A naive reverse-dependency walk
// would therefore mark the whole suite dirty on any single-resource change.
//
// The fix is to CUT internal/provider's out-edges from the graph: the node
// stays, but the edges provider -> {every resource} are dropped. Real edges
// survive — a package's own deps, shared helpers (internal/common/*,
// providerdata), and any cross-package test fixtures — so changing
// internal/common/scope still re-tests exactly the scope-bearing resources, no
// more, no less.
//
// # Hub packages, and why the import graph alone is not enough
//
// Three packages are imported by (effectively) everything: internal/provider,
// internal/providerdata and internal/testhelpers. A reverse-dependency walk from
// any of them selects the whole suite — measured at 114-117 of 118 acceptance
// packages. That is correct for a change to shared behaviour and badly wrong for
// the commonest change of all: ADDING something. A new
// providerdata.ConfigureSecurityCloud, a new testhelpers fixture, a new
// validator — none of them can affect a package that does not call them, and no
// unchanged package can call a declaration that did not exist at the base ref.
//
// So changed Go files are attributed at DECLARATION granularity, not file
// granularity, borrowing the idea from the SDK's own acctargets. Each changed
// .go file is parsed at the merge base and at HEAD and its top-level
// declarations compared by printed source (doc comments excluded, so a
// comment-only edit is a no-op). Two rules then apply, both chosen because they
// are provably safe rather than merely plausible:
//
//	Rule A — additive-only. If every changed declaration in the file is NEW,
//	the file contributes its own package and nothing else. An unchanged package
//	cannot reference a declaration that did not exist at the base; a package that
//	gained such a reference changed too, and selects itself.
//
//	Rule B — registration lists. internal/provider/provider.go's Resources /
//	DataSources / ListResources / Actions / Functions methods are pure lists of
//	constructor references. When the file's only modified declarations are those
//	methods, the blast radius is exactly the packages whose constructors were
//	added or removed — resolved through the file's import block at each revision.
//	Reordering a list changes no symbol and so selects nothing, which is right.
//
// Everything else keeps the old behaviour: the file's package is seeded and the
// reverse closure runs. Every uncertainty — an unparseable revision, a modified
// or deleted declaration, a registration method whose shape is not a plain
// composite literal — resolves to that wider answer. Over-running costs time;
// under-running costs a regression.
//
// Output on stdout is one of:
//   - "./..."               full suite (a global file such as go.mod changed)
//   - "<imp> <imp> ..."     explicit space-separated package import paths
//   - ""                    nothing affected (skip the acceptance run)
//
// Usage:
//
//	go run -tags acctargets ./scripts/acctargets [baseRef]   # baseRef default: origin/main
//	BASE_REF=origin/main go run -tags acctargets ./scripts/acctargets
//
// The build tag keeps the tool out of `go build ./...` while still letting
// `go test -tags acctargets ./scripts/acctargets` exercise it. `ignore` would do
// the first job but not the second: passing `-tags ignore` to a package-shaped
// build also satisfies the tag on the standard library's own generator files,
// which then fail to load.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// goListPackage is the subset of `go list -json` fields we consume.
type goListPackage struct {
	ImportPath   string
	Name         string
	Dir          string
	Imports      []string
	TestImports  []string
	XTestImports []string
	TestGoFiles  []string
	XTestGoFiles []string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "acctargets:", err)
		os.Exit(1)
	}
}

func run() error {
	baseRef := os.Getenv("BASE_REF")
	if len(os.Args) > 1 && os.Args[1] != "" {
		baseRef = os.Args[1]
	}
	if baseRef == "" {
		baseRef = "origin/main"
	}

	root, err := gitOutput("rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("locating repo root: %w", err)
	}
	root = strings.TrimSpace(root)

	module, err := cmdOutput(root, "go", "list", "-m")
	if err != nil {
		return fmt.Errorf("reading module path: %w", err)
	}
	module = strings.TrimSpace(module)

	pkgs, err := listPackages(root)
	if err != nil {
		return fmt.Errorf("listing packages: %w", err)
	}

	changedFiles, diffBase, err := changedFiles(root, baseRef)
	if err != nil {
		return fmt.Errorf("computing changed files: %w", err)
	}

	// Map every changed file to a changed package import path. A file that is
	// neither ignorable (docs/examples/markdown) nor mappable to a package, nor
	// a recognised global trigger, forces a full run — the safe default.
	//
	// fanOut packages take the reverse-dependency closure; selfOnly packages are
	// affected in isolation because the change to them was purely additive (see
	// the Rule A note in the package comment).
	fanOut := map[string]bool{}
	selfOnly := map[string]bool{}
	pkgNames := packageNames(pkgs)
	providerPkg := module + "/internal/provider"
	dirIndex := newDirIndex(pkgs)

	for _, f := range changedFiles {
		switch classifyFile(f) {
		case fileIgnore:
			continue
		case fileGlobal:
			fmt.Println("./...")
			return nil
		case filePackage:
			abs := filepath.Join(root, f)
			imp := dirIndex.lookup(filepath.Dir(abs))
			if imp == "" {
				// A changed Go-ish file we cannot attribute to a package (new
				// dir, deleted package, testdata without a package). Be
				// conservative.
				fmt.Println("./...")
				return nil
			}

			// Rule B: a registration-only edit to the provider hub reaches
			// exactly the constructs whose constructors moved.
			if imp == providerPkg && filepath.Base(f) == "provider.go" {
				if delta, confined := registrationDelta(root, diffBase, f, module, pkgNames); confined {
					for target := range delta {
						fanOut[target] = true
					}
					continue
				}
			}

			// Rule A: an additive-only edit cannot reach an unchanged package.
			switch classifyGoFileChange(root, diffBase, f) {
			case changeNoop:
				continue
			case changeAdditive:
				selfOnly[imp] = true
			default:
				fanOut[imp] = true
			}
		}
	}

	if len(fanOut) == 0 && len(selfOnly) == 0 {
		// Nothing code-relevant changed.
		return nil
	}

	// Reverse-dependency closure over the cut graph, plus the self-only packages.
	rev := buildReverseGraph(pkgs, module, providerPkg)
	affected := reverseClosure(rev, fanOut)
	for imp := range selfOnly {
		affected[imp] = true
	}

	// Restrict to packages that actually carry acceptance tests.
	candidates := acceptanceCandidates(pkgs)

	var out []string
	for imp := range affected {
		if candidates[imp] {
			out = append(out, imp)
		}
	}
	sort.Strings(out)
	if len(out) > 0 {
		fmt.Println(strings.Join(out, " "))
	}
	return nil
}

// --- package listing -------------------------------------------------------

func listPackages(root string) ([]goListPackage, error) {
	// -tags acceptance so the acceptance-only test files (and their imports)
	// are visible to the graph.
	cmd := exec.Command("go", "list", "-tags", "acceptance", "-json", "./...")
	cmd.Dir = root
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

// --- dir -> import path resolution -----------------------------------------

type dirIndex struct {
	dirs []string          // sorted longest-first for prefix matching
	imp  map[string]string // dir -> import path
}

func newDirIndex(pkgs []goListPackage) *dirIndex {
	di := &dirIndex{imp: map[string]string{}}
	for _, p := range pkgs {
		if p.Dir == "" {
			continue
		}
		di.imp[p.Dir] = p.ImportPath
		di.dirs = append(di.dirs, p.Dir)
	}
	sort.Slice(di.dirs, func(i, j int) bool { return len(di.dirs[i]) > len(di.dirs[j]) })
	return di
}

// lookup returns the import path of the package owning dir, walking up to the
// nearest ancestor that is a package directory.
func (di *dirIndex) lookup(dir string) string {
	for _, d := range di.dirs {
		if dir == d || strings.HasPrefix(dir, d+string(os.PathSeparator)) {
			return di.imp[d]
		}
	}
	return ""
}

// --- reverse graph + closure ------------------------------------------------

// buildReverseGraph builds the reverse import graph over module-internal edges.
// providerPkg's out-edges are omitted (the registration hub cut).
func buildReverseGraph(pkgs []goListPackage, module, providerPkg string) map[string][]string {
	rev := map[string][]string{}
	for _, p := range pkgs {
		if p.ImportPath == providerPkg {
			continue // cut: drop the registration fan-out
		}
		seen := map[string]bool{}
		add := func(imports []string) {
			for _, dep := range imports {
				if dep == p.ImportPath || seen[dep] {
					continue
				}
				if !strings.HasPrefix(dep, module+"/") {
					continue // external dep — never in the changed set
				}
				seen[dep] = true
				rev[dep] = append(rev[dep], p.ImportPath)
			}
		}
		add(p.Imports)
		add(p.TestImports)
		add(p.XTestImports)
	}
	return rev
}

// reverseClosure returns the seed set plus every package that transitively
// depends on a seed.
func reverseClosure(rev map[string][]string, seed map[string]bool) map[string]bool {
	visited := map[string]bool{}
	var queue []string
	for s := range seed {
		if !visited[s] {
			visited[s] = true
			queue = append(queue, s)
		}
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, dependent := range rev[cur] {
			if !visited[dependent] {
				visited[dependent] = true
				queue = append(queue, dependent)
			}
		}
	}
	return visited
}

// --- acceptance candidate detection ----------------------------------------

// acceptanceCandidates returns packages that carry at least one test file
// guarded by `//go:build acceptance`.
func acceptanceCandidates(pkgs []goListPackage) map[string]bool {
	out := map[string]bool{}
	for _, p := range pkgs {
		files := append(append([]string{}, p.XTestGoFiles...), p.TestGoFiles...)
		for _, f := range files {
			if hasAcceptanceTag(filepath.Join(p.Dir, f)) {
				out[p.ImportPath] = true
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

// --- declaration-level attribution -----------------------------------------

// changeKind describes what a single changed .go file did to its declarations.
type changeKind int

const (
	// changeNoop means no top-level declaration differs: a comment, import or
	// formatting edit. Contributes nothing.
	changeNoop changeKind = iota
	// changeAdditive means every difference is a NEW declaration. No unchanged
	// package can reference it, so only the file's own package is affected.
	changeAdditive
	// changeModified means an existing declaration changed or was removed, or
	// the file could not be attributed confidently. The reverse closure runs.
	changeModified
)

// registrationMethods are the provider methods whose bodies are pure lists of
// construct constructors. A change confined to these has a blast radius of
// exactly the packages whose constructors moved.
var registrationMethods = map[string]bool{
	"Resources":     true,
	"DataSources":   true,
	"ListResources": true,
	"Actions":       true,
	"Functions":     true,
}

// classifyGoFileChange compares a file's top-level declarations at the diff base
// and at HEAD. Anything it cannot establish confidently returns changeModified,
// which is the pre-existing whole-package behaviour.
func classifyGoFileChange(root, diffBase, path string) changeKind {
	if !strings.HasSuffix(path, ".go") {
		return changeModified
	}

	head, headErr := declsAtWorkingTree(root, path)
	if headErr != nil {
		// Deleted, or unparseable at HEAD. Removing a declaration is exactly the
		// case Rule A must not cover.
		return changeModified
	}

	base, baseErr := declsAtRev(root, diffBase, path)
	if baseErr != nil {
		// Absent at the base ref: a brand-new file, every declaration new.
		return changeAdditive
	}

	additions := 0
	for name, src := range head {
		prior, existed := base[name]
		if !existed {
			additions++
			continue
		}
		if prior != src {
			return changeModified
		}
	}
	for name := range base {
		if _, still := head[name]; !still {
			return changeModified // removal
		}
	}
	if additions == 0 {
		return changeNoop
	}
	return changeAdditive
}

// registrationDelta reports the packages whose construct registration changed in
// provider.go, and whether the file's changes are confined to the registration
// methods.
//
// confined is false the moment anything else in the file differs — a Configure
// edit, a new provider attribute, a changed helper — because those reach every
// construct and must keep fanning out.
func registrationDelta(root, diffBase, path, module string, pkgNames map[string]string) (targets map[string]bool, confined bool) {
	headFile, headFset, err := parseFileAtWorkingTree(root, path)
	if err != nil {
		return nil, false
	}
	baseFile, baseFset, err := parseFileAtRev(root, diffBase, path)
	if err != nil {
		return nil, false
	}

	head := declSources(headFile, headFset)
	base := declSources(baseFile, baseFset)

	// Every difference outside the registration methods disqualifies the file.
	for name, src := range head {
		if prior, existed := base[name]; !existed || prior != src {
			if !isRegistrationDecl(name) {
				return nil, false
			}
		}
	}
	for name := range base {
		if _, still := head[name]; !still && !isRegistrationDecl(name) {
			return nil, false
		}
	}

	headSyms, ok := registrationSymbols(headFile, module, pkgNames)
	if !ok {
		return nil, false
	}
	baseSyms, ok := registrationSymbols(baseFile, module, pkgNames)
	if !ok {
		return nil, false
	}

	// The delta is over individual constructors, not their packages. Dropping
	// one construct from a package that still registers others leaves the
	// package set unchanged while breaking that construct's acceptance test, so
	// a package-level diff would miss it.
	targets = map[string]bool{}
	for sym, imp := range headSyms {
		if _, unchanged := baseSyms[sym]; !unchanged {
			targets[imp] = true
		}
	}
	for sym, imp := range baseSyms {
		if _, unchanged := headSyms[sym]; !unchanged {
			targets[imp] = true
		}
	}
	return targets, true
}

// isRegistrationDecl reports whether a declaration key names one of the
// registration methods. Keys for methods are "RecvType.Name".
func isRegistrationDecl(key string) bool {
	if i := strings.LastIndex(key, "."); i >= 0 {
		return registrationMethods[key[i+1:]]
	}
	return registrationMethods[key]
}

// registrationSymbols maps each constructor the registration methods reference
// ("<import path>.<Constructor>") to the import path declaring it. ok is false
// when a method's body is not the plain "return []func() T{a.B, c.D}" shape this
// can reason about — a conditional or a helper call means the list is no longer a
// list, and its contents are no longer statically knowable.
func registrationSymbols(file *ast.File, module string, pkgNames map[string]string) (map[string]string, bool) {
	aliases := importAliases(file, pkgNames)
	out := map[string]string{}

	for _, decl := range file.Decls {
		fn, isFunc := decl.(*ast.FuncDecl)
		if !isFunc || fn.Body == nil || !registrationMethods[fn.Name.Name] || fn.Recv == nil {
			continue
		}
		if len(fn.Body.List) != 1 {
			return nil, false
		}
		ret, isReturn := fn.Body.List[0].(*ast.ReturnStmt)
		if !isReturn || len(ret.Results) != 1 {
			return nil, false
		}
		lit, isLit := ret.Results[0].(*ast.CompositeLit)
		if !isLit {
			return nil, false
		}
		for _, elt := range lit.Elts {
			sel, isSel := elt.(*ast.SelectorExpr)
			if !isSel {
				return nil, false
			}
			pkgIdent, isIdent := sel.X.(*ast.Ident)
			if !isIdent {
				return nil, false
			}
			imp, known := aliases[pkgIdent.Name]
			if !known || !strings.HasPrefix(imp, module+"/") {
				return nil, false
			}
			out[imp+"."+sel.Sel.Name] = imp
		}
	}
	return out, true
}

// importAliases maps the identifier each import is referenced by to its import
// path. An explicit alias wins; otherwise the package's declared name is used,
// which is not always the last path segment (internal/actions/device declares
// package deviceactions).
func importAliases(file *ast.File, pkgNames map[string]string) map[string]string {
	out := map[string]string{}
	for _, spec := range file.Imports {
		imp := strings.Trim(spec.Path.Value, `"`)
		switch {
		case spec.Name != nil:
			out[spec.Name.Name] = imp
		case pkgNames[imp] != "":
			out[pkgNames[imp]] = imp
		default:
			out[imp[strings.LastIndex(imp, "/")+1:]] = imp
		}
	}
	return out
}

// packageNames indexes declared package names by import path, so an import can
// be resolved to the identifier it is referenced by.
func packageNames(pkgs []goListPackage) map[string]string {
	out := make(map[string]string, len(pkgs))
	for _, p := range pkgs {
		out[p.ImportPath] = p.Name
	}
	return out
}

// declsAtWorkingTree returns the file's top-level declaration sources as they
// stand on disk.
func declsAtWorkingTree(root, path string) (map[string]string, error) {
	file, fset, err := parseFileAtWorkingTree(root, path)
	if err != nil {
		return nil, err
	}
	return declSources(file, fset), nil
}

// declsAtRev returns the file's top-level declaration sources at a git revision.
func declsAtRev(root, rev, path string) (map[string]string, error) {
	file, fset, err := parseFileAtRev(root, rev, path)
	if err != nil {
		return nil, err
	}
	return declSources(file, fset), nil
}

func parseFileAtWorkingTree(root, path string) (*ast.File, *token.FileSet, error) {
	src, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		return nil, nil, err
	}
	return parseSource(path, src)
}

func parseFileAtRev(root, rev, path string) (*ast.File, *token.FileSet, error) {
	out, err := gitOutputDir(root, "show", rev+":"+path)
	if err != nil {
		return nil, nil, err
	}
	return parseSource(path, []byte(out))
}

// parseSource parses without comments, so a doc-comment edit leaves every
// declaration's printed source untouched.
func parseSource(path string, src []byte) (*ast.File, *token.FileSet, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, nil, err
	}
	return file, fset, nil
}

// declSources maps each top-level declaration to its printed source. Methods are
// keyed "RecvType.Name" so a method cannot collide with a package-level function
// of the same name. Import declarations are excluded: an import without a
// corresponding symbol use does not compile, so the symbol diff already carries
// the signal.
func declSources(file *ast.File, fset *token.FileSet) map[string]string {
	out := map[string]string{}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			out[funcKey(d)] = printNode(fset, d)
		case *ast.GenDecl:
			if d.Tok == token.IMPORT {
				continue
			}
			src := printNode(fset, d)
			for _, spec := range d.Specs {
				for _, name := range specNames(spec) {
					out[name] = src
				}
			}
		}
	}
	return out
}

// funcKey names a function or method uniquely within a file.
func funcKey(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	return receiverTypeName(fn.Recv.List[0].Type) + "." + fn.Name.Name
}

// receiverTypeName unwraps pointer and generic receivers to the bare type name.
func receiverTypeName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.StarExpr:
		return receiverTypeName(e.X)
	case *ast.IndexExpr:
		return receiverTypeName(e.X)
	case *ast.IndexListExpr:
		return receiverTypeName(e.X)
	case *ast.Ident:
		return e.Name
	}
	return "?"
}

// specNames returns the names a value or type spec declares.
func specNames(spec ast.Spec) []string {
	switch s := spec.(type) {
	case *ast.TypeSpec:
		return []string{s.Name.Name}
	case *ast.ValueSpec:
		names := make([]string, 0, len(s.Names))
		for _, n := range s.Names {
			names = append(names, n.Name)
		}
		return names
	}
	return nil
}

func printNode(fset *token.FileSet, node ast.Node) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, node); err != nil {
		// An unprintable node compares unequal to everything, which resolves to
		// changeModified — the safe direction.
		return fmt.Sprintf("<unprintable %T: %v>", node, err)
	}
	return buf.String()
}

// --- changed-file discovery + classification --------------------------------

func changedFiles(root, baseRef string) (files []string, diffBase string, err error) {
	// Prefer the merge-base so we only see what this branch introduced. Fall
	// back to a plain diff against baseRef if merge-base is unavailable (e.g. a
	// shallow checkout that lacks the common ancestor).
	diffBase = baseRef
	if mb, err := gitOutputDir(root, "merge-base", baseRef, "HEAD"); err == nil {
		if mb = strings.TrimSpace(mb); mb != "" {
			diffBase = mb
		}
	}

	set := map[string]bool{}
	// Committed + working-tree changes since the base.
	if out, diffErr := gitOutputDir(root, "diff", "--name-only", diffBase); diffErr == nil {
		addLines(set, out)
	} else {
		return nil, "", diffErr
	}
	// New, not-yet-tracked files.
	if out, err := gitOutputDir(root, "ls-files", "--others", "--exclude-standard"); err == nil {
		addLines(set, out)
	}

	files = make([]string, 0, len(set))
	for f := range set {
		files = append(files, f)
	}
	sort.Strings(files)
	return files, diffBase, nil
}

func addLines(set map[string]bool, out string) {
	for _, l := range strings.Split(out, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			set[l] = true
		}
	}
}

type fileClass int

const (
	filePackage fileClass = iota // belongs to (or should map to) a package
	fileGlobal                   // forces a full run
	fileIgnore                   // irrelevant to acceptance scope
)

func classifyFile(path string) fileClass {
	base := filepath.Base(path)

	// Global triggers: dependency, tooling, and CI surface that can affect any
	// package's behaviour.
	switch {
	case base == "go.mod" || base == "go.sum":
		return fileGlobal
	case base == "GNUmakefile" || base == "Makefile":
		return fileGlobal
	case path == "main.go": // provider entrypoint
		return fileGlobal
	case strings.HasPrefix(path, ".github/workflows/"):
		return fileGlobal
	case strings.HasPrefix(path, "scripts/acctargets/"): // this tool itself
		return fileGlobal
	case strings.HasPrefix(path, "tools/"):
		return fileGlobal
	}

	// Ignore: docs, examples, and pure-prose files never gate acceptance tests.
	switch {
	case strings.HasPrefix(path, "docs/"):
		return fileIgnore
	case strings.HasPrefix(path, "examples/"):
		return fileIgnore
	case strings.HasSuffix(base, ".md"):
		return fileIgnore
	case base == "LICENSE" || base == ".gitignore" || base == ".golangci.yml":
		return fileIgnore
	}

	return filePackage
}

// --- small command helpers --------------------------------------------------

func gitOutput(args ...string) (string, error) { return cmdOutput("", "git", args...) }
func gitOutputDir(dir string, args ...string) (string, error) {
	return cmdOutput(dir, "git", args...)
}

func cmdOutput(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return string(out), err
}
