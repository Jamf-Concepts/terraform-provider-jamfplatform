// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package permissions

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
)

// SDKCallsInFile returns the distinct SDK method names the named Go source file
// calls, filtered to those present in reg. Per-construct drift-guard tests use
// it to assert that a permissions.go method list matches the calls the source
// really makes.
//
// It parses the file rather than scanning its bytes, because the regex it
// replaces cannot tell code from prose. Every such guard historically matched
// `\b\w+\.([A-Za-z0-9]+)\(` over the raw file, so a comment or a string literal
// naming a method satisfied the "is called" half of the comparison on its own: a
// stale `// formerly: r.proClient.ListPatchPoliciesV2(listCtx, nil, "")` left
// behind after the real call was deleted kept the guard green, and the construct
// went on publishing a permission its endpoints no longer need. Comments are
// not merely ignored here — parser.ParseComments is deliberately not passed, so
// they are absent from the tree and no future walk can reach one by accident.
// parser.SkipObjectResolution is passed because nothing below needs the
// identifier graph parser.ParseFile would otherwise build.
//
// A call is any *ast.CallExpr whose Fun is an *ast.SelectorExpr, which covers
// every receiver shape the provider uses without enumerating them — `client.X()`,
// `r.client.X()`, `r.proClient.X()`, `a.actions.X()`, and a package-qualified
// `pro.X()` all parse to a selector, nested or not. Registry membership is the
// only filter, exactly as in the regex form, so ordinary Go calls
// (resp.Diagnostics.Append, helpers.DerefString) are discarded and a name has to
// be an SDK method the privilege registry knows to count.
//
// Only internal/resources/pro/patch_policy has been migrated to this helper.
// The other ~53 permissions_test.go copies still carry their own clientCallRe
// and are the legacy form; they are a separate follow-up, not a pattern to copy.
func SDKCallsInFile(filename string, reg Registry) (map[string]bool, error) {
	src, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filename, err)
	}
	parsed, err := parser.ParseFile(token.NewFileSet(), filename, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", filename, err)
	}

	called := make(map[string]bool)
	ast.Inspect(parsed, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if _, known := reg[sel.Sel.Name]; known {
			called[sel.Sel.Name] = true
		}
		return true
	})
	return called, nil
}
