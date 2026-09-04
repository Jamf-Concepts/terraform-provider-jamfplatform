// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package enumguard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// write lays out a one-file package in a temp dir and returns its path.
func write(t *testing.T, src string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mappings.go"), []byte(src), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	// A test file in the same directory must be ignored, or the guard would
	// fire on values a test deliberately pins as literals.
	if err := os.WriteFile(filepath.Join(dir, "mappings_test.go"), []byte("package p\n\nvar ignored = \"COVERED\"\n"), 0o600); err != nil {
		t.Fatalf("writing fixture test: %v", err)
	}
	return dir
}

func TestCheckFindsRestatedConst(t *testing.T) {
	dir := write(t, "package p\n\nconst codeThing = \"COVERED\"\n")

	got, err := Check(Params{Dir: dir, Covered: []string{"COVERED"}})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(got.Restated) != 1 {
		t.Fatalf("Restated = %v, want 1 finding", got.Restated)
	}
	if !strings.Contains(got.Restated[0], "codeThing") {
		t.Errorf("finding does not name the declaration: %s", got.Restated[0])
	}
	if !strings.Contains(got.Restated[0], "mappings.go:3") {
		t.Errorf("finding does not carry file:line: %s", got.Restated[0])
	}
}

// TestCheckFindsRestatedSliceElement is the case the reference test in
// device_group missed: almost every restated enum in this provider is a slice
// of literals feeding a OneOf validator, not a lone const.
func TestCheckFindsRestatedSliceElement(t *testing.T) {
	dir := write(t, "package p\n\nvar valid = []string{\"OTHER\", \"COVERED\"}\n")

	got, err := Check(Params{Dir: dir, Covered: []string{"COVERED"}})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(got.Restated) != 1 {
		t.Fatalf("Restated = %v, want 1 finding", got.Restated)
	}
	if !strings.Contains(got.Restated[0], "valid") {
		t.Errorf("finding does not name the declaration: %s", got.Restated[0])
	}
}

func TestCheckFindsRestatedMapKeyAndValue(t *testing.T) {
	dir := write(t, "package p\n\nvar toWire = map[string]string{\"KEY\": \"COVERED\"}\n")

	got, err := Check(Params{Dir: dir, Covered: []string{"COVERED", "KEY"}})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(got.Restated) != 2 {
		t.Fatalf("Restated = %v, want both the key and the value", got.Restated)
	}
}

// TestCheckIgnoresFunctionBodies keeps the guard off prose: a value named in a
// MarkdownDescription sentence or built inside a helper is not a declared enum.
func TestCheckIgnoresFunctionBodies(t *testing.T) {
	dir := write(t, "package p\n\nfunc describe() string { return \"COVERED\" }\n")

	got, err := Check(Params{Dir: dir, Covered: []string{"COVERED"}})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(got.Restated) != 0 {
		t.Errorf("Restated = %v, want none", got.Restated)
	}
}

func TestCheckPassesOnAliasedConstant(t *testing.T) {
	dir := write(t, "package p\n\nimport \"strings\"\n\nvar codeThing = strings.ToUpper(\"x\")\n")

	got, err := Check(Params{Dir: dir, Covered: []string{"COVERED"}})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(got.Problems()) != 0 {
		t.Errorf("Problems = %v, want none", got.Problems())
	}
}

// TestCheckExcusesAbsentValue is the bucket-3 case: a value the SDK genuinely
// does not carry stays a literal, and says why.
func TestCheckExcusesAbsentValue(t *testing.T) {
	dir := write(t, "package p\n\nconst codeThing = \"NOT_IN_SDK\"\n")

	got, err := Check(Params{
		Dir:     dir,
		Covered: []string{"COVERED"},
		Absent:  map[string]string{"NOT_IN_SDK": "the spec documents it in prose only"},
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(got.Problems()) != 0 {
		t.Errorf("Problems = %v, want none", got.Problems())
	}
}

// TestCheckReportsPromotedValue is the other direction, and the reason the
// guard is self-maintaining: an SDK release that starts generating a value the
// package had exempted must fail, not silently keep the literal.
func TestCheckReportsPromotedValue(t *testing.T) {
	dir := write(t, "package p\n\nconst codeThing = \"NOW_IN_SDK\"\n")

	got, err := Check(Params{
		Dir:     dir,
		Covered: []string{"NOW_IN_SDK"},
		Absent:  map[string]string{"NOW_IN_SDK": "absent when this was written"},
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(got.Promoted) != 1 {
		t.Fatalf("Promoted = %v, want 1 finding", got.Promoted)
	}
	if len(got.Restated) != 0 {
		t.Errorf("a promoted value should be reported once, as Promoted, not also as Restated: %v", got.Restated)
	}
}

// TestCheckReportsStaleExemption stops an exemption outliving the literal it
// excused, which is how an allow-list quietly becomes a place to hide things.
func TestCheckReportsStaleExemption(t *testing.T) {
	dir := write(t, "package p\n\nconst codeThing = \"PRESENT\"\n")

	got, err := Check(Params{
		Dir:     dir,
		Covered: []string{"COVERED"},
		Absent:  map[string]string{"GONE": "removed from the package years ago"},
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(got.Stale) != 1 {
		t.Fatalf("Stale = %v, want 1 finding", got.Stale)
	}
}

func TestCheckCountsWhatItExamined(t *testing.T) {
	dir := write(t, "package p\n\nvar valid = []string{\"A\", \"B\", \"C\"}\n")

	got, err := Check(Params{Dir: dir})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got.Examined != 3 {
		t.Errorf("Examined = %d, want 3", got.Examined)
	}
}

func TestUnionDeduplicates(t *testing.T) {
	got := Union([]string{"A", "B"}, []string{"B", "C"})
	want := []string{"A", "B", "C"}
	if len(got) != len(want) {
		t.Fatalf("Union = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Union = %v, want %v", got, want)
		}
	}
}

// TestCheckFindsInlineOneOf covers the shape the biggest offenders use: a
// schema function calling stringvalidator.OneOf with the vocabulary inlined.
func TestCheckFindsInlineOneOf(t *testing.T) {
	dir := write(t, `package p

func schema() any {
	return stringvalidator.OneOf("OTHER", "COVERED")
}
`)

	got, err := Check(Params{Dir: dir, Covered: []string{"COVERED"}})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(got.Restated) != 1 {
		t.Fatalf("Restated = %v, want 1 finding", got.Restated)
	}
	if !strings.Contains(got.Restated[0], "schema -> OneOf") {
		t.Errorf("finding does not name the call site: %s", got.Restated[0])
	}
}

func TestCheckFindsInlineStaticDefault(t *testing.T) {
	dir := write(t, `package p

func schema() any {
	return stringdefault.StaticString("COVERED")
}
`)

	got, err := Check(Params{Dir: dir, Covered: []string{"COVERED"}})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(got.Restated) != 1 {
		t.Fatalf("Restated = %v, want 1 finding", got.Restated)
	}
}

// TestCheckIgnoresProseInFunctionBodies is the counterpart: a value named in a
// description sentence, or compared against in logic, is not a declaration.
func TestCheckIgnoresProseInFunctionBodies(t *testing.T) {
	dir := write(t, `package p

func describe(v string) string {
	if v == "COVERED" {
		return "one of COVERED or other"
	}
	return fmt.Sprintf("saw %s", "COVERED")
}
`)

	got, err := Check(Params{Dir: dir, Covered: []string{"COVERED"}})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(got.Restated) != 0 {
		t.Errorf("Restated = %v, want none", got.Restated)
	}
}

// TestCheckReportsVacuousGuard is the meta-check. Exempting every value in
// Covered leaves a test that passes on any input, which reads as coverage while
// asserting nothing — the exact failure the mutation pass is meant to expose,
// caught structurally instead.
func TestCheckReportsVacuousGuard(t *testing.T) {
	dir := write(t, "package p\n\nconst codeThing = \"COVERED\"\n")

	got, err := Check(Params{
		Dir:     dir,
		Covered: []string{"COVERED"},
		Ignore:  map[string]string{"COVERED": "belongs to another vocabulary"},
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got.Vacuous == "" {
		t.Fatal("Vacuous is empty, want a finding")
	}
	if len(got.Problems()) != 1 {
		t.Errorf("Problems = %v, want just the vacuity finding", got.Problems())
	}
}

// TestCheckAllowsPartialExemption is the counterpart: a guard with some values
// exempted and some live is doing its job.
func TestCheckAllowsPartialExemption(t *testing.T) {
	dir := write(t, "package p\n\nconst codeThing = \"OTHER\"\n")

	got, err := Check(Params{
		Dir:     dir,
		Covered: []string{"COVERED", "OTHER"},
		Absent:  map[string]string{"OTHER": "not generated"},
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got.Vacuous != "" {
		t.Errorf("Vacuous = %q, want empty", got.Vacuous)
	}
}

// TestCheckFindsLocalDefine covers the shape the guard's own review found
// unprotected: a builder naming the one wire value it is about to send. Five
// shipped packages had no other restated enum, so missing this made their guards
// pass on mutated source.
func TestCheckFindsLocalDefine(t *testing.T) {
	dir := write(t, `package p

func printerActionToWire(action string) *string {
	switch action {
	case "Map":
		v := "COVERED"
		return &v
	}
	return nil
}
`)

	got, err := Check(Params{Dir: dir, Covered: []string{"COVERED"}})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(got.Restated) != 1 {
		t.Fatalf("Restated = %v, want 1 finding", got.Restated)
	}
	if !strings.Contains(got.Restated[0], "printerActionToWire -> v") {
		t.Errorf("finding does not name the enclosing function and variable: %s", got.Restated[0])
	}
	if !strings.Contains(got.Restated[0], "mappings.go:6") {
		t.Errorf("finding does not carry file:line: %s", got.Restated[0])
	}
}

// TestCheckPassesOnLocalDefineOfSDKConstant is the fixed form of the same site:
// the value comes from the SDK, so there is no literal to report.
func TestCheckPassesOnLocalDefineOfSDKConstant(t *testing.T) {
	dir := write(t, `package p

func printerActionToWire() *string {
	v := proclassic.PolicyPrintersPrinterItemActionInstall
	return &v
}
`)

	got, err := Check(Params{Dir: dir, Covered: []string{"COVERED"}})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(got.Problems()) != 0 {
		t.Errorf("Problems = %v, want none", got.Problems())
	}
	if got.Examined != 0 {
		t.Errorf("Examined = %d, want 0 — an aliased constant is not a literal", got.Examined)
	}
}

// TestCheckFindsLocalDefineOfComposite keeps stringLits' recursion rules on the
// right-hand side, so a slice or map of literals bound with := is covered the
// same way a package-level var is.
func TestCheckFindsLocalDefineOfComposite(t *testing.T) {
	dir := write(t, `package p

func build() []string {
	valid := []string{"OTHER", "COVERED"}
	return valid
}
`)

	got, err := Check(Params{Dir: dir, Covered: []string{"COVERED"}})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(got.Restated) != 1 {
		t.Fatalf("Restated = %v, want 1 finding", got.Restated)
	}
	if !strings.Contains(got.Restated[0], "build -> valid") {
		t.Errorf("finding does not name the enclosing function and variable: %s", got.Restated[0])
	}
}

// TestCheckStillIgnoresUsesInFunctionBodies pins the boundary the := support
// must not erode: walking one statement shape does not turn the guard into a
// walker of every string in a function. A comparison operand, a case value and a
// returned literal are uses, not declarations.
func TestCheckStillIgnoresUsesInFunctionBodies(t *testing.T) {
	dir := write(t, `package p

func classify(v string) string {
	if v == "COVERED" {
		return "COVERED"
	}
	switch v {
	case "COVERED":
		return "COVERED"
	}
	label := fmt.Sprintf("saw %s", "COVERED")
	_ = label
	out := thing{Kind: "COVERED"}
	_ = out
	return "COVERED"
}
`)

	got, err := Check(Params{Dir: dir, Covered: []string{"COVERED"}})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(got.Restated) != 0 {
		t.Errorf("Restated = %v, want none", got.Restated)
	}
	if got.Examined != 0 {
		t.Errorf("Examined = %d, want 0", got.Examined)
	}
}

// TestCheckFindsLocalDefineOfMap pins the other half of definedLits' allowance:
// a map right-hand side keeps stringLits' key-and-value recursion.
func TestCheckFindsLocalDefineOfMap(t *testing.T) {
	dir := write(t, `package p

func build() map[string]string {
	toWire := map[string]string{"KEY": "COVERED"}
	return toWire
}
`)

	got, err := Check(Params{Dir: dir, Covered: []string{"COVERED", "KEY"}})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(got.Restated) != 2 {
		t.Fatalf("Restated = %v, want both the key and the value", got.Restated)
	}
}

// TestCheckIgnoresStructLiteralInLocalDefine keeps definedLits' one deliberate
// narrowing visible: a `Field: "value"` element populates a type the package did
// not declare, so it stays an audit's problem rather than the guard's.
func TestCheckIgnoresStructLiteralInLocalDefine(t *testing.T) {
	dir := write(t, `package p

func build() any {
	body := proclassic.Criterion{AndOr: "COVERED"}
	return body
}
`)

	got, err := Check(Params{Dir: dir, Covered: []string{"COVERED"}})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(got.Restated) != 0 {
		t.Errorf("Restated = %v, want none", got.Restated)
	}
}

// TestCheckReportsEmptyCoveredAsVacuous is the other half of the vacuity check,
// and the half an SDK release can trigger on its own: withdrawing the last
// vocabulary a package restated leaves the guard called with no Covered set at
// all. That passed silently until the gate stopped skipping the empty case —
// Examined stays non-zero from unrelated literals, so nothing else fired.
func TestCheckReportsEmptyCoveredAsVacuous(t *testing.T) {
	dir := write(t, "package p\n\nconst codeThing = \"COVERED\"\n")

	got, err := Check(Params{Dir: dir})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got.Vacuous == "" {
		t.Fatal("Vacuous is empty, want a finding")
	}
	if !strings.Contains(got.Vacuous, "Covered is empty") {
		t.Errorf("Vacuous = %q, want it to name the empty Covered set", got.Vacuous)
	}
	if len(got.Problems()) != 1 {
		t.Errorf("Problems = %v, want just the vacuity finding", got.Problems())
	}
	if got.Examined == 0 {
		t.Error("Examined = 0, want the literal to have been parsed — the Examined check cannot substitute")
	}
}

// TestCheckReportsEmptyCoveredEvenWithExemptions pins that an Absent entry
// cannot rescue an empty guard. The Promoted arm needs the value to appear in
// Covered, so with nothing covered it can never fire; the entry is merely stale.
func TestCheckReportsEmptyCoveredEvenWithExemptions(t *testing.T) {
	dir := write(t, "package p\n\nconst codeThing = \"COVERED\"\n")

	got, err := Check(Params{
		Dir:    dir,
		Absent: map[string]string{"COVERED": "the SDK withdrew the enum"},
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got.Vacuous == "" {
		t.Fatal("Vacuous is empty, want a finding")
	}
	if len(got.Promoted) != 0 {
		t.Errorf("Promoted = %v, want none — nothing is covered", got.Promoted)
	}
}
