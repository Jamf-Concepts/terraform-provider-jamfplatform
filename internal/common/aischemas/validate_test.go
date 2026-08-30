// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package aischemas

import (
	"encoding/json"
	"strings"
	"testing"
)

// parse builds a Document from a schema literal, failing the test if it cannot.
func parse(t *testing.T, schema string) *Document {
	t.Helper()
	document, err := Parse("com.example.tool", "2026-01-01", json.RawMessage(schema))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return document
}

// settings decodes a settings literal the way the resource does.
func settings(t *testing.T, payload string) map[string]any {
	t.Helper()
	var decoded map[string]any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("settings: %v", err)
	}
	return decoded
}

// check asserts the exact set of (kind, path) pairs a validation produces.
func check(t *testing.T, schema, payload string, want []Problem) {
	t.Helper()
	got := parse(t, schema).Validate(settings(t, payload))
	if len(got) != len(want) {
		t.Fatalf("got %d problems, want %d:\n%s", len(got), len(want), render(got))
	}
	for i := range want {
		if got[i].Kind != want[i].Kind || got[i].Path != want[i].Path {
			t.Errorf("problem %d = kind %d at %q, want kind %d at %q (detail: %s)", i, got[i].Kind, got[i].Path, want[i].Kind, want[i].Path, got[i].Detail)
		}
	}
}

func render(problems []Problem) string {
	lines := make([]string, 0, len(problems))
	for _, problem := range problems {
		lines = append(lines, "  kind "+string(rune('0'+problem.Kind))+" at "+problem.Path+": "+problem.Detail)
	}
	return strings.Join(lines, "\n")
}

// TestWrongTypeMatchesServiceFieldPath pins the case wire probing produced: the service answered
// SCHEMA_VALIDATION_FAILED with field "/verbose" for a boolean setting given a string, and the
// plan-time finding must point at the same place so the two read as one problem.
func TestWrongTypeMatchesServiceFieldPath(t *testing.T) {
	check(t,
		`{"type":"object","properties":{"verbose":{"type":"boolean"}},"additionalProperties":true}`,
		`{"verbose":"yes"}`,
		[]Problem{{Kind: WrongType, Path: "/verbose"}})
}

// TestUndeclaredKeyIsAdvisoryOnlyWhenTheSchemaIsOpen pins the asymmetry the two products exhibit:
// com.anthropic.claudecode declares additionalProperties true and stores an unknown key silently,
// com.openai.codex declares it false and the service rejects the write.
func TestUndeclaredKeyIsAdvisoryOnlyWhenTheSchemaIsOpen(t *testing.T) {
	const payload = `{"model":"sonnet","totallyMadeUpKey":123}`

	open := parse(t, `{"type":"object","properties":{"model":{"type":"string"}},"additionalProperties":true}`).Validate(settings(t, payload))
	if len(open) != 1 || open[0].Kind != UnrecognisedKey || !open[0].Advisory() {
		t.Fatalf("open schema: %s", render(open))
	}

	closed := parse(t, `{"type":"object","properties":{"model":{"type":"string"}},"additionalProperties":false}`).Validate(settings(t, payload))
	if len(closed) != 1 || closed[0].Kind != UndeclaredKey || closed[0].Advisory() {
		t.Fatalf("closed schema: %s", render(closed))
	}
}

// TestAbsentAdditionalPropertiesIsOpen pins draft-07's default: com.anthropic.claudefordesktop omits
// the keyword entirely, which means extra keys are allowed.
func TestAbsentAdditionalPropertiesIsOpen(t *testing.T) {
	check(t,
		`{"type":"object","properties":{"secureVmFeaturesEnabled":{"type":"boolean"}}}`,
		`{"secureVmFeaturesEnabled":true,"nope":1}`,
		[]Problem{{Kind: UnrecognisedKey, Path: "/nope"}})
}

// TestFreeFormObjectIsNotJudged pins the rule that keeps a passthrough sub-object passthrough: a node
// declaring no properties constrains nothing, so its keys are not reported.
func TestFreeFormObjectIsNotJudged(t *testing.T) {
	check(t,
		`{"type":"object","properties":{"env":{"type":"object"}},"additionalProperties":false}`,
		`{"env":{"ANYTHING":"goes","nested":{"too":1}}}`,
		nil)
}

// TestAdditionalPropertiesSchemaValidatesMapValues pins the map-typed object form (Claude Code's
// `env`): the keys are free but the values are constrained.
func TestAdditionalPropertiesSchemaValidatesMapValues(t *testing.T) {
	check(t,
		`{"type":"object","properties":{"env":{"type":"object","additionalProperties":{"type":"string"}}},"additionalProperties":false}`,
		`{"env":{"A":"1","B":2}}`,
		[]Problem{{Kind: WrongType, Path: "/env/B"}})
}

// TestAllOfRefWrapperResolves pins the shape all 204 Codex allOf sites take: a single-element allOf
// wrapping a $ref, used to attach a description to a referenced definition.
func TestAllOfRefWrapperResolves(t *testing.T) {
	const schema = `{
	  "type":"object",
	  "additionalProperties":false,
	  "properties":{
	    "config":{
	      "type":"object",
	      "additionalProperties":false,
	      "definitions":{"ApprovalPolicy":{"type":"string","enum":["never","on-request"]}},
	      "properties":{
	        "approval_policy":{"allOf":[{"$ref":"#/properties/config/definitions/ApprovalPolicy"}],"description":"how approvals work"}
	      }
	    }
	  }
	}`
	check(t, schema, `{"config":{"approval_policy":"on-request"}}`, nil)
	check(t, schema, `{"config":{"approval_policy":"whenever"}}`, []Problem{{Kind: NotInEnum, Path: "/config/approval_policy"}})
}

// TestDeepPointerRefResolves pins that references are resolved as general JSON pointers. Codex points
// at #/properties/config/definitions/X, so a $defs-only lookup would resolve nothing for it.
func TestDeepPointerRefResolves(t *testing.T) {
	check(t,
		`{"type":"object","$defs":{"outer":{"inner":{"type":"integer"}}},"properties":{"n":{"$ref":"#/$defs/outer/inner"}},"additionalProperties":false}`,
		`{"n":"x"}`,
		[]Problem{{Kind: WrongType, Path: "/n"}})
}

// TestUnresolvableRefIsIgnored pins that a reference the provider cannot follow — a remote one, or a
// pointer into a schema shape it does not expect — constrains nothing rather than failing the plan.
func TestUnresolvableRefIsIgnored(t *testing.T) {
	check(t, `{"type":"object","properties":{"a":{"$ref":"https://example.com/x.json"},"b":{"$ref":"#/nope/missing"}},"additionalProperties":false}`,
		`{"a":1,"b":"anything"}`, nil)
}

// TestRecursiveRefTerminates pins the cycle guard. The Codex schema is recursive, and a $ref cycle
// consumes no level of the value, so nothing but the visited set stops it.
func TestRecursiveRefTerminates(t *testing.T) {
	done := make(chan []Problem, 1)
	go func() {
		done <- parse(t, `{"$defs":{"loop":{"allOf":[{"$ref":"#/$defs/loop"}],"type":"object","properties":{"child":{"$ref":"#/$defs/loop"}},"additionalProperties":false}},"$ref":"#/$defs/loop"}`).
			Validate(settings(t, `{"child":{"child":{"child":{}}}}`))
	}()
	select {
	case got := <-done:
		if len(got) != 0 {
			t.Fatalf("unexpected problems: %s", render(got))
		}
	case <-timeout():
		t.Fatal("validation did not terminate on a recursive $ref")
	}
}

// TestAnyOfUnionAcceptsAMatchingBranch pins Claude Code's hookCommand shape: four object forms with
// different required keys, of which one must match.
func TestAnyOfUnionAcceptsAMatchingBranch(t *testing.T) {
	const schema = `{
	  "type":"object","additionalProperties":false,
	  "properties":{"hook":{"anyOf":[
	    {"type":"object","required":["type","command"],"additionalProperties":false,"properties":{"type":{"const":"command"},"command":{"type":"string"}}},
	    {"type":"object","required":["type","prompt"],"additionalProperties":false,"properties":{"type":{"const":"prompt"},"prompt":{"type":"string"}}}
	  ]}}
	}`
	check(t, schema, `{"hook":{"type":"command","command":"make test"}}`, nil)
	check(t, schema, `{"hook":{"type":"prompt","prompt":"check it"}}`, nil)
	check(t, schema, `{"hook":{"type":"command"}}`, []Problem{{Kind: NoBranchMatches, Path: "/hook"}})
}

// TestAnyOfReportsTheClosestBranch pins that a union failure quotes one branch rather than every
// branch's contradictory demands.
func TestAnyOfReportsTheClosestBranch(t *testing.T) {
	got := parse(t, `{"type":"object","additionalProperties":false,"properties":{"x":{"anyOf":[
	    {"type":"object","required":["a","b","c"],"properties":{"a":{"type":"string"},"b":{"type":"string"},"c":{"type":"string"}}},
	    {"type":"object","required":["z"],"properties":{"z":{"type":"string"}}}
	  ]}}}`).Validate(settings(t, `{"x":{}}`))

	if len(got) != 1 || got[0].Kind != NoBranchMatches {
		t.Fatalf("got %s", render(got))
	}
	if !strings.Contains(got[0].Detail, `"z"`) {
		t.Errorf("detail should quote the closest branch's single missing key, got: %s", got[0].Detail)
	}
	if !strings.Contains(got[0].Detail, "2 accepted shapes") {
		t.Errorf("detail should count the branches, got: %s", got[0].Detail)
	}
}

// TestAdvisoryFindingDoesNotDisqualifyABranch pins that an open branch carrying an unrecognised key
// still counts as matching, so a union does not turn a warning into an error.
func TestAdvisoryFindingDoesNotDisqualifyABranch(t *testing.T) {
	check(t, `{"type":"object","additionalProperties":false,"properties":{"x":{"anyOf":[
	    {"type":"object","additionalProperties":true,"properties":{"a":{"type":"string"}}},
	    {"type":"string"}
	  ]}}}`, `{"x":{"a":"ok","extra":1}}`, nil)
}

func TestMissingRequiredKey(t *testing.T) {
	check(t,
		`{"type":"object","required":["model","tier"],"properties":{"model":{"type":"string"},"tier":{"type":"string"}},"additionalProperties":false}`,
		`{"model":"sonnet"}`,
		[]Problem{{Kind: MissingRequiredKey, Path: ""}})
}

// TestRequiredKeyFromAllOfMemberCounts pins that composition contributes required keys as well as
// properties.
func TestRequiredKeyFromAllOfMemberCounts(t *testing.T) {
	check(t,
		`{"allOf":[{"required":["tier"]}],"type":"object","properties":{"tier":{"type":"string"}},"additionalProperties":false}`,
		`{}`,
		[]Problem{{Kind: MissingRequiredKey, Path: ""}})
}

// TestPropertyDeclaredByAnAllOfMemberIsNotUndeclared pins the reason object keys are judged against
// the union of composed shapes rather than per node.
func TestPropertyDeclaredByAnAllOfMemberIsNotUndeclared(t *testing.T) {
	check(t,
		`{"allOf":[{"properties":{"extra":{"type":"integer"}}}],"type":"object","properties":{"model":{"type":"string"}},"additionalProperties":false}`,
		`{"model":"sonnet","extra":1}`,
		nil)
}

func TestNumericAndLengthBounds(t *testing.T) {
	check(t,
		`{"type":"object","additionalProperties":false,"properties":{
		  "hours":{"type":"integer","minimum":1,"maximum":72},
		  "port":{"type":"integer","exclusiveMinimum":0},
		  "name":{"type":"string","minLength":2,"maxLength":4}
		}}`,
		`{"hours":99,"port":0,"name":"x"}`,
		[]Problem{
			{Kind: OutOfRange, Path: "/hours"},
			{Kind: LengthOutOfRange, Path: "/name"},
			{Kind: OutOfRange, Path: "/port"},
		})
}

// TestLengthCountsRunesNotBytes pins that a multi-byte string is measured the way an author counts it.
func TestLengthCountsRunesNotBytes(t *testing.T) {
	check(t, `{"type":"object","additionalProperties":false,"properties":{"s":{"type":"string","maxLength":3}}}`,
		`{"s":"héé"}`, nil)
}

func TestPatternAndItemRules(t *testing.T) {
	check(t,
		`{"type":"object","additionalProperties":false,"properties":{
		  "key":{"type":"string","pattern":"^[A-Z_][A-Z0-9_]*$"},
		  "tags":{"type":"array","items":{"type":"string"},"uniqueItems":true,"minItems":2}
		}}`,
		`{"key":"lower","tags":["a","a"]}`,
		[]Problem{
			{Kind: PatternMismatch, Path: "/key"},
			{Kind: DuplicateItems, Path: "/tags"},
		})
}

// TestUncompilablePatternIsSkipped pins that an ECMA-262 construct RE2 cannot express is passed over
// rather than reported. Go's regexp rejects lookahead.
func TestUncompilablePatternIsSkipped(t *testing.T) {
	check(t, `{"type":"object","additionalProperties":false,"properties":{"s":{"type":"string","pattern":"^(?=x)y$"}}}`,
		`{"s":"anything"}`, nil)
}

func TestTupleItems(t *testing.T) {
	check(t,
		`{"type":"object","additionalProperties":false,"properties":{"pair":{"type":"array","items":[{"type":"string"},{"type":"integer"}]}}}`,
		`{"pair":["a","b"]}`,
		[]Problem{{Kind: WrongType, Path: "/pair/1"}})
}

func TestPropertyNamesConstraint(t *testing.T) {
	check(t,
		`{"type":"object","additionalProperties":false,"properties":{"env":{"type":"object","propertyNames":{"pattern":"^[A-Z_]+$"},"additionalProperties":{"type":"string"}}}}`,
		`{"env":{"OK":"1","not ok":"2"}}`,
		[]Problem{{Kind: InvalidPropertyName, Path: "/env/not ok"}})
}

// TestIntegerAcceptsWholeValuedNumber pins that JSON's single number type does not make an integer
// setting unwritable — 3 decodes to float64(3) and must satisfy "integer".
func TestIntegerAcceptsWholeValuedNumber(t *testing.T) {
	check(t, `{"type":"object","additionalProperties":false,"properties":{"n":{"type":"integer"}}}`, `{"n":3}`, nil)
	check(t, `{"type":"object","additionalProperties":false,"properties":{"n":{"type":"integer"}}}`, `{"n":3.5}`,
		[]Problem{{Kind: WrongType, Path: "/n"}})
}

func TestTypeUnion(t *testing.T) {
	const schema = `{"type":"object","additionalProperties":false,"properties":{"v":{"type":["string","null"]}}}`
	check(t, schema, `{"v":null}`, nil)
	check(t, schema, `{"v":"x"}`, nil)
	check(t, schema, `{"v":1}`, []Problem{{Kind: WrongType, Path: "/v"}})
}

// TestWrongTypeStopsDescent pins that a value of the wrong shape produces one finding, not one per
// key of the shape that was expected.
func TestWrongTypeStopsDescent(t *testing.T) {
	check(t,
		`{"type":"object","additionalProperties":false,"properties":{"nested":{"type":"object","required":["a"],"properties":{"a":{"type":"string"}},"additionalProperties":false}}}`,
		`{"nested":"not an object"}`,
		[]Problem{{Kind: WrongType, Path: "/nested"}})
}

// TestPointerEscaping pins RFC 6901 escaping, so a key containing a slash still produces a pointer
// that names it unambiguously.
func TestPointerEscaping(t *testing.T) {
	got := parse(t, `{"type":"object","additionalProperties":false,"properties":{"a/b":{"type":"string"},"c~d":{"type":"string"}}}`).
		Validate(settings(t, `{"a/b":1,"c~d":2}`))
	if len(got) != 2 {
		t.Fatalf("got %s", render(got))
	}
	if got[0].Path != "/a~1b" || got[1].Path != "/c~0d" {
		t.Errorf("paths = %q, %q; want /a~1b, /c~0d", got[0].Path, got[1].Path)
	}
}

// TestNilDocumentAcceptsEverything pins the degrade-to-silent rule: a schema the provider could not
// fetch must not produce findings.
func TestNilDocumentAcceptsEverything(t *testing.T) {
	var document *Document
	if got := document.Validate(map[string]any{"anything": 1}); got != nil {
		t.Errorf("nil document returned %s", render(got))
	}
	if got := parse(t, `true`).Validate(map[string]any{"anything": 1}); got != nil {
		t.Errorf("boolean schema returned %s", render(got))
	}
}

func TestParseRejectsMalformedSchema(t *testing.T) {
	if _, err := Parse("t", "v", json.RawMessage(`{"type":`)); err == nil {
		t.Fatal("expected an error for a malformed schema")
	}
}

func TestDeterministicOrder(t *testing.T) {
	const schema = `{"type":"object","additionalProperties":false,"properties":{"a":{"type":"string"},"b":{"type":"string"},"c":{"type":"string"}}}`
	const payload = `{"c":1,"a":2,"b":3}`
	first := parse(t, schema).Validate(settings(t, payload))
	for range 20 {
		next := parse(t, schema).Validate(settings(t, payload))
		for i := range first {
			if first[i].Path != next[i].Path {
				t.Fatalf("order changed: %q then %q", first[i].Path, next[i].Path)
			}
		}
	}
	if len(first) != 3 || first[0].Path != "/a" || first[2].Path != "/c" {
		t.Errorf("want sorted key order, got %s", render(first))
	}
}
