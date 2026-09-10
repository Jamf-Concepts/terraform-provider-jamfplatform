// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

// specWith wraps a `payloadkeys` block in the minimum schema file the parsers accept, so a test case
// is the key list under test and nothing else.
func specWith(payloadKeys string) string {
	return "title: Test\npayload:\n  payloadtype: com.example.test\npayloadkeys:\n" + payloadKeys
}

// declarationWith wraps a `payloadkeys` block in the minimum declaration schema file, which differs
// from a profile only in naming the type under `declarationtype`.
func declarationWith(payloadKeys string) string {
	return "title: Test\npayload:\n  declarationtype: com.apple.configuration.test\npayloadkeys:\n" + payloadKeys
}

// captureWarnings redirects the generator's audit output for the duration of a test, so a case can
// assert what a regeneration would have reported without the suite printing it.
func captureWarnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	captured := &bytes.Buffer{}
	previous := warnOutput
	warnOutput = captured
	t.Cleanup(func() { warnOutput = previous })
	return captured
}

// unionBranches folds two synthetic branch checkouts in read order, release first, exactly as
// mergeProfiles does for one payload type.
func unionBranches(t *testing.T, releaseSpec, seedSpec string) map[string]*schema {
	t.Helper()
	merged := make(map[string]*schema)
	for _, branch := range []struct{ ref, raw string }{{"release", releaseSpec}, {"seed", seedSpec}} {
		spec, err := parseSpec([]byte(branch.raw))
		if err != nil {
			t.Fatalf("parseSpec(%s): %v", branch.ref, err)
		}
		for name, keySchema := range dictionaryKeys(spec.PayloadKeys, 1) {
			merged[name] = unionSchema(merged[name], keySchema, branch.ref, "com.example.test."+name)
		}
	}
	return merged
}

// TestUnionWidensADisagreementBetweenBranches covers the rule the union exists for: a constraint
// holds only where every branch declaring the key agrees, so a tightening on the newer branch cannot
// reject a configuration the older branch accepted.
func TestUnionWidensADisagreementBetweenBranches(t *testing.T) {
	tests := []struct {
		name    string
		release string
		seed    string
		key     string
		assert  func(t *testing.T, got *schema)
	}{
		{
			name:    "a type disagreement widens to any",
			release: "- key: Setting\n  type: <string>\n",
			seed:    "- key: Setting\n  type: <dictionary>\n",
			key:     "Setting",
			assert: func(t *testing.T, got *schema) {
				if got.Type != typeAny {
					t.Errorf("type = %q, want %q", got.Type, typeAny)
				}
			},
		},
		{
			name:    "required on the newer branch alone emerges optional",
			release: "- key: Setting\n  type: <string>\n  presence: optional\n",
			seed:    "- key: Setting\n  type: <string>\n  presence: required\n",
			key:     "Setting",
			assert: func(t *testing.T, got *schema) {
				if got.Required {
					t.Error("required = true, want false: a key optional on release must stay optional")
				}
			},
		},
		{
			name:    "required on the older branch alone emerges optional",
			release: "- key: Setting\n  type: <string>\n  presence: required\n",
			seed:    "- key: Setting\n  type: <string>\n  presence: optional\n",
			key:     "Setting",
			assert: func(t *testing.T, got *schema) {
				if got.Required {
					t.Error("required = true, want false: Apple demotes a key to optional when an alternative arrives beside it")
				}
			},
		},
		{
			name:    "a narrowed enum emerges as the union of both branches",
			release: "- key: Setting\n  type: <string>\n  rangelist:\n  - Allow\n  - Deny\n",
			seed:    "- key: Setting\n  type: <string>\n  rangelist:\n  - Deny\n  - Ask\n",
			key:     "Setting",
			assert: func(t *testing.T, got *schema) {
				want := []any{"Allow", "Deny", "Ask"}
				if fmt.Sprint(got.Enum) != fmt.Sprint(want) {
					t.Errorf("enum = %v, want %v", got.Enum, want)
				}
			},
		},
		{
			name:    "an enum one branch does not declare emerges unconstrained",
			release: "- key: Setting\n  type: <string>\n",
			seed:    "- key: Setting\n  type: <string>\n  rangelist:\n  - Deny\n",
			key:     "Setting",
			assert: func(t *testing.T, got *schema) {
				if got.Enum != nil {
					t.Errorf("enum = %v, want nil: release never restricted this key", got.Enum)
				}
			},
		},
		{
			name:    "a narrowed range widens to the loosest bounds",
			release: "- key: Setting\n  type: <integer>\n  range:\n    min: 0\n    max: 100\n",
			seed:    "- key: Setting\n  type: <integer>\n  range:\n    min: 10\n    max: 60\n",
			key:     "Setting",
			assert: func(t *testing.T, got *schema) {
				if got.Min == nil || *got.Min != 0 {
					t.Errorf("min = %v, want 0", formatBound(got.Min))
				}
				if got.Max == nil || *got.Max != 100 {
					t.Errorf("max = %v, want 100", formatBound(got.Max))
				}
			},
		},
		{
			name:    "a bound one branch leaves off emerges unbounded",
			release: "- key: Setting\n  type: <integer>\n  range:\n    min: 0\n",
			seed:    "- key: Setting\n  type: <integer>\n  range:\n    min: 10\n    max: 60\n",
			key:     "Setting",
			assert: func(t *testing.T, got *schema) {
				if got.Max != nil {
					t.Errorf("max = %v, want unbounded", formatBound(got.Max))
				}
				if got.Min == nil || *got.Min != 0 {
					t.Errorf("min = %v, want 0", formatBound(got.Min))
				}
			},
		},
		{
			name:    "a nested key's constraints widen the same way",
			release: "- key: Setting\n  type: <dictionary>\n  subkeys:\n  - key: Inner\n    type: <string>\n    presence: required\n",
			seed:    "- key: Setting\n  type: <dictionary>\n  subkeys:\n  - key: Inner\n    type: <integer>\n    presence: optional\n",
			key:     "Setting",
			assert: func(t *testing.T, got *schema) {
				inner := got.Keys["Inner"]
				if inner == nil {
					t.Fatal("Inner is absent from the union")
				}
				if inner.Type != typeAny {
					t.Errorf("Inner type = %q, want %q", inner.Type, typeAny)
				}
				if inner.Required {
					t.Error("Inner required = true, want false")
				}
			},
		},
		{
			name:    "an array element's constraints widen the same way",
			release: "- key: Setting\n  type: <array>\n  subkeys:\n  - key: Item\n    type: <string>\n    rangelist:\n    - Allow\n",
			seed:    "- key: Setting\n  type: <array>\n  subkeys:\n  - key: Item\n    type: <string>\n    rangelist:\n    - Deny\n",
			key:     "Setting",
			assert: func(t *testing.T, got *schema) {
				if got.Item == nil {
					t.Fatal("the array element schema is absent from the union")
				}
				want := []any{"Allow", "Deny"}
				if fmt.Sprint(got.Item.Enum) != fmt.Sprint(want) {
					t.Errorf("item enum = %v, want %v", got.Item.Enum, want)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			captured := captureWarnings(t)
			merged := unionBranches(t, specWith(test.release), specWith(test.seed))
			got := merged[test.key]
			if got == nil {
				t.Fatalf("%s is absent from the union", test.key)
			}
			test.assert(t, got)
			if !strings.Contains(captured.String(), "com.example.test."+test.key) {
				t.Errorf("the widening went unreported; stderr was %q", captured.String())
			}
		})
	}
}

// TestUnionKeepsAConstraintBothBranchesAgreeOn is the other half of the widening rule: agreement is
// not a disagreement, so nothing is loosened and nothing is reported.
func TestUnionKeepsAConstraintBothBranchesAgreeOn(t *testing.T) {
	captured := captureWarnings(t)
	spec := specWith("- key: Setting\n  type: <integer>\n  presence: required\n  range:\n    min: 1\n    max: 9\n  rangelist:\n  - 1\n  - 9\n")

	merged := unionBranches(t, spec, spec)
	got := merged["Setting"]
	if got == nil {
		t.Fatal("Setting is absent from the union")
	}
	if !got.Required {
		t.Error("required = false, want true")
	}
	if got.Type != "integer" {
		t.Errorf("type = %q, want integer", got.Type)
	}
	if got.Min == nil || *got.Min != 1 || got.Max == nil || *got.Max != 9 {
		t.Errorf("range = %s to %s, want 1 to 9", formatBound(got.Min), formatBound(got.Max))
	}
	if want := fmt.Sprint([]any{int64(1), int64(9)}); fmt.Sprint(got.Enum) != want {
		t.Errorf("enum = %v, want %v", got.Enum, want)
	}
	if captured.Len() != 0 {
		t.Errorf("two branches that agree reported %q", captured.String())
	}
}

// TestUnionKeepsEverySiblingApplePresentsAsRequired pins the reading of upstream that the union rule
// deliberately does not second-guess: several required keys in one dictionary are several real
// requirements, not a flattening of mutually exclusive variants.
//
// The evidence is Apple's own YAML. com.apple.applicationaccess.new declares an ApplicationItem with
// `bundleID` and `appID` both `presence: required` on the release branch and on seed_OS_27_0, and
// com.apple.AssetCache.managed declares a listen range needing both `first` and `last`; 94
// dictionaries in the committed table carry two or more. Where a key really does become one of
// several alternatives, Apple demotes it to `optional` and says so in prose — see
// com.apple.configuration.legacy, where ProfileURL turned optional when ProfileAssetReference
// arrived — which is the case the required conjunction covers.
func TestUnionKeepsEverySiblingApplePresentsAsRequired(t *testing.T) {
	captureWarnings(t)
	spec := declarationWith("- key: Item\n  type: <dictionary>\n  subkeys:\n  - key: bundleID\n    type: <string>\n    presence: required\n  - key: appID\n    type: <data>\n    presence: required\n  - key: disabled\n    type: <boolean>\n    presence: optional\n")

	parsed, err := parseDeclarationSpec([]byte(spec))
	if err != nil {
		t.Fatalf("parseDeclarationSpec: %v", err)
	}
	keys := dictionaryKeys(parsed.payloadKeys, 1)
	item := unionSchema(nil, keys["Item"], "release", "com.apple.configuration.test.Item")
	item = unionSchema(item, keys["Item"], "seed", "com.apple.configuration.test.Item")

	for _, name := range []string{"bundleID", "appID"} {
		if sub := item.Keys[name]; sub == nil || !sub.Required {
			t.Errorf("%s required = %v, want true", name, sub != nil && sub.Required)
		}
	}
	if sub := item.Keys["disabled"]; sub == nil || sub.Required {
		t.Error("disabled required = true, want false")
	}
}

// TestUnionRecordsEveryBranchThatDeclaresAKey covers what the Refs stamp is for: a diagnostic tells
// a key Apple has always published from one that exists only on a pre-release branch, and the
// stamping has to reach nested keys or the second branch read makes a long-standing key look new.
func TestUnionRecordsEveryBranchThatDeclaresAKey(t *testing.T) {
	captureWarnings(t)
	release := specWith("- key: Outer\n  type: <dictionary>\n  subkeys:\n  - key: Inner\n    type: <string>\n")
	seed := specWith("- key: Outer\n  type: <dictionary>\n  subkeys:\n  - key: Inner\n    type: <string>\n  - key: Added\n    type: <string>\n- key: New\n  type: <string>\n")

	merged := unionBranches(t, release, seed)

	tests := map[string]struct {
		schema   *schema
		wantRefs string
	}{
		"Outer":       {merged["Outer"], "[release seed]"},
		"Outer.Inner": {merged["Outer"].Keys["Inner"], "[release seed]"},
		"Outer.Added": {merged["Outer"].Keys["Added"], "[seed]"},
		"New":         {merged["New"], "[seed]"},
	}
	for path, test := range tests {
		if test.schema == nil {
			t.Errorf("%s is absent from the union", path)
			continue
		}
		if got := fmt.Sprint(test.schema.Refs); got != test.wantRefs {
			t.Errorf("%s refs = %s, want %s", path, got, test.wantRefs)
		}
	}
}

// TestParseKeysBoundsAnAliasGraph covers the cost of upstream's anchors. visiting cuts a sequence
// that reaches itself, but a sequence aliased twice from different points is a directed acyclic
// graph, and expanding one doubles per level. Measured on the 25 levels below, 3.9 KB of YAML: 10 ms
// and 6 MB allocated with the walk bounded, 34 s and 23 GB without it.
func TestParseKeysBoundsAnAliasGraph(t *testing.T) {
	var spec strings.Builder
	spec.WriteString("- key: k0\n  type: <dictionary>\n  subkeys: &l0\n  - key: leaf\n    type: <string>\n")
	for level := 1; level <= 25; level++ {
		fmt.Fprintf(&spec, "- key: k%d\n  type: <dictionary>\n  subkeys: &l%d\n", level, level)
		for _, name := range []string{"a", "b"} {
			fmt.Fprintf(&spec, "  - key: %s\n    type: <dictionary>\n    subkeys: *l%d\n", name, level-1)
		}
	}

	raw := []byte(specWith(spec.String()))
	if len(raw) > 8000 {
		t.Fatalf("the crafted input is %d bytes, which is no longer the small-input case under test", len(raw))
	}

	done := make(chan int, 1)
	go func() {
		parsed, err := parseSpec(raw)
		if err != nil {
			done <- -1
			return
		}
		done <- len(parsed.PayloadKeys)
	}()

	select {
	case count := <-done:
		if count != 26 {
			t.Fatalf("parsed %d top-level keys, want 26", count)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("parseSpec did not finish: an alias graph is being expanded rather than bounded")
	}
}
