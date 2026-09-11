// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package conformance

// Guards the uniformity of helpers.APIErrorDetail across every construct.
//
// This file carries NO acceptance build tag on purpose: it needs no credentials
// and no network, and it must run in `make test`. The whole point is to catch a
// regressed call site at PR time.
//
// # Why uniformity is the property, not coverage of "likely" sites
//
// An edge error page — a CDN, WAF or IP allowlist answering instead of Jamf —
// can arrive on any call. Which one it lands on is a property of the network at
// that moment, not of the resource, so there is no subset of call sites where
// the guidance belongs and no subset where it does not. The two CloudFront
// failures that prompted this landed on creates (CI runs 34202213638 and
// 34128074623), the operations the obvious "reads and deletes only" narrowing
// would have skipped.
//
// APIErrorDetail is a no-op for an error a Jamf service produced, which is
// almost all of them, so applying it everywhere costs nothing at runtime and
// removes the judgment call from every future construct.
//
// # Why an AST walk and not a grep
//
// The detail argument has to be matched structurally: a grep for `err.Error()`
// also hits it inside a tflog call, inside a map value and in prose in a
// comment, and it cannot tell the second argument of AddError from the third of
// AddAttributeError. The walk below looks at exactly the diagnostic detail
// position and nothing else.
//
// # Why every rendering shape, not just a bare .Error() call
//
// The first version of this walk recognised one shape — a bare `err.Error()`
// standing alone in the detail — and reported a clean tree while roughly 180
// details still rendered raw, because a detail is written four ways here and
// three of them slipped past. `diag.NewErrorDiagnostic(summary, err.Error())`
// is the standard list-resource shape; `fmt.Sprintf("API error: %v", err)` is
// the standard classic-Delete shape; and `"Could not delete X: "+err.Error()`
// is what most CRUD prose looks like. All three reached the operator with the
// condensed one-line edge page and no remedy, and `internal/resources/pro/
// category` — the STYLE_GUIDE's own reference implementation — held two of
// them, so a construct copied from it inherited the gap by default.
//
// So the detail expression is walked rather than pattern-matched: a `.Error()`
// call anywhere in it is flagged, and so is an error-shaped identifier handed
// to fmt.Sprintf, through any amount of concatenation. Recursion deliberately
// does not enter the arguments of calls other than fmt.Sprintf, which is what
// keeps `helpers.APIErrorDetail(err)` itself from reading as an offender.
//
// The `.Error()` rule matches any receiver, including a selector such as
// `resp.Err.Error()`, which no call site uses today: the rule is written for
// the shape a future one could take, not only for the shapes present.
//
// # Why a locally produced error goes through it too
//
// A handful of the details caught this way render an error no Jamf service
// produced: a base64 or JSON decode of an operator's own attribute, a semantic
// version that would not parse, a state upgrader re-encoding prior state. They
// are converted rather than exempted, because nothing structural separates them
// from an API error — both are an `err` in a detail — so the only exemption
// available would be a per-site marker, and a guard with an escape hatch is the
// guard this one replaced. The conversion is free: APIErrorDetail appends
// nothing to an error the SDK never marked, so those details render exactly as
// before, byte for byte.
//
// What this does NOT enforce: a warning diagnostic. AddWarning and
// NewWarningDiagnostic are left alone because most of them render a decode or
// plan-modifier failure rather than an API error, and the handful that do
// report an API call are converted by hand.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// providerRoot is the repository's Go source root, relative to this package.
const providerRoot = "../.."

// TestNoBareErrorInDiagnosticDetail fails when a diagnostic renders an error
// directly instead of through helpers.APIErrorDetail(err).
//
// Test files are excluded: an acceptance test asserting on an exact message is
// entitled to the raw text, and a unit test building a diagnostic is not a call
// site an operator ever sees.
func TestNoBareErrorInDiagnosticDetail(t *testing.T) {
	t.Parallel()

	var offenders []string

	err := filepath.WalkDir(providerRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "docs", "examples", "local-testing", "spike", "templates":
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		src := token.NewFileSet()
		f, perr := parser.ParseFile(src, path, nil, parser.SkipObjectResolution)
		if perr != nil {
			return perr
		}

		ast.Inspect(f, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			detail, ok := diagnosticDetailArg(call)
			if !ok {
				return true
			}
			if !rendersErrorRaw(detail) {
				return true
			}

			pos := src.Position(detail.Pos())
			offenders = append(offenders, filepath.ToSlash(path)+":"+itoa(pos.Line))
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walking the provider source: %v", err)
	}

	if len(offenders) > 0 {
		t.Errorf("%d diagnostic detail(s) render an error directly instead of through "+
			"helpers.APIErrorDetail(err), so an edge error page reaches the operator with no "+
			"remedy — it reads as an ordinary API failure and sends them looking for a mistake "+
			"in their configuration:\n  %s",
			len(offenders), strings.Join(offenders, "\n  "))
	}
}

// TestNoBareErrorInDiagnosticDetail_Fixtures pins the shapes the walk above
// recognises, so a later narrowing of it cannot silently re-open one of the
// three blind spots the guard was widened to cover.
//
// It asserts on the detector rather than on a fixture file, because a fixture
// file holding a real offender would have to live outside the walk's own tree
// to avoid failing the guard it is testing.
func TestNoBareErrorInDiagnosticDetail_Fixtures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call string
		want bool
	}{
		{
			name: "bare Error call",
			call: `resp.Diagnostics.AddError("Error reading category", err.Error())`,
			want: true,
		},
		{
			name: "selector receiver Error call",
			call: `resp.Diagnostics.AddError("Error reading category", out.Err.Error())`,
			want: true,
		},
		{
			name: "list resource diagnostic",
			call: `diag.NewErrorDiagnostic("Error listing categories", err.Error())`,
			want: true,
		},
		{
			name: "classic delete Sprintf",
			call: `resp.Diagnostics.AddError("Error deleting category", fmt.Sprintf("API error: %v", err))`,
			want: true,
		},
		{
			name: "Sprintf over a suffixed error name",
			call: `resp.Diagnostics.AddError("Error deleting category", fmt.Sprintf("API error: %v", deleteErr))`,
			want: true,
		},
		{
			name: "prose concatenation",
			call: `resp.Diagnostics.AddError("Error creating blueprint", "Could not create blueprint: "+err.Error())`,
			want: true,
		},
		{
			name: "attribute error",
			call: `resp.Diagnostics.AddAttributeError(path.Root("name"), "Invalid name", err.Error())`,
			want: true,
		},
		{
			name: "attribute error diagnostic",
			call: `diag.NewAttributeErrorDiagnostic(path.Root("name"), "Invalid name", err.Error())`,
			want: true,
		},
		{
			name: "converted",
			call: `resp.Diagnostics.AddError("Error reading category", helpers.APIErrorDetail(err))`,
			want: false,
		},
		{
			name: "converted with prose",
			call: `resp.Diagnostics.AddError("Error creating blueprint", "Could not create blueprint: "+helpers.APIErrorDetail(err))`,
			want: false,
		},
		{
			name: "converted through Sprintf",
			call: `resp.Diagnostics.AddError("Error deleting category", fmt.Sprintf("Category %s: %s", id, helpers.APIErrorDetail(err)))`,
			want: false,
		},
		{
			name: "no error at all",
			call: `resp.Diagnostics.AddError("Missing ID", "Cannot delete category without ID.")`,
			want: false,
		},
		{
			name: "non-error Sprintf operand",
			call: `resp.Diagnostics.AddError("Unsupported type", fmt.Sprintf("Got %T.", req.ProviderData))`,
			want: false,
		},
		{
			name: "warning is out of scope",
			call: `resp.Diagnostics.AddWarning("Could not decode plan", err.Error())`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expr, perr := parser.ParseExpr(tt.call)
			if perr != nil {
				t.Fatalf("parsing the fixture: %v", perr)
			}
			call, ok := expr.(*ast.CallExpr)
			if !ok {
				t.Fatalf("fixture is not a call expression: %s", tt.call)
			}

			var got bool
			if detail, ok := diagnosticDetailArg(call); ok {
				got = rendersErrorRaw(detail)
			}
			if got != tt.want {
				t.Errorf("flagged = %t, want %t for %s", got, tt.want, tt.call)
			}
		})
	}
}

// diagnosticDetailArg returns the expression occupying the detail position of
// call, when call is one of the four ways this provider raises an error
// diagnostic.
//
// The argument count is pinned exactly, so a helper of the same name taking a
// different shape is passed over rather than mis-indexed.
func diagnosticDetailArg(call *ast.CallExpr) (ast.Expr, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, false
	}

	var detailIdx int
	switch sel.Sel.Name {
	case "AddError", "NewErrorDiagnostic":
		detailIdx = 1
	case "AddAttributeError", "NewAttributeErrorDiagnostic":
		detailIdx = 2
	default:
		return nil, false
	}
	if len(call.Args) != detailIdx+1 {
		return nil, false
	}
	return call.Args[detailIdx], true
}

// rendersErrorRaw reports whether expr puts an error's own text into a
// diagnostic detail without passing it through helpers.APIErrorDetail.
//
// Concatenation is descended into, because surrounding prose is the norm and
// says nothing about whether the error itself is rendered. fmt.Sprintf is
// descended into for its operands only, so an error handed to a `%v` or `%s`
// verb is caught; the format string itself is a BasicLit and matches nothing.
// No other call's arguments are examined, which is what keeps a correct
// `helpers.APIErrorDetail(err)` — and a `%s` operand already wrapping it — from
// being reported.
func rendersErrorRaw(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.ParenExpr:
		return rendersErrorRaw(e.X)
	case *ast.BinaryExpr:
		if e.Op != token.ADD {
			return false
		}
		return rendersErrorRaw(e.X) || rendersErrorRaw(e.Y)
	case *ast.CallExpr:
		sel, ok := e.Fun.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		if sel.Sel.Name == "Error" && len(e.Args) == 0 {
			return true
		}
		if !isSprintf(sel) {
			return false
		}
		for _, arg := range e.Args[1:] {
			if isErrorShaped(arg) || rendersErrorRaw(arg) {
				return true
			}
		}
	}
	return false
}

// isSprintf reports whether sel names fmt.Sprintf.
func isSprintf(sel *ast.SelectorExpr) bool {
	if sel.Sel.Name != "Sprintf" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "fmt"
}

// isErrorShaped reports whether expr names something that is almost certainly
// an error value, by its identifier alone.
//
// A name test rather than a type test, because the walk parses without type
// information: loading the whole module through go/packages to type-check one
// argument costs more than the precision buys. The suffix rule is deliberately
// narrow — `err`, `perr`, `apiErr`, `resp.Err` — so a detail rendering a
// non-error value through fmt.Sprintf is not dragged in. The cost of the
// narrowing is that an error held in a differently named variable is missed,
// which no call site does today; the alternative, flagging every Sprintf
// operand, makes the guard unusable.
func isErrorShaped(expr ast.Expr) bool {
	var name string
	switch e := expr.(type) {
	case *ast.Ident:
		name = e.Name
	case *ast.SelectorExpr:
		name = e.Sel.Name
	default:
		return false
	}
	return name == "err" || name == "Err" ||
		strings.HasSuffix(name, "err") || strings.HasSuffix(name, "Err")
}

// itoa renders a line number without pulling strconv in for one call.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
