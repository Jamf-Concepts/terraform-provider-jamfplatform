// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// List resources that hydrate each item (IncludeResource / config generation)
// must give every per-item GET its own deadline. Issuing those GETs on the
// list-level context.WithTimeout budget makes the deadline cumulative across
// N sequential requests, so a high-cardinality tenant exhausts it partway
// through and the whole resource type aborts with "context deadline exceeded".
//
// This walks every list_resource.go and fails if a per-item client call is
// passed a context declared by a context.WithTimeout at the top level of
// List(). It is a structural guard: the original fix (decoupling per-item
// hydration timeouts) shipped without one, and eight list resources were
// subsequently left on the shared budget.
func TestListResources_PerItemHydrationHasOwnDeadline(t *testing.T) {
	files := findListResourceFiles(t)
	if len(files) == 0 {
		t.Fatal("found no list_resource.go files to inspect")
	}

	var hydrating int

	for _, path := range files {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}

		listFn := findMethod(file, "List")
		if listFn == nil || listFn.Body == nil {
			continue
		}

		// Contexts created by context.WithTimeout directly in List()'s body —
		// the shared, whole-operation budgets.
		listLevelCtx := map[string]bool{}
		for _, stmt := range listFn.Body.List {
			assign, ok := stmt.(*ast.AssignStmt)
			if !ok || len(assign.Rhs) != 1 || len(assign.Lhs) == 0 {
				continue
			}
			if !isCall(assign.Rhs[0], "context", "WithTimeout") {
				continue
			}
			if name, ok := assign.Lhs[0].(*ast.Ident); ok {
				listLevelCtx[name.Name] = true
			}
		}

		hydrateBlock := findIncludeResourceBlock(listFn.Body)
		if hydrateBlock == nil {
			continue
		}

		rel := relPath(path)

		// Every SDK call in the hydration block must take a per-item context.
		perItemCalls := false
		ast.Inspect(hydrateBlock, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 || !isClientCall(call) {
				return true
			}
			perItemCalls = true

			arg, ok := call.Args[0].(*ast.Ident)
			if !ok {
				return true
			}
			if listLevelCtx[arg.Name] {
				t.Errorf("%s: per-item hydration call %s is passed the list-level context %q; "+
					"derive a per-item context (context.WithTimeout(ctx, defaultItemReadTimeout)) so the "+
					"deadline is not shared across every item",
					rel, callName(call), arg.Name)
			}
			return true
		})
		if !perItemCalls {
			continue
		}
		hydrating++

		// A resource that hydrates per item and also holds a shared list-level
		// budget must carve out its own per-item deadline inside the loop.
		// Resources with no list-level timeout at all (Platform Services list
		// resources pass the request context straight through) have no shared
		// budget to exhaust, so they are exempt.
		if len(listLevelCtx) > 0 && !blockCreatesTimeout(hydrateBlock) {
			t.Errorf("%s: hydrates per item under a list-level timeout but never derives a per-item "+
				"context.WithTimeout inside the loop; the deadline is cumulative across items", rel)
		}
	}

	if hydrating == 0 {
		t.Fatal("inspected list resources but matched no per-item hydration blocks — the detector has drifted from the code")
	}
	t.Logf("checked %d list resources, %d with per-item hydration", len(files), hydrating)
}

// findListResourceFiles collects every list_resource.go under internal/resources.
func findListResourceFiles(t *testing.T) []string {
	t.Helper()
	var out []string
	root := filepath.Join("..", "resources")
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "list_resource.go" {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return out
}

// findMethod returns the named method declaration, if present.
func findMethod(file *ast.File, name string) *ast.FuncDecl {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == name && fn.Recv != nil {
			return fn
		}
	}
	return nil
}

// findIncludeResourceBlock locates the `if req.IncludeResource { ... }` body.
func findIncludeResourceBlock(body *ast.BlockStmt) *ast.BlockStmt {
	var found *ast.BlockStmt
	ast.Inspect(body, func(n ast.Node) bool {
		if found != nil {
			return false
		}
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		if sel, ok := ifStmt.Cond.(*ast.SelectorExpr); ok && sel.Sel.Name == "IncludeResource" {
			found = ifStmt.Body
			return false
		}
		return true
	})
	return found
}

// isClientCall reports whether the call is made on the resource's SDK client
// (r.client.Something(...)).
func isClientCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	inner, ok := sel.X.(*ast.SelectorExpr)
	return ok && inner.Sel.Name == "client"
}

// blockCreatesTimeout reports whether the block derives its own
// context.WithTimeout, i.e. a deadline scoped to a single item.
func blockCreatesTimeout(block *ast.BlockStmt) bool {
	found := false
	ast.Inspect(block, func(n ast.Node) bool {
		if found {
			return false
		}
		if expr, ok := n.(ast.Expr); ok && isCall(expr, "context", "WithTimeout") {
			found = true
			return false
		}
		return true
	})
	return found
}

// isCall reports whether the expression is pkg.Fn(...).
func isCall(expr ast.Expr, pkg, fn string) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != fn {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	return ok && ident.Name == pkg
}

func callName(call *ast.CallExpr) string {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		return sel.Sel.Name
	}
	return "<call>"
}

func relPath(path string) string {
	return strings.TrimPrefix(filepath.ToSlash(path), "../")
}
