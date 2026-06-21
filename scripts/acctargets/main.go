//go:build ignore

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
// stays (so changing the provider itself still fans out), but the edges
// provider -> {every resource} are dropped. Real edges survive — a package's
// own deps, shared helpers (internal/common/*, providerdata), and any
// cross-package test fixtures — so changing internal/common/scope still
// re-tests exactly the scope-bearing resources, no more, no less.
//
// Output on stdout is one of:
//   - "./..."               full suite (a global file such as go.mod changed)
//   - "<imp> <imp> ..."     explicit space-separated package import paths
//   - ""                    nothing affected (skip the acceptance run)
//
// Usage:
//
//	go run scripts/acctargets/main.go [baseRef]   # baseRef default: origin/main
//	BASE_REF=origin/main go run scripts/acctargets/main.go
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// goListPackage is the subset of `go list -json` fields we consume.
type goListPackage struct {
	ImportPath   string
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

	changedFiles, err := changedFiles(root, baseRef)
	if err != nil {
		return fmt.Errorf("computing changed files: %w", err)
	}

	// Map every changed file to a changed package import path. A file that is
	// neither ignorable (docs/examples/markdown) nor mappable to a package, nor
	// a recognised global trigger, forces a full run — the safe default.
	changed := map[string]bool{}
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
			if imp := dirIndex.lookup(filepath.Dir(abs)); imp != "" {
				changed[imp] = true
				continue
			}
			// A changed Go-ish file we cannot attribute to a package (new dir,
			// deleted package, testdata without a package). Be conservative.
			fmt.Println("./...")
			return nil
		}
	}

	if len(changed) == 0 {
		// Nothing code-relevant changed.
		return nil
	}

	// Reverse-dependency closure over the cut graph.
	providerPkg := module + "/internal/provider"
	rev := buildReverseGraph(pkgs, module, providerPkg)
	affected := reverseClosure(rev, changed)

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
	defer f.Close()
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

// --- changed-file discovery + classification --------------------------------

func changedFiles(root, baseRef string) ([]string, error) {
	// Prefer the merge-base so we only see what this branch introduced. Fall
	// back to a plain diff against baseRef if merge-base is unavailable (e.g. a
	// shallow checkout that lacks the common ancestor).
	diffBase := baseRef
	if mb, err := gitOutputDir(root, "merge-base", baseRef, "HEAD"); err == nil {
		if mb = strings.TrimSpace(mb); mb != "" {
			diffBase = mb
		}
	}

	set := map[string]bool{}
	// Committed + working-tree changes since the base.
	if out, err := gitOutputDir(root, "diff", "--name-only", diffBase); err == nil {
		addLines(set, out)
	} else {
		return nil, err
	}
	// New, not-yet-tracked files.
	if out, err := gitOutputDir(root, "ls-files", "--others", "--exclude-standard"); err == nil {
		addLines(set, out)
	}

	files := make([]string, 0, len(set))
	for f := range set {
		files = append(files, f)
	}
	sort.Strings(files)
	return files, nil
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
