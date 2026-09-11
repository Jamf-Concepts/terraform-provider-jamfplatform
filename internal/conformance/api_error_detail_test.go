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
// also hits it inside fmt.Sprintf, inside a tflog call, and in prose in a
// comment, and it cannot tell the second argument of AddError from the third of
// AddAttributeError. The walk below looks at exactly the diagnostic detail
// position and nothing else.

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
// with err.Error() instead of helpers.APIErrorDetail(err).
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
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			var detailIdx int
			switch sel.Sel.Name {
			case "AddError":
				detailIdx = 1
			case "AddAttributeError":
				detailIdx = 2
			default:
				return true
			}
			if len(call.Args) != detailIdx+1 {
				return true
			}

			inner, ok := call.Args[detailIdx].(*ast.CallExpr)
			if !ok || len(inner.Args) != 0 {
				return true
			}
			innerSel, ok := inner.Fun.(*ast.SelectorExpr)
			if !ok || innerSel.Sel.Name != "Error" {
				return true
			}
			if _, ok := innerSel.X.(*ast.Ident); !ok {
				return true
			}

			pos := src.Position(inner.Pos())
			offenders = append(offenders, filepath.ToSlash(path)+":"+itoa(pos.Line))
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walking the provider source: %v", err)
	}

	if len(offenders) > 0 {
		t.Errorf("%d diagnostic detail(s) render an error with .Error() instead of "+
			"helpers.APIErrorDetail(err), so an edge error page reaches the operator with no "+
			"remedy — it reads as an ordinary API failure and sends them looking for a mistake "+
			"in their configuration:\n  %s",
			len(offenders), strings.Join(offenders, "\n  "))
	}
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
