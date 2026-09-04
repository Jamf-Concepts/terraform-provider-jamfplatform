//go:build acctargets

// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The two narrowing rules decide whether a hub-package edit re-tests one package
// or all 118, so they are worth pinning against real git history rather than
// hand-built ASTs — the failure mode that matters is a rule that says "additive"
// about a modification.

// newRepo builds a throwaway git repository with one commit, and returns its root
// along with the revision to diff against.
func newRepo(t *testing.T, files map[string]string) (root, base string) {
	t.Helper()
	root = t.TempDir()

	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return string(out)
	}

	run("init", "-q")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	writeFiles(t, root, files)
	run("add", "-A")
	run("commit", "-q", "-m", "base")

	rev := run("rev-parse", "HEAD")
	return root, trimNewline(rev)
}

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

func trimNewline(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

const baseHub = `package hub

// Existing is referenced by other packages.
func Existing() string { return "a" }

var Table = map[string]string{"k": "v"}

type Thing struct{ N int }

func (t Thing) Method() int { return t.N }
`

func TestClassifyGoFileChange(t *testing.T) {
	cases := []struct {
		name string
		head string
		want changeKind
	}{
		{
			name: "unchanged",
			head: baseHub,
			want: changeNoop,
		},
		{
			name: "doc comment edit is a no-op",
			head: `package hub

// Existing does a thing. Reworded entirely.
func Existing() string { return "a" }

var Table = map[string]string{"k": "v"}

type Thing struct{ N int }

func (t Thing) Method() int { return t.N }
`,
			want: changeNoop,
		},
		{
			name: "added function is additive",
			head: baseHub + `
func Added() string { return "b" }
`,
			want: changeAdditive,
		},
		{
			name: "added var is additive",
			head: baseHub + `
var Extra = 1
`,
			want: changeAdditive,
		},
		{
			name: "changed function body is a modification",
			head: `package hub

func Existing() string { return "CHANGED" }

var Table = map[string]string{"k": "v"}

type Thing struct{ N int }

func (t Thing) Method() int { return t.N }
`,
			want: changeModified,
		},
		{
			name: "changed var value is a modification",
			head: `package hub

func Existing() string { return "a" }

var Table = map[string]string{"k": "DIFFERENT"}

type Thing struct{ N int }

func (t Thing) Method() int { return t.N }
`,
			want: changeModified,
		},
		{
			name: "changed method is a modification",
			head: `package hub

func Existing() string { return "a" }

var Table = map[string]string{"k": "v"}

type Thing struct{ N int }

func (t Thing) Method() int { return t.N * 2 }
`,
			want: changeModified,
		},
		{
			name: "removed declaration is a modification",
			head: `package hub

var Table = map[string]string{"k": "v"}

type Thing struct{ N int }

func (t Thing) Method() int { return t.N }
`,
			want: changeModified,
		},
		{
			name: "adding one declaration while changing another is a modification",
			head: `package hub

func Existing() string { return "CHANGED" }

var Table = map[string]string{"k": "v"}

type Thing struct{ N int }

func (t Thing) Method() int { return t.N }

func Added() string { return "b" }
`,
			want: changeModified,
		},
		{
			name: "import-only edit is a no-op",
			head: `package hub

import "strings"

func Existing() string { return "a" }

var Table = map[string]string{"k": "v"}

type Thing struct{ N int }

func (t Thing) Method() int { return t.N }

var _ = strings.TrimSpace
`,
			// The new blank var counts as an addition, so this is additive
			// rather than a no-op — still not a fan-out.
			want: changeAdditive,
		},
		{
			name: "unparseable head is a modification",
			head: "package hub\n\nfunc Existing() string { return \n",
			want: changeModified,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, base := newRepo(t, map[string]string{"hub/hub.go": baseHub})
			writeFiles(t, root, map[string]string{"hub/hub.go": tc.head})

			if got := classifyGoFileChange(root, base, "hub/hub.go"); got != tc.want {
				t.Errorf("classifyGoFileChange = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestClassifyGoFileChange_NewFile covers the brand-new-file path, which cannot
// be expressed by rewriting an existing one.
func TestClassifyGoFileChange_NewFile(t *testing.T) {
	root, base := newRepo(t, map[string]string{"hub/hub.go": baseHub})
	writeFiles(t, root, map[string]string{"hub/extra.go": "package hub\n\nfunc Fresh() {}\n"})

	if got := classifyGoFileChange(root, base, "hub/extra.go"); got != changeAdditive {
		t.Errorf("a new file must be additive, got %v", got)
	}
}

// TestClassifyGoFileChange_NonGo keeps non-Go files on the wide path: this tool
// cannot reason about a testdata fixture or a config file.
func TestClassifyGoFileChange_NonGo(t *testing.T) {
	root, base := newRepo(t, map[string]string{"hub/hub.go": baseHub, "hub/data.json": "{}"})
	writeFiles(t, root, map[string]string{"hub/data.json": `{"a":1}`})

	if got := classifyGoFileChange(root, base, "hub/data.json"); got != changeModified {
		t.Errorf("a non-Go file must fan out, got %v", got)
	}
}

const module = "example.com/prov"

// pkgNames stands in for the go list index the real run builds.
var pkgNames = map[string]string{
	module + "/internal/resources/pro/category": "category",
	module + "/internal/resources/pro/script":   "script",
	module + "/internal/actions/device":         "deviceactions",
}

// providerFile renders a provider.go whose registration lists hold the given
// constructor references.
func providerFile(imports []string, resources []string, extra string) string {
	var out strings.Builder
	out.WriteString("package provider\n\nimport (\n")
	for _, imp := range imports {
		out.WriteString("\t\"" + imp + "\"\n")
	}
	out.WriteString(")\n\ntype P struct{}\n\nfunc (p *P) Resources() []func() any {\n\treturn []func() any{\n")
	for _, r := range resources {
		out.WriteString("\t\t" + r + ",\n")
	}
	out.WriteString("\t}\n}\n" + extra)
	return out.String()
}

func TestRegistrationDelta(t *testing.T) {
	catImp := module + "/internal/resources/pro/category"
	scriptImp := module + "/internal/resources/pro/script"

	base := providerFile(
		[]string{catImp},
		[]string{"category.NewCategoryResource"},
		"",
	)

	// A base that also registers a list resource, so a case can drop just that
	// one constructor while the package stays registered.
	baseWithList := providerFile(
		[]string{catImp},
		[]string{"category.NewCategoryResource"},
		`
func (p *P) ListResources() []func() any {
	return []func() any{
		category.NewCategoryListResource,
	}
}
`,
	)

	cases := []struct {
		name         string
		base         string // defaults to base
		head         string
		wantConfined bool
		wantTargets  []string
	}{
		{
			name: "added registration selects only the new package",
			head: providerFile(
				[]string{catImp, scriptImp},
				[]string{"category.NewCategoryResource", "script.NewScriptResource"},
				"",
			),
			wantConfined: true,
			wantTargets:  []string{scriptImp},
		},
		{
			name: "removed registration selects the dropped package",
			head: providerFile(
				[]string{catImp, scriptImp},
				[]string{"script.NewScriptResource"},
				"",
			),
			wantConfined: true,
			wantTargets:  []string{catImp, scriptImp},
		},
		{
			name: "dropping one constructor from a still-registered package selects it",
			base: baseWithList,
			head: providerFile(
				[]string{catImp},
				[]string{"category.NewCategoryResource"},
				`
func (p *P) ListResources() []func() any {
	return []func() any{}
}
`,
			),
			wantConfined: true,
			wantTargets:  []string{catImp},
		},
		{
			name: "reordering selects nothing",
			head: providerFile(
				[]string{catImp},
				[]string{"category.NewCategoryResource"},
				"",
			),
			wantConfined: true,
			wantTargets:  nil,
		},
		{
			name: "an edit outside the registration methods is not confined",
			head: providerFile(
				[]string{catImp},
				[]string{"category.NewCategoryResource"},
				"\nfunc (p *P) Configure() { panic(\"changed\") }\n",
			),
			wantConfined: false,
		},
		{
			name: "a conditional in the list is not confined",
			head: `package provider

import (
	"` + catImp + `"
)

type P struct{}

func (p *P) Resources() []func() any {
	if true {
		return nil
	}
	return []func() any{
		category.NewCategoryResource,
	}
}
`,
			wantConfined: false,
		},
		{
			name: "a helper call in the list is not confined",
			head: `package provider

import (
	"` + catImp + `"
)

type P struct{}

func (p *P) Resources() []func() any {
	return append([]func() any{category.NewCategoryResource}, nil...)
}
`,
			wantConfined: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			caseBase := tc.base
			if caseBase == "" {
				caseBase = base
			}
			root, baseRev := newRepo(t, map[string]string{"internal/provider/provider.go": caseBase})
			writeFiles(t, root, map[string]string{"internal/provider/provider.go": tc.head})

			targets, confined := registrationDelta(root, baseRev, "internal/provider/provider.go", module, pkgNames)
			if confined != tc.wantConfined {
				t.Fatalf("confined = %v, want %v (targets %v)", confined, tc.wantConfined, targets)
			}
			if !confined {
				return
			}
			if len(targets) != len(tc.wantTargets) {
				t.Fatalf("targets = %v, want %v", targets, tc.wantTargets)
			}
			for _, want := range tc.wantTargets {
				if !targets[want] {
					t.Errorf("targets %v missing %s", targets, want)
				}
			}
		})
	}
}

// TestRegistrationDelta_PackageNameDiffersFromDir pins the alias resolution
// against the case that would otherwise silently mis-attribute: internal/actions/device
// declares `package deviceactions`, so the last path segment is not the
// identifier the import is referenced by.
func TestRegistrationDelta_PackageNameDiffersFromDir(t *testing.T) {
	catImp := module + "/internal/resources/pro/category"
	actImp := module + "/internal/actions/device"

	base := providerFile([]string{catImp}, []string{"category.NewCategoryResource"}, "")
	head := providerFile(
		[]string{catImp, actImp},
		[]string{"category.NewCategoryResource", "deviceactions.NewEraseAction"},
		"",
	)

	root, baseRev := newRepo(t, map[string]string{"internal/provider/provider.go": base})
	writeFiles(t, root, map[string]string{"internal/provider/provider.go": head})

	targets, confined := registrationDelta(root, baseRev, "internal/provider/provider.go", module, pkgNames)
	if !confined {
		t.Fatal("a registration-only change must be confined")
	}
	if !targets[actImp] {
		t.Errorf("targets %v missing %s; package name did not resolve", targets, actImp)
	}
}

// TestRegistrationDelta_UnknownImportIsNotConfined guards the fail-safe: a
// constructor from a package the index does not know about could be anything, so
// the file must fall back to fanning out.
func TestRegistrationDelta_UnknownImportIsNotConfined(t *testing.T) {
	catImp := module + "/internal/resources/pro/category"

	base := providerFile([]string{catImp}, []string{"category.NewCategoryResource"}, "")
	head := providerFile(
		[]string{catImp, "github.com/elsewhere/thing"},
		[]string{"category.NewCategoryResource", "thing.New"},
		"",
	)

	root, baseRev := newRepo(t, map[string]string{"internal/provider/provider.go": base})
	writeFiles(t, root, map[string]string{"internal/provider/provider.go": head})

	if _, confined := registrationDelta(root, baseRev, "internal/provider/provider.go", module, pkgNames); confined {
		t.Error("a constructor from outside the module must not be treated as confined")
	}
}

func TestFuncKeyDistinguishesMethodsFromFunctions(t *testing.T) {
	src := `package hub

func Name() {}

type A struct{}

func (a *A) Name() {}

type B struct{}

func (b B) Name() {}
`
	file, fset, err := parseSource("hub.go", []byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	decls := declSources(file, fset)

	for _, key := range []string{"Name", "A.Name", "B.Name"} {
		if _, ok := decls[key]; !ok {
			t.Errorf("declaration key %q missing from %v", key, keysOf(decls))
		}
	}
}

func keysOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestIsRegistrationDecl(t *testing.T) {
	for key, want := range map[string]bool{
		"JamfPlatformProvider.Resources":   true,
		"JamfPlatformProvider.DataSources": true,
		"JamfPlatformProvider.Configure":   false,
		"Resources":                        true,
		"someHelper":                       false,
	} {
		if got := isRegistrationDecl(key); got != want {
			t.Errorf("isRegistrationDecl(%q) = %v, want %v", key, got, want)
		}
	}
}
