// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package conformance

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// singletonPackageFloor is the number of singleton resource packages this scan
// must find before its result means anything. A detector that silently matches
// nothing reports perfect agreement, which is the one failure mode a structural
// guard cannot afford — the same reason the permissions catalogue refuses a
// parse yielding under 100 capabilities. There are 29 today; the floor sits
// below that so removing a construct does not fail the build, and far enough
// above zero that a rename of helpers.SingletonID or of ImportState does.
const singletonPackageFloor = 25

// A singleton resource must route ImportState through
// helpers.ImportSingletonState, which reads the identifier from whichever field
// carries it.
//
// A practitioner can supply that identifier two ways and only one of them fills
// req.ID: `terraform import` and an `import` block written with `id =` set it,
// while an `import` block written with `identity = { id = … }` leaves req.ID
// EMPTY and puts the value in req.Identity. The identity form is what
// `terraform plan -generate-config-out` emits and the only form Terraform's
// query-driven generation produces, and every singleton resource declares an
// IdentitySchema that advertises it works.
//
// So a bare `if req.ID != helpers.SingletonID` guard refuses the identity form
// outright, with `Got ""`, while promising the opposite. All 27 Jamf Pro
// singletons carried exactly that, and the two Jamf Security Cloud custom DNS
// singletons carried it in different wording — which is why this is a
// structural guard and not a grep: the first sweep keyed on the Pro
// diagnostic's text and missed the two that phrase theirs another way.
func TestSingletonResources_ImportAcceptsBothIdentifierForms(t *testing.T) {
	dirs := findSingletonPackages(t)
	if len(dirs) < singletonPackageFloor {
		t.Fatalf("found only %d singleton resource packages, want at least %d — helpers.SingletonID or "+
			"ImportState has probably been renamed and this scan now matches nothing",
			len(dirs), singletonPackageFloor)
	}

	for _, dir := range dirs {
		for _, fn := range importStateMethods(t, dir) {
			rel := relPath(fn.file)
			if callsSelector(fn.decl.Body, "helpers", "ImportSingletonState") {
				continue
			}

			detail := "it must call helpers.ImportSingletonState(ctx, req, resp, \"<terraform type>\")"
			if referencesRequestID(fn.decl.Body) || callsSelector(fn.decl.Body, "resource", "ImportStatePassthroughID") {
				detail = "it compares req.ID directly and/or passes through with ImportStatePassthroughID, so an " +
					"`identity = { id = \"singleton\" }` import block — the form config generation emits — is " +
					"refused with an empty identifier; call helpers.ImportSingletonState instead"
			}

			t.Errorf("%s: %s is a singleton resource but its ImportState does not delegate to the shared "+
				"helper — %s", rel, fn.receiver, detail)
		}
	}

	t.Logf("checked %d singleton resource packages", len(dirs))
}

// importStateMethod is one ImportState declaration and where it was found.
type importStateMethod struct {
	file     string
	receiver string
	decl     *ast.FuncDecl
}

// findSingletonPackages returns every directory under internal/resources holding
// non-test source that references helpers.SingletonID and declares an
// ImportState method.
//
// Membership is derived from the constant rather than from a list, so a new
// singleton is covered the day it is written. A package that stores
// helpers.SingletonID but declares no ImportState is a data source only
// (pro/jamf_pro_server_url, pro/tenant_id) and has no import path to get wrong.
func findSingletonPackages(t *testing.T) []string {
	t.Helper()

	root := filepath.Join("..", "resources")
	usesSingletonID := map[string]bool{}
	declaresImportState := map[string]bool{}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".go") || strings.HasSuffix(info.Name(), "_test.go") {
			return nil
		}
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		dir := filepath.Dir(path)
		if strings.Contains(string(src), "helpers.SingletonID") {
			usesSingletonID[dir] = true
		}
		if strings.Contains(string(src), ") ImportState(") {
			declaresImportState[dir] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}

	var out []string
	for dir := range usesSingletonID {
		if declaresImportState[dir] {
			out = append(out, dir)
		}
	}
	return out
}

// importStateMethods parses every non-test file in dir and returns each
// ImportState method it declares. The method is not always in resource.go, so
// the whole package is parsed rather than one file by name.
func importStateMethods(t *testing.T, dir string) []importStateMethod {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}

	var out []importStateMethod
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			t.Fatalf("parse %s: %v", path, parseErr)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Name.Name != "ImportState" || fn.Body == nil {
				continue
			}
			out = append(out, importStateMethod{file: path, receiver: receiverTypeName(fn), decl: fn})
		}
	}
	return out
}

// receiverTypeName renders a method's receiver type, pointer star dropped, for
// use in a diagnostic.
func receiverTypeName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return "<unknown>"
	}
	expr := fn.Recv.List[0].Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return "<unknown>"
}

// callsSelector reports whether the block calls pkg.Fn(...) anywhere.
func callsSelector(block *ast.BlockStmt, pkg, fn string) bool {
	found := false
	ast.Inspect(block, func(n ast.Node) bool {
		if found {
			return false
		}
		if expr, ok := n.(ast.Expr); ok && isCall(expr, pkg, fn) {
			found = true
			return false
		}
		return true
	})
	return found
}

// referencesRequestID reports whether the block reads req.ID, which is the
// half of the identifier an identity-form import leaves empty.
func referencesRequestID(block *ast.BlockStmt) bool {
	found := false
	ast.Inspect(block, func(n ast.Node) bool {
		if found {
			return false
		}
		sel, ok := n.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "ID" {
			return true
		}
		if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "req" {
			found = true
			return false
		}
		return true
	})
	return found
}
