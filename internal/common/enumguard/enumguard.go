// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package enumguard is the recurrence guard for restating a value the SDK
// already generates.
//
// Enum values and machine-readable error codes must come from the SDK's
// generated constants, never from a string literal — see STYLE_GUIDE.md
// §"Enum values and error codes come from the SDK, not from literals". That
// rule failed twice by review, so every package declaring such values pins it
// with a test rather than trusting a reader to notice.
//
// The check parses the package's own source rather than taking a hand-written
// list, so it cannot drift. It fires in both directions: on a literal that the
// SDK covers today, and — through [Check]'s Absent map — on a literal that a
// future SDK release promotes into an enum.
//
// It walks const *and* var declarations, including the elements of slice and
// map composite literals, because a restated enum is far more often a
// `var validFoos = []string{...}` feeding a OneOf validator than a lone const.
// It also reaches inside function bodies for the two framework call shapes that
// declare an accepted set in all but name — stringvalidator.OneOf and
// stringdefault.StaticString — since a schema that inlines its vocabulary is
// restating it just as surely as a package-level var would.
//
// It reads every non-test file in the package directory, so moving a
// declaration between files cannot slip past it.
//
// What it deliberately does not cover: a literal compared against in ordinary
// logic (`if cdnType == "AMAZON_S3"`, `case "PENDING":`). Reaching those would
// mean treating every string comparison in the package as an enum reference,
// which fires on prose and on unrelated vocabularies that happen to share a
// spelling. Those are an audit's job, not a guard's.
package enumguard

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// Literal is one string literal found in a package's const or var
// declarations, tagged with the declaration it belongs to so a failure names
// something a reader can grep for.
type Literal struct {
	// Decl is the name of the const or var the literal was declared under.
	Decl string
	// Value is the literal's unquoted content.
	Value string
	// Pos is "file:line", relative to the package directory.
	Pos string
}

// Params configures a [Check].
type Params struct {
	// Dir is the package directory to parse. Empty means ".".
	Dir string

	// Covered is every value the SDK generates for the vocabularies this
	// package writes — the concatenation of the relevant *Values() helpers.
	// A literal appearing here is a violation unless Absent excuses it.
	Covered []string

	// Absent maps a value this package must restate to the reason the SDK
	// carries no constant for it. Stating the reason per value is the point:
	// a comment claiming "the SDK has none of these" is the exact shape of
	// the defect this guard exists to catch, because it is usually true of
	// some values and false of others.
	//
	// An entry that Covered turns out to contain is itself a failure — that
	// is how the guard reports an SDK release promoting a literal into an
	// enum.
	Absent map[string]string

	// Ignore lists literal values that are not enum members at all —
	// attribute names, map keys, sentinels, prose — which happen to collide
	// with an unrelated SDK vocabulary. Each entry needs a reason for the
	// same reason Absent does.
	Ignore map[string]string
}

// Findings is what [Check] returns: violations to fail on, plus the count of
// literals actually examined so a caller can refuse a vacuous pass.
type Findings struct {
	// Restated are literals the SDK already provides a constant for.
	Restated []string
	// Promoted are Absent entries the SDK has since started generating.
	Promoted []string
	// Stale are Absent or Ignore entries no literal in the package uses.
	Stale []string
	// Examined is how many string literals were parsed.
	Examined int
}

// Check parses the package in p.Dir and reports every string literal in its
// const and var declarations that the SDK already covers.
func Check(p Params) (Findings, error) {
	dir := p.Dir
	if dir == "" {
		dir = "."
	}

	lits, err := collect(dir)
	if err != nil {
		return Findings{}, err
	}

	covered := make(map[string]struct{}, len(p.Covered))
	for _, v := range p.Covered {
		covered[v] = struct{}{}
	}

	var out Findings
	out.Examined = len(lits)
	used := map[string]struct{}{}

	for _, l := range lits {
		_, excused := p.Absent[l.Value]
		_, ignored := p.Ignore[l.Value]
		if excused || ignored {
			used[l.Value] = struct{}{}
		}
		if _, ok := covered[l.Value]; !ok || excused || ignored {
			continue
		}
		out.Restated = append(out.Restated, fmt.Sprintf(
			"%s: %s is the literal %q, which the SDK generates a constant for — "+
				"alias the constant so the two cannot drift",
			l.Pos, l.Decl, l.Value,
		))
	}

	for v, why := range p.Absent {
		if _, ok := covered[v]; ok {
			out.Promoted = append(out.Promoted, fmt.Sprintf(
				"%q is listed as absent from the SDK (%s), but the SDK now generates it — "+
					"alias the constant and drop the exemption",
				v, why,
			))
		}
	}

	for _, m := range []map[string]string{p.Absent, p.Ignore} {
		for v := range m {
			if _, ok := used[v]; !ok {
				out.Stale = append(out.Stale, fmt.Sprintf(
					"%q is exempted but no literal in the package uses it — drop the exemption", v,
				))
			}
		}
	}

	sort.Strings(out.Restated)
	sort.Strings(out.Promoted)
	sort.Strings(out.Stale)
	return out, nil
}

// collect returns every string literal declared in a const or var in the
// package's non-test files.
func collect(dir string) ([]Literal, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}

	fset := token.NewFileSet()
	var out []Literal

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", name, err)
		}
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				out = append(out, inlineSets(fset, name, fn)...)
				continue
			}
			gen, ok := decl.(*ast.GenDecl)
			if !ok || (gen.Tok != token.CONST && gen.Tok != token.VAR) {
				continue
			}
			for _, spec := range gen.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok || len(value.Names) == 0 {
					continue
				}
				declName := value.Names[0].Name
				for _, expr := range value.Values {
					for _, lit := range stringLits(expr) {
						unquoted, err := strconv.Unquote(lit.Value)
						if err != nil {
							continue
						}
						pos := fset.Position(lit.Pos())
						out = append(out, Literal{
							Decl:  declName,
							Value: unquoted,
							Pos:   fmt.Sprintf("%s:%d", name, pos.Line),
						})
					}
				}
			}
		}
	}
	return out, nil
}

// stringLits returns every string BasicLit reachable from expr without
// descending into a function body — a literal inside a helper function is not
// a declared enum value, and pulling those in would make the guard fire on
// prose.
func stringLits(expr ast.Expr) []*ast.BasicLit {
	var out []*ast.BasicLit
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind == token.STRING {
			out = append(out, e)
		}
	case *ast.CompositeLit:
		for _, elt := range e.Elts {
			if kv, ok := elt.(*ast.KeyValueExpr); ok {
				out = append(out, stringLits(kv.Key)...)
				out = append(out, stringLits(kv.Value)...)
				continue
			}
			out = append(out, stringLits(elt)...)
		}
	case *ast.ParenExpr:
		out = append(out, stringLits(e.X)...)
	}
	return out
}

// Union concatenates SDK *Values() results into the single Covered slice
// [Check] takes, dropping duplicates so a value two vocabularies share is
// reported once.
func Union(sets ...[]string) []string {
	var out []string
	for _, s := range sets {
		for _, v := range s {
			if !slices.Contains(out, v) {
				out = append(out, v)
			}
		}
	}
	return out
}

// Problems returns every failure in one slice, so a caller can report them
// without having to know which of the three kinds each one is.
func (f Findings) Problems() []string {
	out := make([]string, 0, len(f.Restated)+len(f.Promoted)+len(f.Stale))
	out = append(out, f.Restated...)
	out = append(out, f.Promoted...)
	out = append(out, f.Stale...)
	return out
}

// declaringCalls are the framework calls whose string arguments *are* an enum
// declaration: the set an attribute accepts, and the value it defaults to.
var declaringCalls = map[string]struct{}{
	"OneOf":                {},
	"OneOfCaseInsensitive": {},
	"StaticString":         {},
}

// inlineSets returns the string literals a function body passes to a
// declaringCalls call, attributed to the function so a failure points at the
// schema that inlined its vocabulary.
func inlineSets(fset *token.FileSet, file string, fn *ast.FuncDecl) []Literal {
	if fn.Body == nil {
		return nil
	}
	var out []Literal
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if _, ok := declaringCalls[sel.Sel.Name]; !ok {
			return true
		}
		for _, arg := range call.Args {
			for _, lit := range stringLits(arg) {
				unquoted, err := strconv.Unquote(lit.Value)
				if err != nil {
					continue
				}
				out = append(out, Literal{
					Decl:  fn.Name.Name + " -> " + sel.Sel.Name,
					Value: unquoted,
					Pos:   fmt.Sprintf("%s:%d", file, fset.Position(lit.Pos()).Line),
				})
			}
		}
		return true
	})
	return out
}
