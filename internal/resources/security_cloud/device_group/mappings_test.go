// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strconv"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/securitycloud"
)

// TestErrorCodes pins each restated literal against the body captured during the
// wire probe. The SDK's generated ApiErrorItemCode enum is the DNS namespace's
// error schema and carries none of these, so there is no constant to key on and a
// typo would otherwise only show up as a diagnostic that silently never fires.
func TestErrorCodes(t *testing.T) {
	cases := map[string]string{
		"codeGroupAlreadyExists": codeGroupAlreadyExists,
		"codeReservedGroupName":  codeReservedGroupName,
		"codeGroupNotFound":      codeGroupNotFound,
		"codeInvalidField":       codeInvalidField,
		"codeNotEntitled":        codeNotEntitled,
		"codeBadPermissions":     codeBadPermissions,
	}
	want := map[string]string{
		"codeGroupAlreadyExists": "GROUP_ALREADY_EXISTS",
		"codeReservedGroupName":  "RESERVED_GROUP_NAME",
		"codeGroupNotFound":      "GROUP_NOT_FOUND",
		"codeInvalidField":       "INVALID_FIELD",
		"codeNotEntitled":        "NOT_ENTITLED",
		"codeBadPermissions":     "BAD_PERMISSIONS",
	}

	for name, got := range cases {
		if got != want[name] {
			t.Errorf("%s = %q, want %q", name, got, want[name])
		}
	}
}

// TestDefaultGroupName pins the reserved name the validator compares against. Get
// this wrong and the plan-time refusal stops firing, and the mistake resurfaces as
// a mid-apply 400.
func TestDefaultGroupName(t *testing.T) {
	if defaultGroupName != "Default Group" {
		t.Errorf("defaultGroupName = %q, want %q", defaultGroupName, "Default Group")
	}
}

// TestErrorCodeLiteralsAreNotInTheSDKEnum is the recurrence guard for restating a
// code the SDK already generates.
//
// This package originally wrote all six codes as string literals, including
// INVALID_FIELD and NOT_ENTITLED, which securitycloud.ApiErrorItemCode does carry —
// while every sibling package aliased the ones it could. Reviewers caught it twice
// by eye, which is exactly the job to hand to a test.
//
// The check parses this package's own const declarations rather than taking a
// hand-written list, so it cannot drift: a code added as a literal that the SDK
// covers fails here, and so does a literal that a future SDK release promotes into
// the enum. A code the enum genuinely lacks — the group-specific ones — passes,
// which is the whole point: literals are correct exactly when there is no constant
// to use.
func TestErrorCodeLiteralsAreNotInTheSDKEnum(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "mappings.go", nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parsing mappings.go: %v", err)
	}

	fromSDK := securitycloud.ApiErrorItemCodeValues()
	checked := 0

	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok || len(value.Names) == 0 || len(value.Values) == 0 {
				continue
			}
			lit, ok := value.Values[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue // an alias from the SDK, which is what we want
			}
			code, err := strconv.Unquote(lit.Value)
			if err != nil {
				t.Fatalf("unquoting %s: %v", value.Names[0].Name, err)
			}
			checked++

			if slices.Contains(fromSDK, code) {
				t.Errorf(
					"%s is written as the literal %q, but securitycloud.ApiErrorItemCode carries it — "+
						"alias the SDK constant instead so the two cannot drift",
					value.Names[0].Name, code,
				)
			}
		}
	}

	if checked == 0 {
		t.Fatal("no string-literal constants found in mappings.go — the parse found nothing to check")
	}
}
